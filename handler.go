package panda

import (
	"fmt"
	"github.com/kataras/go-errors"
	"github.com/plar/go-adaptive-radix-tree" ///TODO: find a way to fix it
)

// Arg TODO:
type Arg interface{}

// Args TOOD:
type Args []Arg

// Get returns an argument based on its called position
func (args Args) Get(idx int) interface{} {
	for i := range args {
		if i != idx {
			continue
		}
		return args[i]
	}
	return nil
}

// String returns a typeof string argument based on its called position
func (args Args) String(idx int) string {
	if arg := args.Get(idx); arg != nil {
		if s, ok := arg.(string); ok {
			return s
		}
	}
	return ""
}

// Int returns a typeof int argument based on its called position
func (args Args) Int(idx int) int {
	if arg := args.Get(idx); arg != nil {
		return MustDecodeInt(arg)
	}
	return -1
}

/*
no...

// GetString returns a value(string) from a key inside this Args
// If no argument with this key given then it returns an empty string
func (args Args) Get(key string) interface{} {
	for _, p := range params {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

// GetString returns a value(string) from a key inside this Args
// If no argument with this key given then it returns an empty string
func (args Args) GetString(key string) string {
	for _, p := range params {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

// String returns a string implementation of all arguments that this Args object keeps
// has the form of key1=value1,key2=value2...
func (args Args) String() string {
	var buff bytes.Buffer
	for i := range params {
		buff.WriteString(params[i].Key)
		buff.WriteString("=")
		buff.WriteString(params[i].Value)
		if i < len(params)-1 {
			buff.WriteString(",")
		}

	}
	return buff.String()
}

// ParseParams receives a string and returns PathParameters (slice of Args)
// received string must have this form:  key1=value1,key2=value2...
func ParseParams(str string) Args {
	_paramsstr := strings.Split(str, ",")
	if len(_paramsstr) == 0 {
		return nil
	}

	params := make(PathParameters, 0) // PathParameters{}

	//	for i := 0; i < len(_paramsstr); i++ {
	for i := range _paramsstr {
		idxOfEq := strings.IndexRune(_paramsstr[i], '=')
		if idxOfEq == -1 {
			//error
			return nil
		}

		key := _paramsstr[i][:idxOfEq]
		val := _paramsstr[i][idxOfEq+1:]
		params = append(params, PathParameter{key, val})
	}
	return params
}
*/

// Handler TODO:
type Handler func(*Request) // no better to do it like request.Result() and request.Error() which can be changed between multi handlers (interface{}, error)
// Handlers the handler chain
type Handlers []Handler

// the done handlers execute after handler on each own go routine, doesn't communicate with the client, it's useful for logginf purposes

// Middleware TODO:
type Middleware interface {
	Begin(...Handler)
	Done(...Handler)
}

var _ Middleware = &middleware{}

type middleware struct {
	begin Handlers
	done  Handlers
}

// Begin TODO:
func (t *middleware) Begin(h ...Handler) {
	t.begin = append(t.begin, h...)
}

// Done TODO:
func (t *middleware) Done(h ...Handler) {
	t.done = append(t.done, h...)
}

type handlersMux struct {
	tree art.Tree
}

func newHandlersMux() *handlersMux {
	return &handlersMux{tree: art.New()}
}

func (h *handlersMux) add(statement string, handlers Handlers) {
	h.tree.Insert(art.Key(statement), handlers)
}

func (h *handlersMux) find(statement string) Handlers {
	handler, found := h.tree.Search(art.Key(statement))
	if !found {
		return nil
	}
	return handler.(Handlers)
}

func (h *handlersMux) forEach(callback func(statement string, handlers Handlers) bool) {
	artCb := func(node art.Node) bool {
		return callback(string(node.Key()), node.Value().(Handlers))
	}

	h.tree.ForEach(artCb)
}

var errHandlerNotFound = errors.New("Statement handler: %s not found")

func (h *handlersMux) exec(req *Request) (interface{}, error) {
	handlers := h.find(req.Statement)
	req.handlers = handlers
	req.Serve()
	var err error
	if req.errMessage != "" {
		err = fmt.Errorf("%s", req.errMessage)
	}
	return req.result, err
}

// middleware helper

// ArgsLen a middleware which allows handler to execute when parameters length are between min and max
func ArgsLen(min int, max int) Handler { ///TOOD: context and Handler or make an Args
	return func(req *Request) {
		args := req.Args
		largs := len(args)
		if largs < min || largs > max {
			req.Error("Arguments expected: %d >= %d && %d <= %d", largs, min, largs, max)
		}
	}

}
