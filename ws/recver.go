package ws

import (
	"sync"

	"github.com/mahtues/go-chat/frame"
	"golang.org/x/net/websocket"
)

func NewRecver(ws *websocket.Conn) *Recver {
	r := &Recver{
		ws:        ws,
		outch:     make(chan frame.Frame),
		closech:   make(chan chan error),
		closeonce: sync.Once{},
	}
	go r.loop()
	return r
}

type Recver struct {
	ws        *websocket.Conn
	outch     chan frame.Frame
	closech   chan chan error
	err       error
	closeonce sync.Once
}

func (r *Recver) Ch() <-chan frame.Frame {
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
		frm         frame.Frame
		framech     chan frame.Frame = nil
		framechmemo chan frame.Frame = make(chan frame.Frame)
		err         error            = nil
		errch       chan error       = nil
		errchmemo   chan error       = make(chan error)
		outch       chan frame.Frame = nil
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

func recv(ws *websocket.Conn, framech chan<- frame.Frame, errch chan<- error) {
	var frm frame.Frame
	err := websocket.JSON.Receive(ws, &frm) // blocking
	if err != nil {
		errch <- err
	} else {
		framech <- frm
	}
}

func cleanup(framech <-chan frame.Frame, errch <-chan error) {
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
