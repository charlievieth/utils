package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
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

func main() {
	const (
		SID          = "REPLACE_ME"
		Token        = "REPLACE_ME"
		LogFile      = "log.json"
		BaseInterval = time.Minute * 15
	)
	f, err := os.OpenFile(LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)

	client := gotwilio.NewTwilioClient(SID, Token)
	first := true
	for {
		var fact string
		if first {
			fact = "DO YOU NOT ENJOY YOUR CAT FACTS!!!"
			first = false
			log.Printf("fact (first): %s\n", fact)
		} else {
			n := rand.Intn(len(CatFacts))
			fact = CatFacts[n]
			log.Printf("fact (%d): %s\n", n, fact)
		}

		for _, number := range Numbers {
			res, exc, err := client.SendSMS("REPLACE_ME", number, fact, "", "REPLACE_ME")
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
		}
		dur := time.Second * time.Duration(rand.Intn(120))

		log.Printf("sleep: %s\n", BaseInterval+dur)
		time.Sleep(BaseInterval + dur)
	}

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
