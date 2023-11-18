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
			mqfrwdchmemo chan frwdResult = make(chan frwdResult)
			wsrecvch     chan recvResult = nil
			mqfrwdch     chan frwdResult = nil

			// mq -> ws
			mqrecvchmemo chan recvResult = make(chan recvResult)
			wsfrwdchmemo chan frwdResult = make(chan frwdResult)
			mqrecvch     chan recvResult = nil
			wsfrwdch     chan frwdResult = nil
		)

		wsrecvch, mqfrwdch = wsrecvchmemo, nil
		go wsrecv(wsconn, wsrecvch)

		mqrecvch, wsfrwdch = mqrecvchmemo, nil
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
				// validate frame
				mqfrwdch = mqfrwdchmemo
				go mqfrwd(mqchan, result.frame, mqfrwdch)

			case result := <-mqfrwdch:
				mqfrwdch = nil
				if result.err != nil {
					errorf("error: %v", result.err)
					abort = true
					break
				}
				// confirm forward
				wsrecvch = wsrecvchmemo
				go wsrecv(wsconn, wsrecvch)

			case result := <-mqrecvch:
				mqrecvch = nil
				if result.err != nil {
					errorf("error: %v", result.err)
					abort = true
					break
				}
				// validate frame
				wsfrwdch = wsfrwdchmemo
				go wsfrwd(wsconn, result.frame, wsfrwdch)

			case result := <-wsfrwdch:
				wsfrwdch = nil
				if result.err != nil {
					errorf("error: %v", result.err)
					abort = true
					break
				}
				// confirm forward
				mqrecvch = mqrecvchmemo
				go mqrecv(mqinch, mqrecvch)
			}
		}

		if wsrecvch != nil {
			go func() { <-wsrecvch }()
		}

		if mqfrwdch != nil {
			go func() { <-mqfrwdch }()
		}

		if mqrecvch != nil {
			go func() { <-mqrecvch }()
		}

		if wsfrwdch != nil {
			go func() { <-wsfrwdch }()
		}
	}))

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		fatalf("listen error: %v", err)
	}
}

type recvFunc func(chan<- recvResult)

type frwdFunc func(frame.Frame, chan<- frwdResult)

type recvResult struct {
	frame frame.Frame
	err   error
}

type frwdResult struct {
	err error
}

func wsfrwd(wsconn *websocket.Conn, frm frame.Frame, resultch chan<- frwdResult) {
	err := websocket.JSON.Send(wsconn, frm) // blocking. needs a goroutine
	resultch <- frwdResult{err}
}

func wsrecv(wsconn *websocket.Conn, resultch chan<- recvResult) {
	var frm frame.Frame
	err := websocket.JSON.Receive(wsconn, &frm) // blocking. needs a goroutine
	resultch <- recvResult{frm, err}
}

func mqfrwd(mqchan *amqp.Channel, frm frame.Frame, resultch chan<- frwdResult) {
	b, _ := json.Marshal(frm)
	err := mqchan.Publish("", "guest", false, false, amqp.Publishing{ContentType: "application/json", Body: b}) // blocking. needs a goroutine
	resultch <- frwdResult{err}
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
