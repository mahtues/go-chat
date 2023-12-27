package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/net/websocket"

	"github.com/mahtues/go-chat/app/cli/chat"
	"github.com/mahtues/go-chat/app/cli/login"
	"github.com/mahtues/go-chat/data"
	"github.com/mahtues/go-chat/log"
	"github.com/mahtues/go-chat/misc"
	"github.com/mahtues/go-chat/ws"
)

var (
	urlArg = flag.String("url", "", "server host url")
	urlEnv = os.Getenv("url")
	url    = "ws://localhost:8080/ws"
)

func main() {
	flag.Parse()

	url := misc.FirstNonEmpty(*urlArg, urlEnv, url)

	log.Infof("url=%+v", url)

	var (
		p      *tea.Program
		wsconn *websocket.Conn
		err    error
	)

	wsconn, err = websocket.Dial(url, "", "http://127.0.0.1/")
	if err != nil {
		log.Fatalf("dial failed: %v", err)
	}
	defer wsconn.Close()

	c := chat.New()
	c.SendTextTo = func(frm data.Event) {
		go func() {
			err := websocket.JSON.Send(wsconn, frm)
			if err != nil {
				log.Fatalf("send text failed: %v", err)
			}
		}()
	}

	l := login.New(c)

	p = tea.NewProgram(l)

	go func() {
		recv := ws.NewRecver(wsconn)
		for f := range recv.Ch() {
			p.Send(f)
		}
	}()

	if _, err = p.Run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func old(url string) {
	wsconn, err := websocket.Dial(url, "", "http://127.0.0.1/")
	if err != nil {
		log.Fatalf("dial failed: %v", err)
	}
	defer wsconn.Close()

	go func() {
		recv := ws.NewRecver(wsconn)
		for f := range recv.Ch() {
			log.Debugf("frame: %+v", f)
		}
	}()

	id := uint64(0)
	scanner := bufio.NewScanner(os.Stdin)
	for fmt.Printf("> "); scanner.Scan(); fmt.Printf("> ") {
		text := scanner.Text()
		err := websocket.JSON.Send(wsconn, data.Event{Id: id, TextTo: &data.TextTo{"guest", text}})
		if err != nil {
			log.Fatalf("send text failed: %v", err)
		}
		id++
	}
}

type foo struct{}

func (m foo) Init() tea.Cmd {
	return nil
}
