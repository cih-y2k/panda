package panda

import (
	"bufio"
	"bytes"
	"net"
	"sync"
)

// CID the type of the connection ID, currently int
type CID uint64

type (
	// Response the response's channel, aka result. It's exported because its used on Async requests
	Response chan responsePayload
	// Request the request's channel, aka question. It's not exported :)
	request chan requestPayload
)

// Conn the connection, both server and client at the same time
type Conn struct {
	net.Conn
	id     CID
	engine *Engine
	mu     sync.Mutex
	// client
	outcomingReq     request
	incomingRes      Response
	pending          map[string]Response // request id: responsePayload channel
	cancelAllPending chan struct{}
	// server
	incomingReq request

	stopCh   chan struct{}
	reqPool  sync.Pool
	isClosed bool
}

// ID returns the connection's (incremental) id
func (c *Conn) ID() CID {
	return c.id
}

// setID used only by client to set its connection's ID which is coming from the server with the ack request and response payload
func (c *Conn) setID(id int) {
	c.id = CID(id)
}

func (c *Conn) serve() {
	go c.startClient()
	go c.startServer()
	c.reader()
}

func (c *Conn) reader() {
	defer func() {
		c.stopCh <- struct{}{}
	}()
	scanner := bufio.NewScanner(c)
	for scanner.Scan() {
		incomingData := scanner.Bytes()
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
			var resp responsePayload
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
			var req requestPayload
			err := c.engine.opt.codec.Deserialize(data, &req)
			if err != nil {
				return
			}
			c.incomingReq <- req
			// DEBUG: println("Sent")

		} else {

		}

	}

}

func (c *Conn) sendRequest(statement string, args Args, expectRaw bool) Response {
	response := make(Response, 1)
	c.outcomingReq <- requestPayload{
		ID:              RandomString(10),
		From:            c.ID(),
		Statement:       statement,
		Args:            args,
		ExpectRawResult: expectRaw,
		response:        response,
	}
	return response
}

func (c *Conn) startClient() {
	for {
		select {
		case <-c.stopCh:
			{
				break
			}
		case <-c.cancelAllPending:
			{
				for id, pending := range c.pending {
					// clean up any pending message by sending a canceled error Result
					pending <- canceledResponsePayload(id, "")
					close(pending)
					delete(c.pending, id)
				}
			}
		case out, ok := <-c.outcomingReq:
			{
				if !ok {
					return
				}
				data, err := c.engine.opt.codec.Serialize(out)
				if err != nil {
					out.response <- canceledResponsePayload(out.ID, err.Error()) // c.incomingRes should be buffered here, I don't want to change the flow.
					continue
				}

				c.pending[out.ID] = out.response
				c.Write(append(requestPrefixW, data...))
			}
		case in, ok := <-c.incomingRes:
			{
				if !ok {
					return
				}

				if in.RequestID == "" {
					continue
				}

				if pendingResp, found := c.pending[in.RequestID]; found {
					pendingResp <- in
					close(pendingResp)
					delete(c.pending, in.RequestID)
				}
			}
		}
	}
}

func (c *Conn) startServer() {
	for {
		select {

		case <-c.stopCh:
			{
				break
			}
		case in, ok := <-c.incomingReq:
			{
				if !ok {
					return
				} // edw na to allaksw apo acquireReuest se acquireContext mh kanw olo ta idia kai ta idia gt varieme,  p na exei sxesi me panda ?:P bamboo!
				req := c.acquireRequest(in.Statement, in.Args)
				req.ID = in.ID // send the correct request id
				defer c.releaseRequest(req)

				// we don't care if found or not, if an error must be sent to the client
				handlers := c.engine.handlers.find(req.Statement)
				req.handlers = handlers
				req.Serve()
				resp := responsePayload{RequestID: req.ID, Error: req.errMessage}

				if in.ExpectRawResult {
					if b, isBytes := req.result.([]byte); isBytes {
						resp.RawResult = b
					} else {
						resultData, serr := c.engine.opt.codec.Serialize(req.result)
						if serr != nil {
							c.engine.logf("Serialization failed for %#v on Connection with ID: %d, on Statement: %s", resp, c.ID(), req.Statement)
							return
						}
						resp.RawResult = resultData
					}
				} else {
					resp.Result = req.result // is decoded as general
				}

				//println("conn.go:224 AFTER execute handler with statement: " + req.Statement + " and requestID: " + req.ID)

				// first check for serialization errors, if and only here the error is not going back to the client for security reasons, but the server is notified
				data, serr := c.engine.opt.codec.Serialize(resp)
				if serr != nil {
					c.engine.logf("Serialization failed for %#v on Connection with ID: %d, on Statement: %s", resp, c.ID(), req.Statement)
					return
				}
				c.Write(append(responsePrefixW, data...))

			}
		}
	}
}

func (c *Conn) acquireRequest(statement string, args Args) *Request {
	req := c.reqPool.Get().(*Request)
	req.Statement = statement
	req.Args = args
	// fields From and conn are not changing*
	return req
}

func (c *Conn) releaseRequest(req *Request) {
	req.ID = ""
	// Dave Cheney says that making a new map is faster than deleting the previous, so:
	req.Args = make(Args)
	req.Statement = ""
	req.values.Reset()
	req.handlers = nil
	req.errMessage = ""
	req.pos = 0
	c.reqPool.Put(req)
}

var newLineBytes = []byte("\n")

func appendNewline(b []byte) []byte {
	return append(b, newLineBytes...)
}

func (c *Conn) Write(data []byte) (int, error) {
	return c.Conn.Write(appendNewline(data))
}

// Closed returns true if the connection has been closed already
func (c *Conn) Closed() bool {
	return c.isClosed
}

// Close closes the connection
// the cleanup goes on releaseConn
func (c *Conn) Close() error {
	c.isClosed = true
	return c.Conn.Close()
}
