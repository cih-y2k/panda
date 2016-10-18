package main

import (
	"fmt"
	"github.com/greekdev/panda"
	"github.com/greekdev/panda/http"
	"log"
	"net"
)

func main() {
	// panda server without listening
	srv := panda.NewServer(panda.NewEngine())
	srv.Handle("mypath", func(c panda.Conn, args ...panda.Arg) (interface{}, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("You should pass name argument! (for http: /mypath?name=greekdev")
		}
		name := args[0].(string) // /mypath?name=myname or /mypath?name=Gerasimos,Maropoulos
		return "Hello " + name, nil
	})

	// now http

	ln, err := net.Listen("tcp4", "127.0.0.1:8080")
	if err != nil {
		panic(err)
	}
	log.Fatal(http.Serve(ln, srv))
}
