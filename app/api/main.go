package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/mahtues/go-chat/channeling"
	"github.com/mahtues/go-chat/misc"
	"golang.org/x/net/websocket"
)

var (
	rabbitMqUrlArg = flag.String("rabbitmq-url", "", "rabbitmq url")
	rabbitMqUrlEnv = os.Getenv("rabbitmq-url")
	rabbitMqUrl    = "amqp://localhost:5672"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile | log.Lmsgprefix)
	flag.Parse()

	var (
		rabbitMqUrl = misc.FirstNonEmpty(*rabbitMqUrlArg, rabbitMqUrlEnv, rabbitMqUrl)
	)

	log.Printf("rabbitMqUrl=%#v", rabbitMqUrl)

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "pong\r\n")
	})

	http.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		log.Printf("ws handler start")
		defer log.Printf("ws handler exit")
		defer ws.Close()

		recv := channeling.NewRecver(ws)

		for f := range recv.Ch() {
			log.Printf("frame id: %+v / text: %+v", f.Id, f.TextTo.Text)
		}
	}))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
}
