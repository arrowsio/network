package network
import (
	"net"
	"time"
	"errors"
	"github.com/chuckpreslar/emission"
	"encoding/json"
)

const (
	writeTimeout = 1 * time.Minute
	minTimeout = 15 * time.Second
	maxTimeout = 2 * time.Minute
	pingInterval = 1 * time.Minute
)

type Pipe struct {
	conn      net.Conn
	data      []byte
	lastWrite time.Time
	pongStart time.Time
	linger    bool
	pongs     chan time.Duration
	listeners []func(*Pipe, uint16, []byte)
	Emission  *emission.Emitter
}

func Dial(ifp string) (*Pipe, error) {
	conn, err := net.Dial("tcp", ifp)
	if err != nil {
		return nil, err
	}
	pipe := createPipe(conn, emission.NewEmitter())
	go pipe.routine()
	return pipe, nil
}

func createPipe(conn net.Conn, em *emission.Emitter) *Pipe {
	pipe := Pipe{conn:conn, linger: false, pongs: make(chan time.Duration, 1), Emission: em}
	pipe.Subscribe(pipe.pubSub)
	return &pipe
}

func (this *Pipe) routine() {
	hdr, e := this.readData(8)
	if e != nil {
		this.Close()
		return
	}
	header, _ := headerFrom(hdr)
	if !header.isValid() {
		this.Close()
	}
	switch header.getType() {
	case TLinger:
		this.linger = true
	case TPing:
		this.pong()
	case TPong:
		this.pongs <- time.Now().Sub(this.pongStart)
	default:
		chunk, err := this.readData(header.chunkSize())
		if err != nil {
			this.Close()
		}
		go this.handle(header.getType(), chunk)
	}
	this.routine()
}

// Read x amount of data from the wire
func (c *Pipe) readData(want uint32) ([]byte, error) {
	var buffer []byte
	transfer, take := uint32(0), uint32(0)

	// How long should we wait before a header gets sent in
	if c.linger {
		c.setReadDeadLine(maxTimeout)
	} else {
		c.setReadDeadLine(minTimeout)
	}
	//Loop till we grab all the data
	for transfer < want {
		take = want
		if take > 512 {
			take = 512
		}
		buf := make([]byte, take)
		c.setReadDeadLine(minTimeout)
		count, err := c.conn.Read(buf)
		if err != nil {
			return nil, err
		}
		transfer += uint32(count)
		buffer = append(buffer, buf[:count]...)
	}
	return buffer, nil
}

// Put data on the wire
func (c *Pipe) writeData(b []byte) (int, error) {
	c.setWriteDeadLine()
	return c.conn.Write(b)
}

// Set timeout for reading data from the wire
func (c *Pipe) setReadDeadLine(d time.Duration) {
	c.conn.SetReadDeadline(time.Now().Add(d))
}

// Set timeout for writing data onto the wire
func (c *Pipe) setWriteDeadLine() {
	c.conn.SetReadDeadline(time.Now().Add(writeTimeout))
}

// Respond to a ping
func (c *Pipe) pong() {
	c.writeData(createHeader(TPong, 0))
}

// Loop the byte handler
func (this *Pipe) handle(u uint16, b []byte) {
	for _, fn := range this.listeners {
		fn(this, u, b)
	}
}

// Emission handler
func (this *Pipe) pubSub(sender *Pipe, typ uint16, chunk []byte) {
	if typ != TPubSub {
		return
	}
	a := []interface{}{}
	json.Unmarshal(chunk, &a)
	if len(a) == 1 {
		this.Emission.Emit(a[0], this)
	} else {
		this.Emission.Emit(a[0], append([]interface{}{this}, a[1:]...)...)
	}
}

// Emit data to emission
func (c *Pipe) Emit(event interface{}, arguments ...interface{}) error {
	a := []interface{}{event}
	a = append(a, arguments...)
	bts, err := json.Marshal(a)
	if err != nil {
		return err
	}
	_, err = c.Write(TPubSub, bts)
	return err
}

// Tell the other side to keep this connection open for as long as possible.
func (c *Pipe) Linger() error {
	c.linger = true
	_, err := c.writeData(createHeader(TLinger, 0))
	return err
}

// Close connection and cleanup
func (c *Pipe) Close() error {
	close(c.pongs)
	return c.conn.Close()
}

// Send a ping to the other side.
func (c *Pipe) Ping() (chan time.Duration, error) {
	c.pongStart = time.Now()
	if _, err := c.writeData(createHeader(TPing, 0)); err != nil {
		return nil, err
	}
	return c.pongs, nil;
}

// Get channel of pongs that are held
func (c *Pipe) Pongs() chan time.Duration {
	return c.pongs
}

// Write some kind of data onto the wire WITH a header
func (c *Pipe) Write(typ uint16, chunk []byte) (int, error) {
	le := len(chunk)
	if le > MaxChunk {
		return 0, errors.New("Chunk size overflows header buffer.")
	}
	if le == 0 {
		return 0, errors.New("Refusing to send a chunk of size 0.")
	}
	if _, err := c.writeData(createHeader(typ, uint32(le))); err != nil {
		return 0, err
	}
	count, err := c.writeData(chunk)
	return count, err
}

func (c *Pipe) Subscribe(fn func(*Pipe, uint16, []byte)) {
	c.listeners = append(c.listeners, fn)
}

func (c *Pipe) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Pipe) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Pipe) KeepAlive() {
	go func() {
		for {
			if time.Now().Sub(c.lastWrite).Seconds() >= pingInterval.Seconds() {
				_, err := c.Ping()
				if err != nil {
					return
				}
			}
			time.Sleep(pingInterval)
		}
	}()
}

// Subscribe to emission
func (c *Pipe) On(event, listener interface{}) *Pipe {
	c.Emission.AddListener(event, listener)
	return c
}

// Release an emission subscription
func (c *Pipe) Off(event, listener interface{}) *Pipe {
	c.Emission.RemoveListener(event, listener)
	return c
}
