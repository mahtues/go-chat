package ws

import (
	"sync"

	"github.com/mahtues/go-chat/data"
	"golang.org/x/net/websocket"
)

func NewRecver(ws *websocket.Conn) *Recver {
	r := &Recver{
		ws:        ws,
		outch:     make(chan data.Event),
		closech:   make(chan chan error),
		closeonce: sync.Once{},
	}
	go r.loop()
	return r
}

type Recver struct {
	ws        *websocket.Conn
	outch     chan data.Event
	closech   chan chan error
	err       error
	closeonce sync.Once
}

func (r *Recver) Ch() <-chan data.Event {
	return r.outch
}

func (r *Recver) Close() error {
	r.closeonce.Do(func() {
		errch := make(chan error)
		r.closech <- errch
		r.err = <-errch
	})
	return r.err
}

func (r *Recver) loop() {
	var (
		frm         data.Event
		framech     chan data.Event = nil
		framechmemo chan data.Event = make(chan data.Event)
		err         error           = nil
		errch       chan error      = nil
		errchmemo   chan error      = make(chan error)
		outch       chan data.Event = nil
	)

	framech, errch, outch = framechmemo, errchmemo, nil
	go recv(r.ws, framech, errch)

	for {
		select {
		case closech := <-r.closech:
			closech <- err
			close(r.outch)
			cleanup(framech, errch)
			return
		case err = <-errch:
			framech, errch, outch = nil, nil, nil
			go r.Close()
		case frm = <-framech:
			framech, errch, outch = nil, nil, r.outch
		case outch <- frm:
			framech, errch, outch = framechmemo, errchmemo, nil
			go recv(r.ws, framech, errch)
		}
	}
}

func recv(ws *websocket.Conn, framech chan<- data.Event, errch chan<- error) {
	var frm data.Event
	err := websocket.JSON.Receive(ws, &frm) // blocking
	if err != nil {
		errch <- err
	} else {
		framech <- frm
	}
}

func cleanup(framech <-chan data.Event, errch <-chan error) {
	// handle hanging recv() goroutine
	if framech != nil || errch != nil {
		go func() {
			select {
			case <-errch:
			case <-framech:
			}
		}()
	}
}
