package panda

import (
	"bufio"
	"bytes"
	"net"
	//	"reflect"
	"strconv"
	"sync"
)

// CID the type of the connection ID, currently int
type CID int

// Conn TODO:
type Conn interface {
	net.Conn
	ID() CID
	CancelAllPending()
	Closed() bool
}

type (

	// Result , from sender to receiver
	Result struct {
		RequestID string // the request id, maybe empty if it's not created to answer for a request from the client, it's made by client
		// the deserialized result, which is map when default json codec is used,
		// but I am not explicit set it as map[string]interface{} because your custom codec may differs
		// becaue of that we have a .To function which will convert this , as map to a struct
		// error if any from the server for this particular request's Result
		// If it's a struct then it's a map[string]interface{}, json ready-to-use. If it's int then it's float64, all other standar types as they are.
		Data  interface{}
		Error string // error cannot be json-encoded/decoded so it's string but handlers returns error as user expects
	}

	// Ack acknowedge packet message, sent from server to the client when the client connection connected (for first time)
	// client waits to receive the 'ok' channel, it's used mostly to block between client's .Connect/.Dial and .Do/.DoAsync methods
	// but no.. why use a different message type and dublicate our code twice..? let's use the already Request and Response implementation
	//Ack chan ack

	// Response the Result channel binds to request
	Response chan Result
)

// Decode writes the 'Result', which should be a map[string]interface{} if receiver expected a custom type struct,
// to the vPointer which should be a custom type of go struct.
//
// Note: it's useless if you wanna re-send this to your http api
func (r Result) Decode(vPointer interface{}) {
	if r.Data != nil {
		DecodeResult(vPointer, r.Data)
	}
}

func canceledResponse(reqID string) Result {
	return Result{
		RequestID: reqID,
		Error:     Canceled{}.Error(),
	}
}

func invalidResponse(conn *conn) Result {
	return Result{
		Error: Invalid{conn}.Error(),
	}
}

type conn struct {
	net.Conn
	id     CID
	engine *Engine
	// client
	incomingRes Response
	// server
	incomingReq chan Request
	// client
	pending map[string]Response
	mu      *sync.RWMutex

	reqPool  sync.Pool
	isClosed bool
}

var _ Conn = &conn{}

// ID returns the connection's (incremental) id
func (c *conn) ID() CID {
	return c.id
}

// setID used only by client to set its connection's ID which is coming from the server with the ack request and Result message type
func (c *conn) setID(id int) {
	c.id = CID(id)
}

// acquireRequest used both on client and server
// when act like client sending to the server called on sendRequest and released very soon
// when act like server is created by the Request, called on handleRequest
// general rule: when we see Request without a pointer receiver then it's payload, sent from client to server, otherwise it's the request 'context
func (c *conn) acquireRequest(statement string, args Args) *Request {
	req := c.reqPool.Get().(*Request)
	if req.ID == "" { // if from client to server, then generate id, otherwise
		req.ID = strconv.Itoa(int(c.ID())) + "_" + RandomString(6)
	}

	req.Statement = statement
	req.Args = args
	// fields From and conn are not changing*
	return req
}

// releaseRequest used both on client and server
// when conn acts like client sending to the server called on sendRequest and released very soon
// when conn acts like server is created by the Request, called on handleRequest
// general rule: when we see Request without a pointer receiver then it's payload, sent from client to server, otherwise it's the request 'context
func (c *conn) releaseRequest(req *Request) {
	req.ID = ""
	req.Args = req.Args[0:0]
	req.Statement = ""
	req.values.Reset()
	req.handlers = nil
	req.errMessage = ""
	req.pos = 0
	c.reqPool.Put(req)
}

var (
	requestPrefix   = []byte("REQ")
	responsePrefix  = []byte("RES")
	prefixLen       = 3
	whiteSpaceBytes = []byte(" ")
	requestPrefixW  = append(requestPrefix, whiteSpaceBytes...)
	responsePrefixW = append(responsePrefix, whiteSpaceBytes...)
)

func (c *conn) serve() {
	go c.handle()
	c.reader()
	c.Close()
}

func (c *conn) reader() {
	scanner := bufio.NewScanner(c)
	for scanner.Scan() {
		incomingData := scanner.Bytes()
		go func(incomingData []byte) {
			if len(incomingData) < prefixLen+1 {
				return // skip this message it's not request or Result/channel
			}
			// because of requestPrefix and ResultPrefix have the same length,
			// we take only the  the first prefixLen digits as the cmd
			command := incomingData[0:prefixLen] // prefixLen does not contains the space
			data := incomingData[prefixLen+1:]
			// DEBUG:
			//println("DEBUG server.go:107 | command: " + string(command) + " data: " + string(data))

			// act like client
			if bytes.Equal(command, responsePrefix) {
				// DEBUG:
				//println("act like client waiting for channel")
				// is incoming Result/ channel
				var resp Result
				err := c.engine.opt.codec.Deserialize(data, &resp)
				if err != nil {
					return
				}
				c.incomingRes <- resp
				// DEBUG: println("Sent")

			} else if bytes.Equal(command, requestPrefix) {
				// act like server
				// DEBUG: println("act like server sending a Result channel")
				// is incoming request waits for Result
				var req Request
				err := c.engine.opt.codec.Deserialize(data, &req)
				if err != nil {
					return
				}
				c.incomingReq <- req
				// DEBUG: println("Sent")

			} else {
				// println("conn.go:161 no compatible message")
				// if neither request or Result/channel then, break, close the conn because no other type of message is supported.
				return
			}

		}(incomingData)

	}

}

func (c *conn) handle() {
	// the conn should be closed on reader's error via releaseConn but make sure that if channels are closed the conn is closed too
	for {
		select {
		// conn acts like client
		case resp, ok := <-c.incomingRes:
			{
				if !ok || c.isClosed {
					return
				}
				//println("DEBUG conn.go:208 incoming res with req id: " + resp.RequestID)

				if resp.RequestID == "" {
					//println("DEBUG conn.go:210 req id is EMPTY?")
					continue
				}
				c.mu.Lock() // THIS SHOULD NOT BE NEEDED HERE BUT otherwise we have concurrent map writes on line 267 on 1kk goroutines per 20 seconds
				if pendingResp, found := c.pending[resp.RequestID]; found {
					// //println("DEBUG conn.go:214 sending back to the .Do the Result")
					// var vData interface{}
					// err := c.engine.opt.codec.Deserialize(resp.Data, &vData)
					// if err != nil {
					// 	resp.Error = err.Error()
					// } else {
					// 	resp.returnData = vData
					// }
					pendingResp <- resp
					close(pendingResp)
					delete(c.pending, resp.RequestID)
				}
				c.mu.Unlock()

			}
		// conn acts like server
		case req, ok := <-c.incomingReq:
			{
				if !ok || c.isClosed {
					return
				}
				go c.handleRequest(req)
			}

		}
	}
}

func (c *conn) sendRequestAsync(statement string, args Args, resCh Response) {
	if c == nil || c.isClosed {
		resCh <- invalidResponse(c)
		return
	}

	req := c.acquireRequest(statement, args)
	defer c.releaseRequest(req)

	// before contine,  check if it's a local handler
	if h := c.engine.handlers.find(statement); h != nil {
		req.handlers = h
		req.Serve()

		resp := Result{
			RequestID: "",
			Data:      req.result,
			Error:     req.errMessage,
		}
		resCh <- resp

		return
	}
	// otherwise send the request payload to the server
	data, err := c.engine.opt.codec.Serialize(req)

	if err != nil || c.pending == nil {
		resCh <- canceledResponse(req.ID)
	}

	// add the pending here, after go 1.6 we should do with write and read lock here...(;)
	c.mu.Lock()

	c.pending[req.ID] = resCh

	c.mu.Unlock()
	//c.engine.logf("Sending request: %#v", req)

	c.Write(append(requestPrefixW, data...))
}

// HandleRequest ...TODO:
func (c *conn) handleRequest(payload Request) {
	req := c.acquireRequest(payload.Statement, payload.Args)
	req.ID = payload.ID // send the correct request id
	defer c.releaseRequest(req)

	// we don't care if found or not, if an error must be sent to the client
	handlers := c.engine.handlers.find(req.Statement)
	req.handlers = handlers
	req.Serve()
	resp := Result{RequestID: req.ID, Data: req.result, Error: req.errMessage}

	//println("conn.go:224 AFTER execute handler with statement: " + req.Statement + " and requestID: " + req.ID)

	// first check for serialization errors, if and only here the error is not going back to the client for security reasons, but the server is notified
	data, serr := c.engine.opt.codec.Serialize(resp)
	if serr != nil {
		c.engine.logf("Serialization failed for %#v on Connection with ID: %d, on Statement: %s", resp, c.ID(), req.Statement)
		return
	}
	c.Write(append(responsePrefixW, data...))
}

// CancelAllPending TODO:
func (c *conn) CancelAllPending() {
	c.mu.Lock()
	for id, pending := range c.pending {
		// clean up any pending message by sending a canceled error Result
		pending <- canceledResponse(id)
		close(pending)
		delete(c.pending, id)
	}
	c.mu.Unlock()
}

var newLineBytes = []byte("\n")

func appendNewline(b []byte) []byte {
	return append(b, newLineBytes...)
}

func (c *conn) Write(data []byte) (int, error) {
	return c.Conn.Write(appendNewline(data))
}

// Closed returns true if the connection has been closed already
func (c *conn) Closed() bool {
	return c.isClosed
}

func (c *conn) Close() error {
	c.isClosed = true
	return c.Conn.Close()
}
