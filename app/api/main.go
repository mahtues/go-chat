package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/mahtues/go-chat/frame"
	"github.com/mahtues/go-chat/misc"
	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/net/websocket"
)

var (
	portArg = flag.String("port", "", "listening port")
	portEnv = os.Getenv("port")
	port    = "8080"

	rabbitMqUrlArg = flag.String("rabbitmq-url", "", "rabbitmq url")
	rabbitMqUrlEnv = os.Getenv("rabbitmq-url")
	rabbitMqUrl    = "amqp://localhost:5672"

	infof  = log.Printf
	errorf = log.Printf
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile | log.Lmsgprefix)
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

		type recvResult struct {
			frame frame.Frame
			err   error
		}

		type sendResult struct {
			err error
		}

		var (
			// ws -> mq
			//wsrecvch chan recvResult = make(chan recvResult)
			//mqsendch chan sendResult = make(chan sendResult)

			wsfrm          frame.Frame
			wserr          error
			wsfrmch        chan frame.Frame = nil
			wsinerrch      chan error       = nil
			mqoutokch      chan struct{}    = nil
			mqouterrch     chan error       = nil
			wsfrmchmemo    chan frame.Frame = make(chan frame.Frame)
			wsinerrchmemo  chan error       = make(chan error)
			mqoutokchmemo  chan struct{}    = make(chan struct{})
			mqouterrchmemo chan error       = make(chan error)

			// mq -> ws
			mqfrm          frame.Frame
			mqerr          error
			mqfrmch        chan frame.Frame = nil
			mqinerrch      chan error       = nil
			wsoutokch      chan struct{}    = nil
			wsouterrch     chan error       = nil
			mqfrmchmemo    chan frame.Frame = make(chan frame.Frame)
			mqinerrchmemo  chan error       = make(chan error)
			wsoutokchmemo  chan struct{}    = make(chan struct{})
			wsouterrchmemo chan error       = make(chan error)
		)

		wsfrmch, wsinerrch, mqoutokch, mqouterrch = wsfrmchmemo, wsinerrchmemo, nil, nil
		go wsrecv(wsconn, wsfrmch, wsinerrch)
		mqfrmch, mqinerrch, wsoutokch, wsouterrch = mqfrmchmemo, mqinerrchmemo, nil, nil
		go mqrecv(mqinch, mqfrmch, mqinerrch)

		abort := false

		for !abort {
			select {
			case wsfrm = <-wsfrmch:
				wsfrmch, wsinerrch, mqoutokch, mqouterrch = nil, nil, mqoutokchmemo, mqouterrchmemo
				go mqsend(mqchan, wsfrm, mqoutokch, mqouterrch)
			case <-mqoutokch:
				wsfrmch, wsinerrch, mqoutokch, mqouterrch = wsfrmchmemo, wsinerrchmemo, nil, nil
				go wsrecv(wsconn, wsfrmch, wsinerrch)
			case wserr = <-mqouterrch:
				errorf("error: %v", wserr)
				abort = true
				break
			case wserr = <-wsinerrch:
				errorf("error: %v", wserr)
				abort = true
				break

			case mqfrm = <-mqfrmch:
				mqfrmch, mqinerrch, wsoutokch, wsouterrch = nil, nil, wsoutokchmemo, wsouterrchmemo
				go wssend(wsconn, mqfrm, wsoutokch, wsouterrch)
			case <-wsoutokch:
				mqfrmch, mqinerrch, wsoutokch, wsouterrch = mqfrmchmemo, mqinerrchmemo, nil, nil
				go mqrecv(mqinch, mqfrmch, mqinerrch)
			case mqerr = <-wsouterrch:
				errorf("error: %v", mqerr)
				abort = true
				break
			case mqerr = <-mqinerrch:
				errorf("error: %v", mqerr)
				abort = true
				break
			}
		}

		if wsfrmch != nil {
			go func() {
				select {
				case <-wsfrmch:
				case <-wsinerrch:
				}
			}()
		}

		if mqoutokch != nil {
			go func() {
				select {
				case <-mqoutokch:
				case <-mqouterrch:
				}
			}()
		}

		if mqfrmch != nil {
			go func() {
				select {
				case <-mqfrmch:
				case <-mqinerrch:
				}
			}()
		}

		if wsoutokch != nil {
			go func() {
				select {
				case <-wsoutokch:
				case <-wsouterrch:
				}
			}()
		}
	}))

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
}

func wssend(wsconn *websocket.Conn, frm frame.Frame, okch chan struct{}, errch chan<- error) {
	err := websocket.JSON.Send(wsconn, frm) // blocking. needs a goroutine
	if err == nil {
		okch <- struct{}{}
	} else {
		errch <- err
	}
}

func wsrecv(wsconn *websocket.Conn, frmch chan<- frame.Frame, errch chan<- error) {
	var frm frame.Frame
	err := websocket.JSON.Receive(wsconn, &frm) // blocking. needs a goroutine
	if err == nil {
		frmch <- frm
	} else {
		errch <- err
	}
}

func mqsend(mqchan *amqp.Channel, frm frame.Frame, okch chan<- struct{}, errch chan<- error) {
	b, _ := json.Marshal(frm)
	err := mqchan.Publish("", "guest", false, false, amqp.Publishing{ContentType: "application/json", Body: b}) // blocking. needs a goroutine
	if err == nil {
		okch <- struct{}{}
	} else {
		errch <- err
	}
}

func mqrecv(mqinch <-chan amqp.Delivery, frmch chan<- frame.Frame, errch chan<- error) {
	delivery, ok := <-mqinch
	if !ok {
		errch <- io.EOF
	}
	var frm frame.Frame
	err := json.Unmarshal(delivery.Body, &frm)
	if err == nil {
		frmch <- frm
	} else {
		errch <- err
	}
}

func nop(args ...any) {
}
