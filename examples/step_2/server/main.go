package main

import (
	"github.com/geekypanda/panda"
	"log"
)

func main() {
	server := panda.NewServer(panda.NewEngine())

	server.OnConnection(func(c *panda.Conn) {
		log.Printf("Connection has connected!")
		// get the client current working directory path
		res, err := server.Do(c, "getWorkingDir")
		if err != nil {
			log.Printf("Error: %s", err)
			return
		}
		log.Printf("Connection with ID: %d working dir path: %s", c.ID(), res.(string))
	})

	log.Fatal(server.ListenAndServe("tcp4", "127.0.0.1:2222"))
}
