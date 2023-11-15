package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"

	"golang.org/x/net/websocket"

	"github.com/mahtues/go-chat/frame"
	"github.com/mahtues/go-chat/log"
	"github.com/mahtues/go-chat/misc"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	portArg = flag.String("port", "", "listening port")
	portEnv = os.Getenv("port")
	port    = "8080"

	rabbitMqUrlArg = flag.String("rabbitmq-url", "", "rabbitmq url")
	rabbitMqUrlEnv = os.Getenv("rabbitmq-url")
	rabbitMqUrl    = "amqp://localhost:5672"

	infof  = log.Infof
	errorf = log.Errorf
	fatalf = log.Fatalf
)

func main() {
	flag.Parse()

	var (
		rabbitMqUrl = misc.FirstNonEmpty(*rabbitMqUrlArg, rabbitMqUrlEnv, rabbitMqUrl)
		port        = misc.FirstNonEmpty(*portArg, portEnv, port)
	)

	infof("rabbitMqUrl=%#v", rabbitMqUrl)
	infof("port=%#v", port)

	mqconn, err := amqp.Dial(rabbitMqUrl)
	if err != nil {
		errorf("create mq connection failed: %v", err)
		return
	}

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "pong\r\n")
	})

	http.Handle("/ws", websocket.Handler(func(wsconn *websocket.Conn) {
		defer infof("handler exit")
		defer wsconn.Close()

		mqchan, err := mqconn.Channel()
		if err != nil {
			errorf("create mq channel failed: %v", err)
			return
		}
		defer mqchan.Close()

		mq, err := mqchan.QueueDeclare("guest", false, false, false, false, nil)
		if err != nil {
			errorf("create mq channel failed: %v", err)
			return
		}

		mqinch, err := mqchan.Consume(mq.Name, "", true, false, false, false, nil)
		if err != nil {
			errorf("create mq consume channel failed: %v", err)
			return
		}

		var (
			// ws -> mq
			wsrecvchmemo chan recvResult = make(chan recvResult)
			mqsendchmemo chan sendResult = make(chan sendResult)
			wsrecvch     chan recvResult = nil
			mqsendch     chan sendResult = nil

			// mq -> ws
			mqrecvchmemo chan recvResult = make(chan recvResult)
			wssendchmemo chan sendResult = make(chan sendResult)
			mqrecvch     chan recvResult = nil
			wssendch     chan sendResult = nil
		)

		wsrecvch, mqsendch = wsrecvchmemo, nil
		go wsrecv(wsconn, wsrecvch)

		mqrecvch, wssendch = mqrecvchmemo, nil
		go mqrecv(mqinch, mqrecvch)

		abort := false

		for !abort {
			select {
			case result := <-wsrecvch:
				wsrecvch = nil
				if result.err != nil {
					errorf("error: %v", result.err)
					abort = true
					break
				}
				mqsendch = mqsendchmemo
				go mqsend(mqchan, result.frame, mqsendch)

			case result := <-mqsendch:
				mqsendch = nil
				if result.err != nil {
					errorf("error: %v", result.err)
					abort = true
					break
				}
				wsrecvch = wsrecvchmemo
				go wsrecv(wsconn, wsrecvch)

			case result := <-mqrecvch:
				mqrecvch = nil
				if result.err != nil {
					errorf("error: %v", result.err)
					abort = true
					break
				}
				wssendch = wssendchmemo
				go wssend(wsconn, result.frame, wssendch)

			case result := <-wssendch:
				wssendch = nil
				if result.err != nil {
					errorf("error: %v", result.err)
					abort = true
					break
				}
				mqrecvch = mqrecvchmemo
				go mqrecv(mqinch, mqrecvch)
			}
		}

		if wsrecvch != nil {
			go func() { <-wsrecvch }()
		}

		if mqsendch != nil {
			go func() { <-mqsendch }()
		}

		if mqrecvch != nil {
			go func() { <-mqrecvch }()
		}

		if wssendch != nil {
			go func() { <-wssendch }()
		}
	}))

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		fatalf("listen error: %v", err)
	}
}

type recvResult struct {
	frame frame.Frame
	err   error
}

type sendResult struct {
	err error
}

func wssend(wsconn *websocket.Conn, frm frame.Frame, resultch chan<- sendResult) {
	err := websocket.JSON.Send(wsconn, frm) // blocking. needs a goroutine
	resultch <- sendResult{err}
}

func wsrecv(wsconn *websocket.Conn, resultch chan<- recvResult) {
	var frm frame.Frame
	err := websocket.JSON.Receive(wsconn, &frm) // blocking. needs a goroutine
	resultch <- recvResult{frm, err}
}

func mqsend(mqchan *amqp.Channel, frm frame.Frame, resultch chan<- sendResult) {
	b, _ := json.Marshal(frm)
	err := mqchan.Publish("", "guest", false, false, amqp.Publishing{ContentType: "application/json", Body: b}) // blocking. needs a goroutine
	resultch <- sendResult{err}
}

func mqrecv(mqinch <-chan amqp.Delivery, resultch chan<- recvResult) {
	var frm frame.Frame
	delivery, ok := <-mqinch
	if !ok {
		resultch <- recvResult{frm, io.EOF}
		return
	}
	err := json.Unmarshal(delivery.Body, &frm)
	resultch <- recvResult{frm, err}
}

func nop(args ...any) {
}
