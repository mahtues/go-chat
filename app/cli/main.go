package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mahtues/go-chat/frame"
	"github.com/mahtues/go-chat/misc"
	"golang.org/x/net/websocket"
)

var (
	urlArg = flag.String("url", "", "server host url")
	urlEnv = os.Getenv("url")
	url    = "ws://localhost:8080/ws"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile | log.Lmsgprefix)
	flag.Parse()

	var (
		url = misc.FirstNonEmpty(*urlArg, urlEnv, url)
	)

	log.Printf("url=%+v", url)

	ws, err := websocket.Dial(url, "", "http://localhost/")
	if err != nil {
		log.Fatalf("dial failed: %v", err)
	}

	conn := NewConn(ws, 10)

	go func() {
		log.Printf("conn.Receive range start")
		defer log.Printf("conn.Receive range exit")
		for f := range conn.Receive() {
			log.Printf("frame: %+v", f)
		}
	}()

	id := uint64(0)
	scanner := bufio.NewScanner(os.Stdin)
	for fmt.Printf("> "); scanner.Scan(); fmt.Printf("> ") {
		text := scanner.Text()
		err := conn.Send(frame.Frame{Id: id, TextTo: &frame.TextTo{"mahtues", text}})
		if err != nil {
			log.Fatalf("send text failed: %v", err)
		}
		id++
	}
}

type Conn struct {
	ws         *websocket.Conn
	pending    []frame.Frame
	maxPending uint64
	lastErr    error
	out        chan frame.Frame
}

func NewConn(ws *websocket.Conn, maxPending uint64) *Conn {
	conn := &Conn{
		ws:         ws,
		pending:    make([]frame.Frame, 0, maxPending),
		maxPending: maxPending,
		lastErr:    nil,
	}
	go conn.loop()
	return conn
}

func (c *Conn) Receive() <-chan frame.Frame {
	return c.out
}

func (c *Conn) Send(frame frame.Frame) error {
	return websocket.JSON.Send(c.ws, frame)
}

func (c *Conn) Close() error {
	return c.lastErr
}

func (c *Conn) loop() {
	for {
		select {}
	}
}
