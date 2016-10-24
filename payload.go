package panda

import (
	"fmt"
)

var (
	requestPrefix   = []byte("REQ")
	responsePrefix  = []byte("RES")
	prefixLen       = 3
	whiteSpaceBytes = []byte(" ")
	requestPrefixW  = append(requestPrefix, whiteSpaceBytes...)
	responsePrefixW = append(responsePrefix, whiteSpaceBytes...)
)

type (
	// client -> server
	requestPayload struct {
		// the unique request's id which waits for a Result with the same RequestID, may empty if not waiting for Result.
		// Channel is just a non-struct methodology for request-Result-reqsponse-request communication,
		// its id made by client before sent to the server, the same id is used for the server's Result
		ID                string
		From              CID    // connection id,
		Statement         string // the call statement
		Args              Args   // the statement's method's arguments, if any
		SkipSerialization bool   // if true then serialization is not applied for the response
		response          Response
	}

	// server -> client , it's the result of the requestPayload's statement handler
	responsePayload struct {
		RequestID string // the request id, maybe empty if it's not created to answer for a request from the client, it's made by client
		// the deserialized result, which is map when default json codec is used,
		// but I am not explicit set it as map[string]interface{} because your custom codec may differs
		// becaue of that we have a .To function which will convert this , as map to a struct
		// error if any from the server for this particular request's Result
		// If it's a struct then it's a map[string]interface{}, json ready-to-use. If it's int then it's float64, all other standar types as they are.
		Data   []byte      `json:",omitempty"` //json.RawMessage // if request's SkipSerialization is true then this is filled and result is nil
		Result interface{} `json:",omitempty"` // if request's SkipSerialization is false, then this is filled by decoded handler's result
		Error  string      // error type cannot be json-encoded/decoded so it's string but handlers returns error as user expects
		//Raw   []byte      `json:"-"` // if requestPayload's SkipSerialization is true, then this interface{} its 100% raw []byte
	}
)

// Decode writes the 'Result', which should be a map[string]interface{} if receiver expected a custom type struct,
// to the vPointer which should be a custom type of go struct.
//
// Note: it's useless if you wanna re-send this to your http api
func (r responsePayload) Decode(vPointer interface{}) {
	if r.Data != nil {
		DecodeResult(vPointer, r.Data)
	}
}

// Canceled TODO:
type Canceled struct {
	reason string
}

func (c Canceled) Error() (s string) {
	s = "Canceled"
	if c.reason != "" {
		s += " reason: " + c.reason
	}
	return
}

// Invalid TOOD:
type Invalid struct {
	conn *Conn
}

func (i Invalid) Error() string {
	return fmt.Sprintf("Invalid response, on connection: '%#v'", i.conn) // conn should be nil but for any case print it
}

func canceledResponsePayload(reqID string, reason string) responsePayload {
	return responsePayload{
		RequestID: reqID,
		Error:     Canceled{reason}.Error(),
	}
}

func invalidResponsePayload(conn *Conn) responsePayload {
	return responsePayload{
		Error: Invalid{conn}.Error(),
	}
}
