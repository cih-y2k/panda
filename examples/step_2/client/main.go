package main

import (
	"github.com/geekypanda/panda"
	"log"
	"os"
)

func main() {

	client := panda.NewClient(panda.NewEngine())

	// yes, you can use the client connection as server too, panda designed to work nice with dublex communication
	// NOTE: if you're gonna to call immediatly a function from client then you MUST register the handlers before .Dial, same as with Server before its ListenAndServe
	client.Handle("getWorkingDir", func(req *panda.Request) {
		dir, err := os.Getwd()
		if err != nil {
			req.Error(err.Error())
			return
		}

		req.Result(dir)
	})

	err := client.Dial("tcp4", "127.0.0.1:2222")
	if err != nil {
		log.Fatalf("Error while connecting to the server, trace: %s", err)
		return
	}

	log.Println("Closing connection")
	client.Close()

}
