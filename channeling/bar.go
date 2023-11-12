package channeling

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

	recv := func() {
		var frm frame.Frame
		err := websocket.JSON.Receive(r.ws, &frm) // blocking
		if err != nil {
			errch <- err
		} else {
			framech <- frm
		}
	}

	defer func() {
		close(r.outch)

		// handle hanging recv() goroutine
		if framech != nil || errch != nil {
			go func() {
				select {
				case <-errch:
				case <-framech:
				}
				close(errch)
				close(framech)
			}()
		}
	}()

	framech, errch, outch = framechmemo, errchmemo, nil
	go recv()

	for {
		select {
		case close := <-r.closech:
			close <- err
			return
		case err = <-errch:
			framech, errch, outch = nil, nil, nil
			go r.Close()
		case frm = <-framech:
			framech, errch, outch = nil, nil, r.outch
		case outch <- frm:
			framech, errch, outch = framechmemo, errchmemo, nil
			go recv()
		}
	}
}
