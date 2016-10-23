package panda

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
)

type (
	// Options contains the engine's option fields
	Options struct {
		logger *log.Logger
		codec  Codec
		buffer int
	}
	// OptionSetter sets a configuration field to the engine's Options
	// used to help developers to write less and configure only what they really want and nothing else
	OptionSetter interface {
		Set(*Options)
	}
	// OptionSet implements the OptionSetter
	OptionSet func(*Options)
)

// Set is the func which makes the OptionSet an OptionSetter, this is used mostly
func (o OptionSet) Set(main *Options) {
	o(main)
}

// Set implements the OptionSetter for the Options struct itself so it can be passed as OptionSetter to the 'constructor' too
func (o Options) Set(main *Options) {
	if o.logger != nil {
		main.logger = o.logger
	}

	if o.codec != nil {
		main.codec = o.codec
	}

	main.buffer = o.buffer
}

// OptionLogger TODO:
func OptionLogger(val *log.Logger) OptionSet {
	return func(o *Options) {
		o.logger = val
	}
}

// OptionCodec TODO:
func OptionCodec(val Codec) OptionSet {
	return func(o *Options) {
		o.codec = val
	}
}

// OptionBuffer TODO:
func OptionBuffer(val int) OptionSet {
	return func(o *Options) {
		o.buffer = val
	}
}

// Engine the engine which is the same for server and client side
// contains the connections, the handlers and any useful information to serve connection(s)
// both client connection or serve server's side incoming clients/connections
type Engine struct {
	opt      *Options
	handlers *handlersMux
	ns       *namespace
	cpool    sync.Pool
}

// NewEngine returns a new engine for use
// only .Set is exported to set any custom option before listening to the server and serving client connections or before connect to the server as client
func NewEngine(setters ...OptionSetter) *Engine {
	e := &Engine{
		handlers: newHandlersMux(),
	}

	e.ns = &namespace{
		fullname:   "",
		engine:     e,
		middleware: &middleware{},
	}

	e.cpool.New = func() interface{} {
		c := &Conn{
			engine:  e,
			pending: make(map[string]Response, 0),
		}

		c.reqPool.New = func() interface{} {
			return &Request{Conn: c}
		}

		return c
	}

	e.Set(setters...)
	return e
}

// Set sets any options for customization to the Engine
func (e *Engine) Set(setters ...OptionSetter) {
	if e.opt == nil {
		e.opt = &Options{codec: DefaultCodec}
	}
	for _, s := range setters {
		s.Set(e.opt)
	}
}

func (e *Engine) logf(format string, v ...interface{}) {
	if e.opt.logger != nil {
		e.opt.logger.Printf(format, v...)
	}
}

var cidCounter struct {
	value uint64
}

func newCID() CID {
	atomic.AddUint64(&cidCounter.value, 1)
	return CID(atomic.LoadUint64(&cidCounter.value)) // ?
	//connIDCounter.value++
	//return CID(connIDCounter.value)
}

func (e *Engine) acquireConn(underline net.Conn) *Conn {
	c := e.cpool.Get().(*Conn)
	c.Conn = underline

	c.id = newCID()
	c.outcomingReq = make(request, e.opt.buffer)
	c.incomingRes = make(Response, e.opt.buffer)
	c.incomingReq = make(request, e.opt.buffer)
	c.stopCh = make(chan struct{}, 1)
	c.isClosed = false

	return c
}

func (e *Engine) releaseConn(c *Conn) (err error) {
	// close the connection here
	// if manually closed then it should be reach to the releaseConn which will try to close it again so introduce the isClosed
	if !c.isClosed {
		err = c.Close()
	}

	close(c.incomingReq)
	close(c.incomingRes)

	c.CancelAllPending()

	e.cpool.Put(c)
	return
}

///TODO: send an ack message before any other message to the client, which will be setting it's internal connection's ID too
