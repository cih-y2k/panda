package main

import (
	"github.com/geekypanda/panda"
	//"github.com/geekypanda/panda/examples/shared"
	"log"
	"os"
)

func main() {
	logger := log.New(os.Stdout, "CLIENT ", log.LstdFlags)
	engine := panda.NewEngine(panda.OptionLogger(logger))

	client := panda.NewClient(engine)

	err := client.Dial("tcp4", "127.0.0.1:2222")
	if err != nil {
		logger.Fatalf("Error while connecting to the server, trace: %s", err)
		println("\n\n")
		return
	}
	// waits until connected

	data, err := client.DoRaw("getUser", panda.Args{"id": 1})
	if err != nil {
		logger.Println("Error on getUser: " + err.Error())

	} else {
		logger.Printf("Raw data: %#v", data)
	}

	logger.Println("Closing connection")
	client.Close()

}
