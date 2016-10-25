package panda

import (
	"fmt"
	"strconv"
)

// NamespaceAPI TODO:
type NamespaceAPI interface {
	Middleware
	Name() string
	Namespace(name string) NamespaceAPI
	Handle(statement string, h ...Handler)
	Lookup(statement string) Handlers
	VisitLookup(callback func(statement string, h Handlers) bool)
	DoAsync(c *Conn, statement string, raw bool, args ...Args) Response
	Do(c *Conn, statement string, args ...Args) (interface{}, error)
	// DoRaw use it when you want to manually encode/decode the object or just write to another writer, this is faster method
	DoRaw(c *Conn, statement string, args ...Args) ([]byte, error)
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
func (ns *namespace) Handle(statement string, h ...Handler) {
	if statement == "" || len(h) == 0 {
		return
	}

	// build the main handler
	handlers := append(ns.middleware.begin, h...)
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
func (ns *namespace) DoAsync(c *Conn, statement string, raw bool, args ...Args) Response {
	statement = ns.fullname + sep + statement // if root then it's simply /statement

	// ok here we want just map[string]interface{}, the variadic is only for optionallity so:
	// if len(args) > 1 we should pass the [1:] as key["1"], key["2"], hoping that the first argument set is not containing a name like "1"..

	var arguments Args

	if len(args) > 0 {
		arguments = args[0]
	}
	if len(args) > 1 {
		for i := 1; i < len(args); i++ {
			arguments[strconv.Itoa(i)] = args[i]
		}
	}

	return c.sendRequest(statement, arguments, raw)
}

// Do TODO:
func (ns *namespace) Do(c *Conn, statement string, args ...Args) (interface{}, error) {
	resp := <-ns.DoAsync(c, statement, false, args...)
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	return resp.Result, nil
}

// Do TODO:
func (ns *namespace) DoRaw(c *Conn, statement string, args ...Args) ([]byte, error) {
	resp := <-ns.DoAsync(c, statement, true, args...)
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	return resp.RawResult, nil //resp.Data.MarshalJSON()
}

// client side just .Call/Do(statement, args...)
// but server-side should have .Call/Do(conn,statement,args...) (for bind-directional communication)
// so we will separate the namespace in ClientNamespace and ServerNamespace

// ClientNamespace TODO:
type ClientNamespace struct {
	*namespace
	conn *Conn // we need only Conn but if we use it here it will be like we are setting conn here and not to the client itself...
}

// Namespace TODO:
func (ns *ClientNamespace) Namespace(name string) *ClientNamespace {
	return &ClientNamespace{
		conn:      ns.conn,
		namespace: ns.namespace.newNamespace(name),
	}
}

// DoAsync TODO: make the channel exported*
func (ns *ClientNamespace) DoAsync(statement string, raw bool, args ...Args) Response {
	return ns.namespace.DoAsync(ns.conn, statement, raw, args...)
}

// Do TODO:
func (ns *ClientNamespace) Do(statement string, args ...Args) (interface{}, error) {
	return ns.namespace.Do(ns.conn, statement, args...)
}

// DoRaw TODO:
func (ns *ClientNamespace) DoRaw(statement string, args ...Args) ([]byte, error) {
	return ns.namespace.DoRaw(ns.conn, statement, args...)
}
