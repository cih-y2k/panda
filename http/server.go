package http

import (
	"encoding/json"
	"github.com/greekdev/panda"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// implement the panda.Conn but for the http, which ID and CancelAllPending is not needed
type conn struct {
	net.Conn
	isClosed bool
}

func (c *conn) ID() int {
	return 0
}
func (c *conn) CancelAllPending() {

}

func (c *conn) Closed() bool {
	return c.isClosed
}
func (c *conn) Close() error {
	c.isClosed = true
	return c.Conn.Close()
}

// Serve serves panda server in http mode
func Serve(ln net.Listener, srv *panda.Server) error {

	hij := func(res http.ResponseWriter, req *http.Request) {
		hj, ok := res.(http.Hijacker)
		if !ok {
			http.Error(res, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		netConn, buf, err := hj.Hijack()
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		pandaC := &conn{Conn: netConn}

		defer func() {
			buf.Flush()
			defer pandaC.Close()
		}()
		statement := req.URL.EscapedPath()

		if statement == "/favicon.ico" {
			return
		}

		var args []panda.Arg
		// arguments are passed with url parameters
		urlParams := req.URL.Query()
		for _, v := range urlParams {
			argV := v[0]
			if len(v) > 1 {
				argV = strings.Join(v, ",")
			}

			// here pass one argument , and if v is len >0 then we join them with ","
			// this is done only to be compatible with the rest of the user's external panda API

			args = append(args, argV)
		}

		//println("http/server.go:31: Received statement: " + statement)

		result, err := srv.Exec(pandaC, statement, args...)

		if err != nil {
			println(err.Error())
			buf.WriteString(err.Error())
			return
		}

		if result == nil {
			buf.WriteString("Error: Empty result")
			return
		}

		// check for type, if no standard type then json it

		if s, ok := result.(string); ok {
			buf.WriteString(s)
		} else if i, ok := result.(int); ok {
			buf.WriteString(strconv.Itoa(i))
		} else if b, ok := result.(bool); ok {
			buf.WriteString(strconv.FormatBool(b))
		} else if by, ok := result.([]byte); ok {
			buf.Write(by)
		} else {
			//we suppose is json
			data, err := json.Marshal(result)
			if err != nil {
				buf.WriteString(err.Error())
				return
			}
			buf.Write(data)
		}

	}

	handler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		req.Header.Set("Content-Type", "application/json ;charset=utf-8") ///TODO: make a cleanup and a way to customize that
		hij(res, req)
	})

	hsrv := &http.Server{Handler: handler}

	return hsrv.Serve(ln)
}
