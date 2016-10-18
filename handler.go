package panda

import (
	"fmt"
	"github.com/kataras/go-errors"
	"github.com/plar/go-adaptive-radix-tree"
)

// Arg TODO:
type Arg interface{}

// BeginHandler TODO:
// or return a context which will be sended to the Handler as parameter in order to have custom cancelation, timeouts and so on ?
// or just let the user to decide it and code that?....
type BeginHandler func(Conn, ...Arg) bool // if false then the  actual handler never executed instead returns nil and error.Canceled on the client

// Handler TODO:
type Handler func(Conn, ...Arg) (interface{}, error)

// DoneHandler TODO:
type DoneHandler func(Conn, []Arg, interface{}, error)

// the done handlers execute after handler on each own go routine, doesn't communicate with the client, it's useful for logginf purposes

// Middleware TODO:
type Middleware struct {
	begin []BeginHandler
	done  []DoneHandler
}

// Begin TODO:
func (t *Middleware) Begin(h ...BeginHandler) {
	t.begin = append(t.begin, h...)
}

// Done TODO:
func (t *Middleware) Done(h ...DoneHandler) {
	t.done = append(t.done, h...)
}

// Canceled TODO:
type Canceled struct{}

func (c Canceled) Error() string {
	return "Canceled"
}

// Invalid TOOD:
type Invalid struct {
	conn *conn
}

func (i Invalid) Error() string {
	return fmt.Sprintf("Invalid response, connection: '%#v' was not found", i.conn) // conn should be nil but for any case print it
}

func buildHandler(begin []BeginHandler, h Handler, done []DoneHandler) Handler {
	handler := func(c Conn, args ...Arg) (interface{}, error) {
		// check if can continue , if not return nil and canceled as error
		for i := range begin {
			if !begin[i](c, args...) {
				return nil, Canceled{}
			}
		}
		result, err := h(c, args...) // execute the handler and save them

		// run done middleware on it's own goroutine, useful for logging or -(log save to database) purposes
		go func(c Conn, done []DoneHandler, result interface{}, err error) {
			for i := range done {
				done[i](c, args, result, err)
			}
		}(c, done, result, err)

		return result, err

	}

	return handler

}

type handlersMux struct {
	tree art.Tree
}

func newHandlersMux() *handlersMux {
	return &handlersMux{tree: art.New()}
}

func (h *handlersMux) add(statement string, handler Handler) {
	h.tree.Insert(art.Key(statement), handler)
}

func (h *handlersMux) find(statement string) Handler {
	handler, found := h.tree.Search(art.Key(statement))
	if !found {
		return nil
	}
	return handler.(Handler)
}

func (h *handlersMux) forEach(callback func(statement string, handler Handler) bool) {
	artCb := func(node art.Node) bool {
		return callback(string(node.Key()), node.Value().(Handler))
	}

	h.tree.ForEach(artCb)
}

var errHandlerNotFound = errors.New("Statement handler: %s not found")

func (h *handlersMux) exec(conn Conn, statement string, args ...Arg) (interface{}, error) {
	if handler := h.find(statement); handler != nil {
		return handler(conn, args...)
	}
	return nil, errHandlerNotFound.Format(statement)
}
