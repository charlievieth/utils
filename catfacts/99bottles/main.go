package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/sfreiberg/gotwilio"
)

type Response struct {
	Time      time.Time
	Response  *gotwilio.SmsResponse
	Exception *gotwilio.Exception
	Error     error
}

func NewResponse(res *gotwilio.SmsResponse, exc *gotwilio.Exception, err error) *Response {
	return &Response{
		Time:      time.Now(),
		Response:  res,
		Exception: exc,
		Error:     err,
	}
}

var Numbers = []string{
	"REPLACE_ME",
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	OutboundNumber = ""
	SID            = ""
	Token          = ""
	LogFile        = "log.json"
	BaseInterval   = time.Minute * 15
)

func main() {
	f, err := os.OpenFile(LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)

	client := gotwilio.NewTwilioClient(SID, Token)
	wg := new(sync.WaitGroup)
	for _, number := range Numbers {
		wg.Add(1)
		go Ping(wg, client, enc, number)
	}
	wg.Wait()
}

var Message = [...]string{
	"Take one down and pass it around, %[1]s bottles of beer on the wall.",
	"%[1]s bottles of beer on the wall, %[1]s bottles of beer.",
}

func Ping(wg *sync.WaitGroup, client *gotwilio.Twilio, enc *json.Encoder, number string) {
	const Start = 999999
	defer wg.Done()

	n := Start
	{
		initialMsg := fmt.Sprintf("Welcome to %d Bottles of Beer on the Wall!", n)
		res, exc, err := client.SendSMS("REPLACE_ME", number, initialMsg, "", SID)
		switch {
		case err != nil:
			log.Printf("(%s) Error: %s\n", number, err)
		case exc != nil:
			log.Printf("(%s) Exception: %+v\n", number, exc)
		default:
			log.Printf("(%s) Success: %+v\n", number, res)
		}
	}

	rr := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i <= Start; i++ {
		s := Message[i%len(Message)]
		msg := fmt.Sprintf(s, Format(strconv.Itoa(n)))
		if s == "%[1]s bottles of beer on the wall, %[1]s bottles of beer." {
			n--
		}
		res, exc, err := client.SendSMS("REPLACE_ME", number, msg, "", SID)
		switch {
		case err != nil:
			log.Printf("(%s) Error: %s\n", number, err)
		case exc != nil:
			log.Printf("(%s) Exception: %+v\n", number, exc)
		default:
			log.Printf("(%s) Success: %+v\n", number, res)
		}
		if err := enc.Encode(NewResponse(res, exc, err)); err != nil {
			Fatal(err)
		}
		dur := time.Second * time.Duration(rr.Intn(600))
		if dur < time.Minute*5 && rr.Intn(100) >= 20 {
			dur += time.Minute * 2
		}
		if dur < time.Second*30 {
			dur = time.Second*15 + (time.Second * time.Duration(rr.Intn(30)))
		}
		time.Sleep(dur)
	}
}

// Format, adds thousands separators to string s.  An error is returned is s
// is not a number.
func Format(s string) string {
	var a [64]byte
	return string(formatNumber(a[:0], []byte(s)))
}

func isNumber(b []byte) bool {
	if len(b) == 0 || b[0] == '.' {
		return false
	}
	for _, c := range b {
		if ('0' > c || c > '9') && c != '.' {
			return false
		}
	}
	return true
}

func formatNumber(dst, b []byte) []byte {
	n := bytes.IndexByte(b, '.')
	if n == -1 {
		n = len(b)
	}
	if n <= 3 {
		return append(dst, b...)
	}
	c := (n % 3)
	if c == 0 {
		c = 3
	}
	dst = append(dst, b[:c]...)
	for i := c; i < n; i += 3 {
		dst = append(dst, ',')
		dst = append(dst, b[i:i+3]...)
	}
	return append(dst, b[n:]...)
}

func Fatal(err interface{}) {
	var s string
	if _, file, line, ok := runtime.Caller(1); ok {
		s = fmt.Sprintf("%s:%d", file, line)
	}
	if err != nil {
		switch err.(type) {
		case error, string:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %s\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			}
		default:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %#v\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %#v\n", err)
			}
		}
		os.Exit(1)
	}
}
