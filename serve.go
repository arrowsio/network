package network

import (
	"net"
	"github.com/chuckpreslar/emission"
)

type Serve struct {
	listener  *net.Listener
	listeners []func(pipe *Pipe)
	Emission  *emission.Emitter
}

func NewServe() *Serve {
	return &Serve{Emission: emission.NewEmitter()}
}

// Loop the pipe handler
func (this *Serve) handle(p *Pipe) {
	for _, fn := range this.listeners {
		fn(p)
	}
}

func (this *Serve) Listen(ifp string) error {
	ln, err := net.Listen("tcp", ifp)
	if err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		pipe := createPipe(conn, this.Emission)
		go pipe.routine()
		go this.handle(pipe)
	}
}

func (this *Serve) Subscribe(fn func(pipe *Pipe)) {
	this.listeners = append(this.listeners, fn)
}

// Subscribe to emission
func (this *Serve) On(event, listener interface{}) *Serve {
	this.Emission.AddListener(event, listener)
	return this
}

// Release an emission subscription
func (this *Serve) Off(event, listener interface{}) *Serve {
	this.Emission.RemoveListener(event, listener)
	return this
}
