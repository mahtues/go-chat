package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/mahtues/go-chat/frame"
	"github.com/mahtues/go-chat/misc"
	"github.com/mahtues/go-chat/ws"
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

		mqch, err := mqchan.Consume(mq.Name, "", true, false, false, false, nil)
		if err != nil {
			errorf("create mq consume channel failed: %v", err)
			return
		}

		wsrecv := ws.NewRecver(wsconn)
		defer wsrecv.Close()
		wsinch := wsrecv.Ch()

		wssend := ws.NewSender(wsconn)
		wsoutch := wssend.Ch()
		defer wssend.Close()

		var (
			mqfrm     frame.Frame
			wsfrm     frame.Frame
			enwsoutch chan<- frame.Frame = nil
			delivery  amqp.Delivery
			enmqch    <-chan amqp.Delivery = nil
			ok        bool
			abort     bool = false
		)

		enwsoutch, enmqch = nil, mqch

		for !abort {
			select {
			case wsfrm, ok = <-wsinch:
				if !ok {
					errorf("<-wsinch error: %v", err)
					abort = true
					break
				}
				b, _ := json.Marshal(wsfrm)
				mqchan.Publish("", mq.Name, false, false, amqp.Publishing{ContentType: "application/json", Body: b})
			case delivery, ok = <-enmqch:
				if !ok {
					abort = true
					errorf("<-enmqch error: %v", err)
					break
				}
				err = json.Unmarshal(delivery.Body, &mqfrm)
				enwsoutch, enmqch = wsoutch, nil
			case enwsoutch <- mqfrm:
				// error handling missing
				enwsoutch, enmqch = nil, mqch
			}
		}

		infof("clean up required")
	}))

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
}

func nop(args ...any) {
}
