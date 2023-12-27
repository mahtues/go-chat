package main

import (
	"context"
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/mahtues/go-chat/data"
	"github.com/mahtues/go-chat/log"
	"github.com/mahtues/go-chat/misc"
	"golang.org/x/sync/semaphore"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	portArg = flag.String("port", "", "listening port")
	portEnv = os.Getenv("port")
	port    = "8080"

	rabbitMqUrlArg = flag.String("rabbitmq-url", "", "rabbitmq url")
	rabbitMqUrlEnv = os.Getenv("rabbitmq-url")
	rabbitMqUrl    = "amqp://localhost:5672"

	maxWorkersArg = flag.Int64("max-workers", 10, "max number of workers")
	maxWorkersEnv = misc.ToInt64(os.Getenv("max-workers"))
	maxWorkers    = int64(10)

	infof  = log.Infof
	errorf = log.Errorf
	fatalf = log.Fatalf
)

func main() {
	flag.Parse()

	var (
		rabbitMqUrl = misc.FirstNonEmpty(*rabbitMqUrlArg, rabbitMqUrlEnv, rabbitMqUrl)
		port        = misc.FirstNonEmpty(*portArg, portEnv, port)
		maxWorkers  = misc.FirstNonNegative(*maxWorkersArg, maxWorkersEnv, maxWorkers)
	)

	go http.ListenAndServe(":"+port, nil)

	infof("rabbitMqUrl=%#v", rabbitMqUrl)
	infof("port=%#v", port)
	infof("maxWorkers=%#v", maxWorkers)

	mqconn, err := amqp.Dial(rabbitMqUrl)
	if err != nil {
		fatalf("create mq connection failed: %v", err)
	}

	mqchan, err := mqconn.Channel()
	if err != nil {
		fatalf("create mq channel failed: %v", err)
	}
	defer mqchan.Close()

	mq, err := mqchan.QueueDeclare("worker", false, false, false, false, nil)
	if err != nil {
		fatalf("create mq channel failed: %v", err)
	}

	mqinch, err := mqchan.Consume(mq.Name, "", true, false, false, false, nil)
	if err != nil {
		fatalf("create mq consume channel failed: %v", err)
	}

	var (
		ctx = context.TODO()
		sem = semaphore.NewWeighted(maxWorkers)
	)

	sem.Acquire(ctx, 1)
	for delivery := range mqinch {
		go func(delivery amqp.Delivery) {
			defer sem.Release(1)

			frm, err := data.FromBytes(delivery.Body)
			if err != nil {
				errorf("frame unmarshal failed: %v", err)
				return
			}
			log.Debugf("frame: #v", frm)
		}(delivery)

		sem.Acquire(ctx, 1)
	}
}
