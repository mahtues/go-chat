package misc

import (
	"log"
	"time"
)

func Sq(donech <-chan struct{}, inch <-chan int) <-chan int {
	outch := make(chan int)

	go func() {
		defer close(outch)
		defer log.Printf("sq exit")

		var (
			ich <-chan int = inch
			och chan<- int = nil
			x   int        = 0
		)

		for {
			select {
			case <-donech:
				return
			case x = <-ich:
				och, ich = outch, nil
			case och <- x * x:
				och, ich = nil, inch
			}
		}
	}()

	return outch
}

func Gen(donech <-chan struct{}, delta time.Duration) <-chan int {
	outch := make(chan int)

	go func() {
		defer close(outch)
		defer log.Printf("gen exit")

		var (
			x      int        = 0
			och    chan<- int = outch
			ticker            = time.NewTicker(delta)
		)

		for {
			select {
			case <-donech:
				return
			case och <- x:
				och = nil
				x++
			case <-ticker.C:
				och = outch
			}
		}
	}()

	return outch
}

func Sqb(donech <-chan struct{}, inch <-chan int) <-chan int {
	outch := make(chan int)

	go func() {
		defer close(outch)
		defer log.Printf("sq exit")

		for x := range inch {
			select {
			case <-donech:
				return
			case outch <- x * x:
			}
		}
	}()

	return outch
}

func Genb(donech <-chan struct{}, delta time.Duration) <-chan int {
	outch := make(chan int)

	go func() {
		defer close(outch)
		defer log.Printf("gen exit")

		var (
			x      int        = 0
			och    chan<- int = outch
			ticker            = time.NewTicker(delta)
		)

		for {
			select {
			case <-donech:
				return
			case och <- x:
				och = nil
				x++
			case <-ticker.C:
				och = outch
			}
		}
	}()

	return outch
}
