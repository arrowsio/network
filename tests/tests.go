package main
import (
	"time"
	"github.com/pranked/network"
	"fmt"
)

type testing struct {
	Someting string
}

func main() {
	serv := network.NewServe()
	serv.On("lol", func(p *network.Pipe, s testing) {

	})
	go serv.Listen(":2000")
	pipe, _ := network.Dial(":2000")
	pipe.Linger()
	pipe.KeepAlive()
	pipe.Emit("lol", "test")
	//Just wait around for awhile
	time.Sleep(2 * time.Hour)
}