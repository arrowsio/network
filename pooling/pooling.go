package pooling
import "github.com/chuckpreslar/emission"

type Pool struct {
	Emission *emission.Emitter
	address string
	min, max uint16

}

func New(address string, min, max uint16) {

}