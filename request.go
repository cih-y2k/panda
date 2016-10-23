package panda

import (
	"fmt"
	"io"
	"time"
)

type requestValue struct {
	key   []byte
	value interface{}
}

type requestValues []requestValue

func (r *requestValues) Set(key string, value interface{}) {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if string(kv.key) == key {
			kv.value = value
			return
		}
	}

	c := cap(args)
	if c > n {
		args = args[:n+1]
		kv := &args[n]
		kv.key = append(kv.key[:0], key...)
		kv.value = value
		*r = args
		return
	}

	kv := requestValue{}
	kv.key = append(kv.key[:0], key...)
	kv.value = value
	*r = append(args, kv)
}

func (r *requestValues) Get(key string) interface{} {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if string(kv.key) == key {
			return kv.value
		}
	}
	return nil
}

func (r *requestValues) Reset() {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		v := args[i].value
		// close any values which implements the Closer, some of the ORM packages does that.
		if vc, ok := v.(io.Closer); ok {
			vc.Close()
		}
	}
	*r = (*r)[:0]
}

var (
	// MaxHandlers the max number of handlers that allowed to be executed per request, default is 255 handlers!
	MaxHandlers = 255
)

// Request from receiver to the sender, and waits for answer, a Result
// one client can send multiple different requests 'server' aka Result sender
// lowercase used by the server handler, uppercase used as payload too
type Request struct {
	Conn       *Conn
	handlers   []Handler
	pos        int
	errMessage string
	result     interface{}

	// the per-request store
	values requestValues

	From      CID    // connection id,
	Statement string // the call statement
	Args      Args   // the statement's method's arguments, if any
	// the unique request's id which waits for a Result with the same RequestID, may empty if not waiting for Result.
	// Channel is just a non-struct methodology for request-Result-reqsponse-request communication,
	// its id made by client before sent to the server, the same id is used for the server's Result
	ID string
}

// Serve calls the middleware
func (req *Request) Serve() {
	if len(req.handlers) == 0 {
		req.Error(errHandlerNotFound.Format(req.Statement).String())
		return
	}

	req.handlers[req.pos](req)
	req.pos++
	//run the next
	if req.pos < len(req.handlers) {
		req.Serve()

	}
}

// ForceNext forces to serve the next handler
func (req *Request) ForceNext() {
	req.pos++

	if req.pos < len(req.handlers) {
		req.handlers[req.pos](req)
	}
}

// Cancel just sets the .pos to the MaxHandlers in order to  not move to the next middlewares(if any)
func (req *Request) Cancel() {
	req.pos = MaxHandlers
}

// CancelWithError cancels the next handler's execution and saves the error which will should be sent to the client
func (req *Request) CancelWithError(format string, a ...interface{}) {
	if format != "" {
		req.Error(format, a...)
		req.Cancel()
	}
}

// Error sends an error to the client and stops the execution of the next handlers
func (req *Request) Error(format string, a ...interface{}) {
	req.errMessage = fmt.Sprintf(format, a...)
	req.Cancel()
}

// Result saves the result to be sent to the client, you're free to change this result from the whole handlers lifetime
func (req *Request) Result(v interface{}) {
	req.result = v
}

/*
  Compatibility with standard Golang's Context
	Implement the interface: https://github.com/golang/net/blob/master/context/context.go#L45
*/

// Deadline returns the time when work done on behalf of this context
// should be canceled.  Deadline returns ok==false when no deadline is
// set.  Successive calls to Deadline return the same results.
func (req *Request) Deadline() (deadline time.Time, ok bool) {
	return
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled.  Done may return nil if this context can
// never be canceled.  Successive calls to Done return the same value.
//
// WithCancel arranges for Done to be closed when cancel is called;
// WithDeadline arranges for Done to be closed when the deadline
// expires; WithTimeout arranges for Done to be closed when the timeout
// elapses.
//
// Done is provided for use in select statements:
//
//  // Stream generates values with DoSomething and sends them to out
//  // until DoSomething returns an error or ctx.Done is closed.
//  func Stream(ctx context.Context, out chan<- Value) error {
//  	for {
//  		v, err := DoSomething(ctx)
//  		if err != nil {
//  			return err
//  		}
//  		select {
//  		case <-ctx.Done():
//  			return ctx.Err()
//  		case out <- v:
//  		}
//  	}
//  }
//
// See http://blog.golang.org/pipelines for more examples of how to use
// a Done channel for cancelation.
func (req *Request) Done() <-chan struct{} {
	return nil
}

// Err returns a non-nil error value after Done is closed.  Err returns
// Canceled if the context was canceled or DeadlineExceeded if the
// context's deadline passed.  No other values for Err are defined.
// After Done is closed, successive calls to Err return the same value.
func (req *Request) Err() error {
	return nil
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key.  Successive calls to Value with
// the same key returns the same result.
//
// Use context values only for request-scoped data that transits
// processes and API boundaries, not for passing optional parameters to
// functions.
//
// A key identifies a specific value in a Context.  Functions that wish
// to store values in Context typically allocate a key in a global
// variable then use that key as the argument to context.WithValue and
// Context.Value.  A key can be any type that supports equality;
// packages should define keys as an unexported type to avoid
// collisions.
//
// Packages that define a Context key should provide type-safe accessors
// for the values stores using that key:
//
// 	// Package user defines a User type that's stored in Contexts.
// 	package user
//
// 	import "golang.org/x/net/context"
//
// 	// User is the type of value stored in the Contexts.
// 	type User struct {...}
//
// 	// key is an unexported type for keys defined in this package.
// 	// This prevents collisions with keys defined in other packages.
// 	type key int
//
// 	// userKey is the key for user.User values in Contexts.  It is
// 	// unexported; clients use user.NewContext and user.FromContext
// 	// instead of using this key directly.
// 	var userKey key = 0
//
// 	// NewContext returns a new Context that carries value u.
// 	func NewContext(ctx context.Context, u *User) context.Context {
// 		return context.WithValue(ctx, userKey, u)
// 	}
//
// 	// FromContext returns the User value stored in ctx, if any.
// 	func FromContext(ctx context.Context) (*User, bool) {
// 		u, ok := ctx.Value(userKey).(*User)
// 		return u, ok
// 	}
func (req *Request) Value(key interface{}) interface{} {
	if key == 0 {
		return req
	}

	// we support only string keys, so if not a string then return nil
	if k, ok := key.(string); ok {
		return req.Get(k)
	}
	return nil
}

// Set sets an item to the request per-call/request storage (KV)
func (req *Request) Set(key string, value interface{}) {
	req.values.Set(key, value)
}

// Get returns an item based on its key from the local per-call/request storage
func (req *Request) Get(key string) interface{} {
	return req.values.Get(key)
}

// GetString returns the string representation of the local per-call/request storage
func (req *Request) GetString(key string) string {
	if v := req.Value(key); v != nil {
		if s, ok := v.(string); ok {
			return s
		}

	}
	return ""
}
