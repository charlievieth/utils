package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const BridgeAddress = "10.0.1.40"

var BridgeUsername string

func init() {
	BridgeUsername = os.Getenv("HUE_USERNAME")
	if BridgeUsername == "" {
		panic("missing env var: HUE_USERNAME")
	}
}

type XY struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

func (x *XY) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var xy [2]float32
	if err := json.Unmarshal(b, &xy); err != nil {
		return err
	}
	*x = XY{X: xy[0], Y: xy[1]}
	return nil
}

func (x XY) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]float32{x.X, x.Y})
}

type ColorGamutType string

const (
	ColorGamutLivingColors ColorGamutType = "A"
	ColorGamutGeneration1  ColorGamutType = "B"
	ColorGamutFull         ColorGamutType = "C"
	ColorGamutOther        ColorGamutType = "other"
)

func (c ColorGamutType) String() string { return string(c) }

func (c ColorGamutType) Type() string {
	switch c {
	case ColorGamutLivingColors:
		return "Living colors & lightstrip v1 gamut"
	case ColorGamutGeneration1:
		return "Hue generation 1 gamut"
	case ColorGamutFull:
		return "Hue full colors gamut"
	case ColorGamutOther:
		return "Other/not properly defined gamuts"
	default:
		return "Invalid Color Gamut: " + string(c)
	}
}

func (c *ColorGamutType) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch ct := ColorGamutType(s); ct {
	case ColorGamutLivingColors, ColorGamutGeneration1, ColorGamutFull:
		*c = ct
	default:
		*c = ColorGamutOther
	}
	return nil
}

type ColorMode string

const (
	ColorModeHueSaturation ColorMode = "hs"
	ColorModeXY            ColorMode = "xy"
	ColorModeTemperature   ColorMode = "ct"
	ColorModeOther         ColorMode = "other"
)

func (c ColorMode) String() string { return string(c) }

func (c *ColorMode) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch cm := ColorMode(s); cm {
	case ColorModeHueSaturation, ColorModeXY, ColorModeTemperature:
		*c = cm
	default:
		*c = ColorModeOther
	}
	return nil
}

type ColorGamut struct {
	Red   XY `json:"red"`   // max X, max Y
	Green XY `json:"green"` // max X, max Y
	Blue  XY `json:"blue"`  // max X, max Y
}

func (c *ColorGamut) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var gamut [3][3]float32
	if err := json.Unmarshal(b, &gamut); err != nil {
		return err
	}
	*c = ColorGamut{
		Red:   XY{X: gamut[0][0], Y: gamut[0][1]},
		Green: XY{X: gamut[1][0], Y: gamut[1][1]},
		Blue:  XY{X: gamut[2][0], Y: gamut[2][1]},
	}
	return nil
}

func (c ColorGamut) MarshalJSON() ([]byte, error) {
	gamut := [3][3]float32{
		0: {c.Red.X, c.Red.Y},
		1: {c.Green.X, c.Green.Y},
		2: {c.Blue.X, c.Blue.Y},
	}
	return json.Marshal(gamut)
}

type State struct {
	On               bool    `json:"on"`
	Reachable        bool    `json:"reachable"`
	Brightness       uint8   `json:"bri"`
	Saturation       *uint8  `json:"sat,omitempty"`
	Hue              *uint16 `json:"hue,omitempty"`
	ColorTemperature uint16  `json:"ct"`
	Alert            string  `json:"alert"`
	Effect           string  `json:"effect,omitempty"`
	ColorMode        string  `json:"colormode"`
	XY               *XY     `json:"xy,omitempty"`
}

type Streaming struct {
	Renderer bool `json:"renderer"`
	Proxy    bool `json:"proxy"`
}

type MinMax struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type Control struct {
	MinDimLevel      int             `json:"mindimlevel"`
	MaxLumen         int             `json:"maxlumen"`
	ColorTemperature MinMax          `json:"ct"`
	ColorGamutType   *ColorGamutType `json:"colorgamuttype,omitempty"`
	ColorGamut       *ColorGamut     `json:"colorgamut,omitempty"`
}

type Capabilities struct {
	Certified bool      `json:"certified"`
	Control   Control   `json:"control"`
	Streaming Streaming `json:"streaming"`
}

type Light struct {
	Type              string       `json:"type"`
	Name              string       `json:"name"`
	ProductName       string       `json:"productname"`
	ManufacturerName  string       `json:"manufacturername"`
	ModelID           string       `json:"modelid"`
	UniqueID          string       `json:"uniqueid"`
	LuminaireUniqueID string       `json:"luminaireuniqueid"`
	SoftwareVersion   string       `json:"swversion"`
	Streaming         Streaming    `json:"streaming"`
	Capabilities      Capabilities `json:"capabilities"`
	Config            LightConfig  `json:"config"`
	State             State        `json:"state"`
}

type LightConfig struct {
	ArcheType string `json:"archetype"`
	Function  string `json:"function"`
	Direction string `json:"direction"`
}

type Lights map[string]Light

type Group struct {
	Name   string   `json:"name"`
	Lights []string `json:"lights"`
	Type   string   `json:"type"`
	State  State    `json:"state"`
}

type statusCodeError struct {
	Code int
	URL  string
}

func (t *statusCodeError) Error() string {
	return fmt.Sprintf("HUE server error (%s): %d - %s", t.URL, t.Code,
		http.StatusText(t.Code))
}

func (t *statusCodeError) HTTPStatusCode() int {
	return t.Code
}

type Client struct {
	Address  string
	Username string
	client   *http.Client
	limit    *RateLimit
}

func NewClient(address, username string) *Client {
	return &Client{
		Address:  address,
		Username: username,
		client:   &http.Client{Timeout: time.Second * 3},
		limit:    NewRateLimit(time.Second / 25),
	}
}

/*
[
  {
    "error": {
      "type": 4,
      "address": "/lightsXX",
      "description": "method, GET, not available for resource, /lightsXX"
    }
  }
]
*/

type internalRequestError struct {
	Error RequestError `json:"error"`
}

type RequestError struct {
	Type        int    `json:"type"`
	Address     string `json:"address"`
	Description string `json:"description"`
}

func (h RequestError) Error() string {
	return fmt.Sprintf("hue error: type: '%d' address: '%s' message: %s",
		h.Type, h.Address, h.Description)
}

func (c *Client) decode(res *http.Response, dst interface{}) error {
	var buf bytes.Buffer
	buf.Grow(bytes.MinRead)
	if _, err := buf.ReadFrom(res.Body); err != nil {
		return err
	}
	if err := json.Unmarshal(buf.Bytes(), dst); err != nil {
		var herr []internalRequestError
		if e := json.Unmarshal(buf.Bytes(), &herr); e == nil && len(herr) != 0 {
			return herr[0].Error
		}
		return err
	}
	return nil
}

func (c *Client) url(endpoint string) string {
	return "http://" + c.Address + "/api/" + c.Username + "/" + endpoint
}

func (c *Client) do(method, endpoint string, body io.Reader) (*http.Response, error) {
	url := c.url(endpoint)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	c.limit.Wait()
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, &statusCodeError{res.StatusCode, url}
	}
	return res, nil
}

func (c *Client) get(endpoint string, v interface{}) error {
	res, err := c.do("GET", "lights", nil)
	if err != nil {
		return err
	}
	return c.decode(res, v)
}

func (c *Client) Lights() (Lights, error) {
	var lights Lights
	return lights, c.get("lights", &lights)
}

func (c *Client) UpdateLights(fn func(id string, light Light) error) error {
	lights, err := c.Lights()
	if err != nil {
		return err
	}
	for id, light := range lights {
		if err := fn(id, light); err != nil {
			return err
		}
	}
	// TODO: update
	return nil
}

type RateLimit struct {
	d    time.Duration
	last time.Time
}

func NewRateLimit(d time.Duration) *RateLimit {
	return &RateLimit{d: d}
}

func (r *RateLimit) Wait() {
	const MinWait = time.Millisecond
	now := time.Now()
	d := r.d - now.Sub(r.last)
	if MinWait < d && d < r.d {
		time.Sleep(d)
		now = time.Now()
	}
	r.last = now
	return
}

func main() {

	c := NewClient(BridgeAddress, BridgeUsername)
	lights, err := c.Lights()
	if err != nil {
		Fatal(err)
	}
	PrintJSON(lights)
	return
}

func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	return enc.Encode(v)
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var s string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		s = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		s = "Error"
	}
	switch err.(type) {
	case error, string, fmt.Stringer:
		fmt.Fprintf(os.Stderr, "%s: %s\n", s, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", s, err)
	}
	os.Exit(1)
}

func DebugResponse(res *http.Response) error {
	var src bytes.Buffer
	if _, err := src.ReadFrom(res.Body); err != nil {
		return err
	}
	var dst bytes.Buffer
	if err := json.Indent(&dst, src.Bytes(), "", "    "); err != nil {
		dst = src // not JSON swap buffers
	}
	dst.WriteByte('\n')
	_, err := dst.WriteTo(os.Stdout)
	return err
}

/*
func (c *Client) debug(endpoint string) error {
	url := fmt.Sprintf("http://%s/api/%s/%s", c.Address, c.Username, endpoint)
	res, err := c.client.Get(url)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		fmt.Errorf("GET (%s): status (%d): %s", url, res.StatusCode, res.Status)
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, b, "", "    "); err != nil {
		return err
	}
	fmt.Println(buf.String())
	return nil
}
*/
