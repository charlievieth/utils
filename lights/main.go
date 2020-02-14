package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"math"
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

func XYToColor(x, y float64) color.RGBA {
	const Y = 1.0
	z := 1.0 - x - y // TODO: move to Z
	X := (Y / y) * x
	Z := (Y / y) * z

	// Convert to RGB using Wide RGB D65 conversion
	r := X*1.656492 - Y*0.354851 - Z*0.255038
	g := -X*0.707196 + Y*1.655397 + Z*0.036152
	b := X*0.051713 - Y*0.121364 + Z*1.011530

	switch {
	case r > b && r > g && r > 1.0:
		// red is too big
		g /= r
		b /= r
		r = 1.0
	case g > b && g > r && g > 1.0:
		// green is too big
		r /= g
		b /= g
		g = 1.0
	case b > r && b > g && b > 1.0:
		// blue is too big
		r /= b
		g /= b
		b = 1.0
	}

	// TODO: make sure this is correct
	//
	// Apply reverse gamma correction
	if r <= 0.0031308 {
		r = 12.92 * r
	} else {
		r = (1.0+0.055)*math.Pow(r, (1.0/2.4)) - 0.055
	}
	if g <= 0.0031308 {
		g = 12.92 * g
	} else {
		g = (1.0+0.055)*math.Pow(g, (1.0/2.4)) - 0.055
	}
	if b <= 0.0031308 {
		b = 12.92 * b
	} else {
		b = (1.0+0.055)*math.Pow(b, (1.0/2.4)) - 0.055
	}

	switch {
	case r > b && r > g:
		// red is biggest
		if r > 1.0 {
			g /= r
			b /= r
			r = 1.0
		}
	case g > b && g > r:
		// green is biggest
		if g > 1.0 {
			r /= g
			b /= g
			g = 1.0
		}
	case b > r && b > g:
		// blue is biggest
		if b > 1.0 {
			r /= b
			g /= b
			b = 1.0
		}
	}

	fmt.Println("R:", r, uint8(r*math.MaxUint8))
	fmt.Println("G:", g, uint8(g*math.MaxUint8))
	fmt.Println("B:", b, uint8(b*math.MaxUint8))

	return color.RGBA{
		R: uint8(r * math.MaxUint8),
		G: uint8(g * math.MaxUint8),
		B: uint8(b * math.MaxUint8),
	}
}

func (x XY) RGB(brightness uint8) color.RGBA {
	// https://developers.meethue.com/develop/application-design-guidance/color-conversion-formulas-rgb-to-xy-and-back/
	//
	// Calculate XYZ values
	xX := float64(x.X)
	xY := float64(x.Y)
	z := 1.0 - xX - xY
	Y := float64(brightness) / 254.0
	fmt.Printf("X: %f Y: %f\n", x.X, x.Y)
	fmt.Println("BRIGHTNESS:", Y, brightness)
	Y = 0.75
	X := (Y / xY) * xX
	Z := (Y / xY) * z

	// Convert to RGB using Wide RGB D65 conversion
	r := X*1.656492 - Y*0.354851 - Z*0.255038
	g := -X*0.707196 + Y*1.655397 + Z*0.036152
	b := X*0.051713 - Y*0.121364 + Z*1.011530

	// WARN: something here is broken the follow code fails
	// to bound R to 0..1
	//
	// Apply reverse gamma correction
	if r <= 0.0031308 {
		r = 12.92 * r
	} else {
		r = (1.0+0.055)*math.Pow(r, (1.0/2.4)) - 0.055
	}
	if g <= 0.0031308 {
		g = 12.92 * g
	} else {
		g = (1.0+0.055)*math.Pow(g, (1.0/2.4)) - 0.055
	}
	if b <= 0.0031308 {
		b = 12.92 * b
	} else {
		b = (1.0+0.055)*math.Pow(b, (1.0/2.4)) - 0.055
	}
	fmt.Printf("XY: R: %f G: %f B: %f\n", r, g, b)

	// WARN: we should not need this!!!
	clamp := func(f float64) uint8 {
		if f >= 1 {
			f = 1
		}
		if f <= 0 {
			f = 0
		}
		return uint8(f * 254.0)
	}
	return color.RGBA{
		R: clamp(r),
		G: clamp(g),
		B: clamp(b),
	}
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
	var gamut [3][2]float32
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
	gamut := [3][2]float32{
		0: {c.Red.X, c.Red.Y},
		1: {c.Green.X, c.Green.Y},
		2: {c.Blue.X, c.Blue.Y},
	}
	return json.Marshal(gamut)
}

type State struct {
	On               bool      `json:"on"`
	Reachable        bool      `json:"reachable"`
	Brightness       uint8     `json:"bri"`
	Saturation       *uint8    `json:"sat,omitempty"`
	Hue              *uint16   `json:"hue,omitempty"`
	ColorTemperature uint16    `json:"ct"`
	Alert            string    `json:"alert"`
	Effect           string    `json:"effect,omitempty"`
	ColorMode        ColorMode `json:"colormode"`
	XY               *XY       `json:"xy,omitempty"`
}

type SoftwareUpdate struct {
	LastInstall PhueTime `json:"lastinstall"`
	State       string   `json:"state"`
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

// TODO: might wanna add "omitempty" tag
type Light struct {
	Type              string         `json:"type"`
	Name              string         `json:"name"`
	ProductName       string         `json:"productname"`
	ManufacturerName  string         `json:"manufacturername"`
	ModelID           string         `json:"modelid"`
	UniqueID          string         `json:"uniqueid"`
	LuminaireUniqueID string         `json:"luminaireuniqueid"`
	SoftwareVersion   string         `json:"swversion"`
	Streaming         Streaming      `json:"streaming"`
	Capabilities      Capabilities   `json:"capabilities"`
	Config            LightConfig    `json:"config"`
	State             State          `json:"state"`
	SoftwareUpdate    SoftwareUpdate `json:"swupdate"`
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

const PhueTimeFormat = "2006-01-02T15:04:05"

type PhueTime time.Time

func (p PhueTime) Time() time.Time { return time.Time(p) }

func (p *PhueTime) MarshalJSON() ([]byte, error) {
	t := p.Time()
	if y := t.Year(); y < 0 || y >= 10000 {
		return nil, errors.New("PhueTime.MarshalJSON: year outside of range [0,9999]")
	}
	b := make([]byte, 0, len(PhueTimeFormat)+2)
	b = append(b, '"')
	b = t.AppendFormat(b, PhueTimeFormat)
	b = append(b, '"')
	return b, nil
}

func (p *PhueTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	t, err := time.Parse(`"`+PhueTimeFormat+`"`, string(data))
	if err != nil {
		return err
	}
	*p = PhueTime(t)
	return nil
}

func main() {
	// {
	// 	fmt.Println(time.Parse("2006-01-02T15:04:05", "2018-12-14T19:31:33"))
	// 	return
	// }
	c := NewClient(BridgeAddress, BridgeUsername)
	lights, err := c.Lights()
	if err != nil {
		Fatal(err)
	}
	for _, l := range lights {
		if l.State.XY != nil {
			c := XYToColor(float64(l.State.XY.X), float64(l.State.XY.Y))

			// c := l.State.XY.RGB(l.State.Brightness)
			// _ = c

			fmt.Printf("R: %d G: %d B: %d\n", c.R, c.G, c.B)
			fmt.Printf("\033[38;2;%d;%d;%dmCOLOR: %s\033[0m\n", c.R, c.G, c.B, l.Name)
			// // printf "\x1b[38;2;255;100;0mTRUECOLOR\x1b[0m\n"
			// // l.State.XY.RGB(l.State.Brightness)
			// // colors = append(colors, )
		}
	}

	// PrintJSON(colors)
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

/*
func unquote(b []byte) []byte {
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return b
	}
	return b[1 : len(b)-1]
}
*/
