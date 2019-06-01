package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stderr)
	log.SetPrefix("[error output] ")
}

type Context struct {
	done  chan struct{}
	sig   chan struct{}
	count int64
	iwg   sync.WaitGroup // init wait group
	cwg   sync.WaitGroup // call wait group
	cond  sync.Cond
}

func (c *Context) Inc() {
	atomic.AddInt64(&c.count, 1)
}

func Call10(ctxt *Context) {
	defer ctxt.cwg.Done()
	select {
	case <-ctxt.done:
		fmt.Fprintln(os.Stderr, "Call100: aborted")
		return
	default:
		ctxt.iwg.Done()
	}
	// tick := time.NewTicker(time.Microsecond)
	for {
		select {
		case <-ctxt.done:
			fmt.Fprintln(os.Stderr, "Call100: aborted")
			return
		case <-ctxt.sig:
			panic("HERE")
			// case <-tick.C:
			// ok
		}
	}
}

func CreateFuncs() {
	const header = `
package main

import (
	"log"
	"os"
)
`

	const template = `
func Call%[1]d(ctxt *Context) {
	log.Println("Call%[1]d: called")
	select {
	case <-ctxt.done:
		log.Println(os.Stderr, "Call%[1]d: aborted")
		return
	default:
		ctxt.Inc()
		log.Println("Call%[1]d: calling next func")
		Call%d(ctxt)
	}
}
`

	fmt.Println(header)
	for i := 1; i < 10; i++ {
		fmt.Printf(template, i, i+1)
	}
}

func main() {
	const N = 10000
	ctxt := &Context{
		done: make(chan struct{}),
		sig:  make(chan struct{}),
	}
	ctxt.iwg.Add(N)
	ctxt.cwg.Add(N)
	for i := 0; i < N; i++ {
		go Call1(ctxt)
	}
	ctxt.iwg.Wait()
	fmt.Println("NumGoroutine:", runtime.NumGoroutine())
	close(ctxt.sig)
	ctxt.cwg.Wait()
}
