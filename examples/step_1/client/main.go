package main

import (
	"github.com/geekypanda/panda"
	"github.com/geekypanda/panda/examples/shared"
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
	var args = panda.Args{"id": 0}
	for i := 1; i <= 50; i++ {
		args["id"] = i
		result, err := client.Do("getUser", args)
		if err != nil {
			logger.Println("Error on getUser: " + err.Error())
			continue
		}
		// optionally, decode the result from map[string]interface{} to struct User
		user := &shared.User{}
		panda.DecodeResult(user, result)

		logger.Println("User firstname: " + user.Firstname)
	}

	logger.Println("Closing connection")
	client.Close()

}
