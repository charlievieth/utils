package main

import (
	"log"
	"os"
)

func Call1(ctxt *Context) {
	log.Println("Call1: called")
	select {
	case <-ctxt.done:
		log.Println(os.Stderr, "Call1: aborted")
		return
	default:
		ctxt.Inc()
		log.Println("Call1: calling next func")
		Call2(ctxt)
	}
}

func Call2(ctxt *Context) {
	log.Println("Call2: called")
	select {
	case <-ctxt.done:
		log.Println(os.Stderr, "Call2: aborted")
		return
	default:
		ctxt.Inc()
		log.Println("Call2: calling next func")
		Call3(ctxt)
	}
}

func Call3(ctxt *Context) {
	log.Println("Call3: called")
	select {
	case <-ctxt.done:
		log.Println(os.Stderr, "Call3: aborted")
		return
	default:
		ctxt.Inc()
		log.Println("Call3: calling next func")
		Call4(ctxt)
	}
}

func Call4(ctxt *Context) {
	log.Println("Call4: called")
	select {
	case <-ctxt.done:
		log.Println(os.Stderr, "Call4: aborted")
		return
	default:
		ctxt.Inc()
		log.Println("Call4: calling next func")
		Call5(ctxt)
	}
}

func Call5(ctxt *Context) {
	log.Println("Call5: called")
	select {
	case <-ctxt.done:
		log.Println(os.Stderr, "Call5: aborted")
		return
	default:
		ctxt.Inc()
		log.Println("Call5: calling next func")
		Call6(ctxt)
	}
}

func Call6(ctxt *Context) {
	log.Println("Call6: called")
	select {
	case <-ctxt.done:
		log.Println(os.Stderr, "Call6: aborted")
		return
	default:
		ctxt.Inc()
		log.Println("Call6: calling next func")
		Call7(ctxt)
	}
}

func Call7(ctxt *Context) {
	log.Println("Call7: called")
	select {
	case <-ctxt.done:
		log.Println(os.Stderr, "Call7: aborted")
		return
	default:
		ctxt.Inc()
		log.Println("Call7: calling next func")
		Call8(ctxt)
	}
}

func Call8(ctxt *Context) {
	log.Println("Call8: called")
	select {
	case <-ctxt.done:
		log.Println(os.Stderr, "Call8: aborted")
		return
	default:
		ctxt.Inc()
		log.Println("Call8: calling next func")
		Call9(ctxt)
	}
}

func Call9(ctxt *Context) {
	log.Println("Call9: called")
	select {
	case <-ctxt.done:
		log.Println(os.Stderr, "Call9: aborted")
		return
	default:
		ctxt.Inc()
		log.Println("Call9: calling next func")
		Call10(ctxt)
	}
}
