package ws

import (
	"sync"

	"github.com/mahtues/go-chat/frame"
	"golang.org/x/net/websocket"
)

func NewSender(ws *websocket.Conn) *Sender {
	r := &Sender{
		ws:        ws,
		inch:      make(chan frame.Frame),
		closech:   make(chan chan error),
		closeonce: sync.Once{},
	}
	go r.loop()
	return r
}

type Sender struct {
	ws        *websocket.Conn
	inch      chan frame.Frame
	closech   chan chan error
	err       error
	closeonce sync.Once
}

func (r *Sender) Ch() chan<- frame.Frame {
	return r.inch
}

func (r *Sender) Close() error {
	r.closeonce.Do(func() {
		errch := make(chan error)
		r.closech <- errch
		r.err = <-errch
	})
	return r.err
}

func (r *Sender) loop() {
	var (
		frm       frame.Frame
		okch      chan struct{}    = nil
		okchmemo  chan struct{}    = make(chan struct{})
		err       error            = nil
		errch     chan error       = nil
		errchmemo chan error       = make(chan error)
		inch      chan frame.Frame = nil
	)

	okch, errch, inch = nil, nil, r.inch

	for {
		select {
		case closech := <-r.closech:
			closech <- err
			close(r.inch)
			cleanupsender(okch, errch)
			return
		case err = <-errch:
			okch, errch, inch = nil, nil, nil
			go r.Close()
		case <-okch:
			okch, errch, inch = nil, nil, r.inch
		case frm = <-inch:
			okch, errch, inch = okchmemo, errchmemo, nil
			go send(r.ws, frm, okch, errch)
		}
	}
}

func send(ws *websocket.Conn, frm frame.Frame, okch chan<- struct{}, errch chan<- error) {
	err := websocket.JSON.Send(ws, frm) // blocking
	if err != nil {
		errch <- err
	} else {
		okch <- struct{}{}
	}
}

func cleanupsender(okch <-chan struct{}, errch <-chan error) {
	// handle hanging recv() goroutine
	if okch != nil || errch != nil {
		go func() {
			select {
			case <-okch:
			case <-errch:
			}
		}()
	}
}
