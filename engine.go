package panda

import (
	"log"
	"net"
	"sync"
)

type (
	// Options contains the engine's option fields
	Options struct {
		logger *log.Logger
		codec  Codec
		buffer uint64
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
func OptionBuffer(val uint64) OptionSet {
	return func(o *Options) {
		o.buffer = val
	}
}

// Engine the engine which is the same for server and client side
// contains the connections, the handlers and any useful information to serve connection(s)
// both client connection or serve server's side incoming clients/connections
type Engine struct {
	opt         *Options
	handlers    *handlersMux
	ns          *namespace
	cpool       sync.Pool
	connections []*conn
	mu          sync.Mutex
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
		Middleware: &Middleware{},
	}

	e.cpool.New = func() interface{} { return &conn{engine: e, mu: &sync.RWMutex{}, pending: make(map[string]Response)} }

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

var connIDCounter struct {
	sync.Mutex
	value CID
}

func newConnID() CID {
	connIDCounter.Lock()
	connIDCounter.value++
	id := connIDCounter.value
	connIDCounter.Unlock()
	return id
}

func (e *Engine) acquireConn(underline net.Conn) *conn {
	c := e.cpool.Get().(*conn)
	c.Conn = underline

	c.id = newConnID()
	c.incomingRes = make(Response, e.opt.buffer)
	c.incomingReq = make(Request, e.opt.buffer)
	c.isClosed = false
	e.mu.Lock()
	e.connections = append(e.connections, c)
	e.mu.Unlock()
	return c
}

func (e *Engine) releaseConn(c *conn) (err error) {
	// close the connection here
	// if manually closed then it should be reach to the releaseConn which will try to close it again and provides an error but we don't care about it atm.
	if !c.Closed() {
		err = c.Close()
	}

	close(c.incomingReq)
	close(c.incomingRes)
	c.CancelAllPending()

	idx := 0
	e.mu.Lock()
	defer e.mu.Unlock()

	for idx = range e.connections {
		if c.ID() == e.connections[idx].ID() {
			break
		}
	}

	if idx > -1 {
		e.connections[idx] = e.connections[len(e.connections)-1]
		e.connections = e.connections[:len(e.connections)-1]
	}

	e.cpool.Put(c)
	return
}

// GetConn returns a connection by id
// the returned Conn cannot be changed
func (e *Engine) getConn(id CID) *conn {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, c := range e.connections {
		if c.ID() == id {
			return c
		}
	}
	return nil
}

///TODO: send an ack message before any other message to the client, which will be setting it's internal connection's ID too
