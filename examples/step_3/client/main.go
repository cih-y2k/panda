package main

import (
	"github.com/geekypanda/panda"
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

	// this should be fail
	result, err := client.Do("getUser", panda.Args{"id": 1, "invalidNumberofArgs": 2})
	if err != nil {
		logger.Println("Error on getUser: " + err.Error())
	} else {
		logger.Printf("User %#v: ", result)
	}

	// this should be ok
	result, err = client.Do("getUser", panda.Args{"id": 1})
	if err != nil {
		logger.Println("Error on getUser: " + err.Error())
	} else {
		logger.Printf("User %#v: ", result)
	}

	logger.Println("Closing connection")
	client.Close()

}
