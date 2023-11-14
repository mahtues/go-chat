package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mahtues/go-chat/frame"
	"github.com/mahtues/go-chat/misc"
	"github.com/mahtues/go-chat/ws"
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

	wsconn, err := websocket.Dial(url, "", "http://localhost/")
	if err != nil {
		log.Fatalf("dial failed: %v", err)
	}
	defer wsconn.Close()

	go func() {
		recv := ws.NewRecver(wsconn)
		for f := range recv.Ch() {
			log.Printf("frame: %+v", f)
		}
	}()

	id := uint64(0)
	scanner := bufio.NewScanner(os.Stdin)
	for fmt.Printf("> "); scanner.Scan(); fmt.Printf("> ") {
		text := scanner.Text()
		err := websocket.JSON.Send(wsconn, frame.Frame{Id: id, TextTo: &frame.TextTo{"guest", text}})
		if err != nil {
			log.Fatalf("send text failed: %v", err)
		}
		id++
	}
}
