package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/mahtues/go-chat/misc"
)

func main() {
	go func() {
		http.ListenAndServe(":10240", nil)
	}()

	for {
		donech := make(chan struct{})

		time.AfterFunc(3*time.Second, func() { close(donech) })

		sq := misc.Sq
		gen := misc.Gen

		for x := range sq(donech, gen(donech, time.Second)) {
			log.Printf("x=%v", x)
		}
	}

	log.Printf("exit")

	select {}
}
