package panda

import (
	"fmt"
)

// NamespaceAPI TODO:
type NamespaceAPI interface {
	Middleware
	Name() string
	Namespace(name string) NamespaceAPI
	Handle(statement string, h Handler)
	Lookup(statement string) Handlers
	VisitLookup(callback func(statement string, h Handlers) bool)
	DoAsync(connID int, statement string, args ...Arg) Response
	Do(connID int, statement string, args ...Arg) (interface{}, error)
}

type namespace struct {
	*middleware // these are the begin and done handlers for all handlers
	fullname    string
	engine      *Engine
}

var _ NamespaceAPI = &namespace{}

const sep = "/"

// Name TODO:
func (ns *namespace) Name() string {
	return ns.fullname
}

func (ns *namespace) newNamespace(name string) *namespace {
	if ns.fullname != "" {
		name = ns.fullname + sep + name
	}

	// copy begin and done parent handlers
	midl := &middleware{}
	if ns.middleware != nil {
		midl.begin = ns.middleware.begin
		midl.done = ns.middleware.done
	}

	return &namespace{
		engine:     ns.engine,
		fullname:   name,
		middleware: midl,
	}
}

// Namespace TODO:
func (ns *namespace) Namespace(name string) NamespaceAPI {
	return ns.newNamespace(name)
}

// Handle TODO:
func (ns *namespace) Handle(statement string, h Handler) {
	if statement == "" || h == nil {
		return
	}

	// build the main handler
	handlers := append(ns.middleware.begin, h)
	handlers = append(handlers, ns.middleware.done...)
	ns.engine.handlers.add(ns.fullname+sep+statement, handlers)
}

// Lookup returns the handlers with this statement on this namespace
func (ns *namespace) Lookup(statement string) Handlers {
	statement = ns.fullname + sep + statement
	return ns.engine.handlers.find(statement)
}

// VisitLookup receives a callback function type for tree traversal.
// if the callback function returns false then iteration is terminated
func (ns *namespace) VisitLookup(callback func(statement string, h Handlers) bool) {
	ns.engine.handlers.forEach(callback)
}

// DoAsync TODO: make the channel exported*
func (ns *namespace) doAsync(c *conn, statement string, args ...Arg) Response {
	statement = ns.fullname + sep + statement // if root then it's simply /statement
	// send to the 'server'
	resCh := make(Response, 1)
	c.sendRequestAsync(statement, args, resCh)
	return resCh
}

// DoAsync TODO: make the channel exported*
func (ns *namespace) DoAsync(connID int, statement string, args ...Arg) Response {
	return ns.doAsync(ns.engine.getConn(CID(connID)), statement, args...)
}

// Do TODO:
func (ns *namespace) do(c *conn, statement string, args ...Arg) (interface{}, error) {
	resp := <-ns.doAsync(c, statement, args...)
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return resp.Data, nil
}

func (ns *namespace) Do(connID int, statement string, args ...Arg) (interface{}, error) {
	return ns.do(ns.engine.getConn(CID(connID)), statement, args...)
}

// client side just .Call/Do(statement, args...)
// but server-side should have .Call/Do(conn,statement,args...) (for bind-directional communication)
// so we will separate the namespace in ClientNamespace and ServerNamespace

// ClientNamespace TODO:
type ClientNamespace struct {
	*namespace
	conn *conn // we need only Conn but if we use it here it will be like we are setting conn here and not to the client itself...
}

// Namespace TODO:
func (ns *ClientNamespace) Namespace(name string) *ClientNamespace {
	return &ClientNamespace{
		conn:      ns.conn,
		namespace: ns.namespace.newNamespace(name),
	}
}

// DoAsync TODO: make the channel exported*
func (ns *ClientNamespace) DoAsync(statement string, args ...Arg) Response {
	return ns.namespace.doAsync(ns.conn, statement, args...)
}

// Do TODO:
func (ns *ClientNamespace) Do(statement string, args ...Arg) (interface{}, error) {
	return ns.namespace.do(ns.conn, statement, args...)
}
