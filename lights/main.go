package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const BridgeAddress = "10.0.1.40"

var BridgeUsername string

// WARN: remove
func init() {
	BridgeUsername = os.Getenv("HUE_USERNAME")
	if BridgeUsername == "" {
		panic("missing env var: HUE_USERNAME")
	}
}

type StatusCodeError struct {
	Code int
	URL  string
}

func (t *StatusCodeError) Error() string {
	return fmt.Sprintf("HUE server error (%s): %d - %s", t.URL, t.Code,
		http.StatusText(t.Code))
}

func (t *StatusCodeError) HTTPStatusCode() int {
	return t.Code
}

type Client struct {
	Address  string
	Username string
	client   *http.Client
	limit    *RateLimit
}

type Option interface {
	apply(c *Client)
}

type optRateLimit struct {
	interval time.Duration
}

func (o *optRateLimit) apply(c *Client) {
	c.limit = NewRateLimit(o.interval)
}

func WithRateLimit(interval time.Duration) Option {
	return &optRateLimit{interval}
}

type optClient struct {
	client *http.Client
}

func (o *optClient) apply(c *Client) {
	c.client = o.client
}

func WithClient(c *http.Client) Option {
	return nil
}

func (c *Client) initDefaults() {
	if c.client == nil {
		c.client = &http.Client{Timeout: time.Second * 3}
	}
	if c.limit == nil {
		c.limit = NewRateLimit(time.Second / 25)
	}
}

func NewClient(address, username string, opts ...Option) *Client {
	c := &Client{
		Address:  address,
		Username: username,
	}
	for _, o := range opts {
		o.apply(c)
	}
	c.initDefaults()
	return c
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

func (c *Client) decodeGet(res *http.Response, dst interface{}) error {
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, dst); err != nil {
		// TODO: do we actually need this???
		var herr []internalRequestError
		if e := json.Unmarshal(b, &herr); e == nil && len(herr) != 0 {
			return herr[0].Error
		}
		return err
	}
	return nil
}

func (c *Client) url(endpoint string) string {
	u := url.URL{
		Scheme: "http",
		Host:   c.Address,
		Path:   path.Join("api", c.Username, endpoint),
	}
	return u.String()
}

func (c *Client) Do(method, endpoint string, body io.Reader) (*http.Response, error) {
	url := c.url(endpoint)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	c.limit.Wait()
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, &StatusCodeError{res.StatusCode, url}
	}
	return res, nil
}

func closeResponse(res *http.Response) {
	io.Copy(ioutil.Discard, res.Body)
	res.Body.Close()
}

func (c *Client) Get(endpoint string, v interface{}) error {
	res, err := c.Do("GET", endpoint, nil)
	if err != nil {
		return err
	}
	defer closeResponse(res)
	return c.decodeGet(res, v)
}

func (c *Client) Put(endpoint string, request, response interface{}) error {
	var body io.Reader
	if request != nil {
		b, err := json.Marshal(request)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	res, err := c.Do("PUT", endpoint, body)
	if err != nil {
		return err
	}
	defer closeResponse(res)

	if response != nil {
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &response); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) GetGroups() ([]Group, error) {
	var m map[string]Group
	if err := c.Get("groups", &m); err != nil {
		return nil, err
	}

	groups := make([]Group, 0, len(m))
	for id, g := range m {
		n, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}
		g.ID = n
		groups = append(groups, g)
	}
	return groups, nil
}

func (c *Client) UpdateLight(id int, state *LightStateRequest) (*Light, error) {
	panic("IMPLEMENT")
}

func (c *Client) GetLight(id int) (*Light, error) {
	var light Light
	if err := c.Get("light", &light); err != nil {
		return nil, err
	}
	light.ID = id
	return &light, nil
}

func (c *Client) GetLights() ([]Light, error) {
	var m map[string]Light
	if err := c.Get("lights", &m); err != nil {
		return nil, err
	}

	lights := make([]Light, 0, len(m))
	for id, l := range m {
		n, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}
		l.ID = n
		lights = append(lights, l)
	}
	return lights, nil
}

/*
func (c *Client) UpdateLights(fn func(id string, light Light) error) error {
	lights, err := c.GetLights()
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
*/

type RateLimit struct {
	mu   sync.Mutex
	d    time.Duration
	last time.Time
}

func NewRateLimit(d time.Duration) *RateLimit {
	return &RateLimit{d: d}
}

func (r *RateLimit) Wait() {
	if r == nil {
		return
	}
	r.mu.Lock()

	now := time.Now()
	d := r.d - now.Sub(r.last)
	if time.Millisecond < d && d < r.d {
		time.Sleep(d)
		now = time.Now()
	}
	r.last = now

	r.mu.Unlock()
	return
}

const PhueTimeFormat = "2006-01-02T15:04:05"

type PhueTime time.Time // TODO: rename to Time

func (p PhueTime) Time() time.Time { return time.Time(p) }

func (p PhueTime) MarshalJSON() ([]byte, error) {
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
	{
		fmt.Println("MaxInt8:", math.MaxInt8)
		fmt.Println("MinInt8:", math.MinInt8)
		fmt.Println("MaxInt16:", math.MaxInt16)
		fmt.Println("MinInt16:", math.MinInt16)
		fmt.Println("MaxInt32:", math.MaxInt32)
		fmt.Println("MinInt32:", math.MinInt32)
		fmt.Println("MaxInt64:", math.MaxInt64)
		fmt.Println("MinInt64:", math.MinInt64)
		fmt.Println("MaxUint8:", math.MaxUint8)
		fmt.Println("MaxUint16:", math.MaxUint16)
		fmt.Println("MaxUint32:", math.MaxUint32)
		// fmt.Println("MaxUint64:", math.MaxUint64)
		return
		// on := true
		// r := &GroupStateRequest{
		// 	On: &on,
		// }
	}

	// {
	// 	fmt.Println(time.Parse("2006-01-02T15:04:05", "2018-12-14T19:31:33"))
	// 	return
	// }
	c := NewClient(BridgeAddress, BridgeUsername)
	lights, err := c.GetLights()
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
