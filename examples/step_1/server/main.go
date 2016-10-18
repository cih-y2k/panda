package main

import (
	"github.com/greekdev/panda"
	"github.com/greekdev/panda/examples/step_1/shared"
	"log"
	"os"
)

var (
	users = make(map[int]*shared.User)
)

func init() {
	for i := 0; i <= 50; i++ {
		users[i] = shared.NewTestUser(i)
	}
}

func main() {
	logger := log.New(os.Stdout, "SERVER ", log.LstdFlags)
	engine := panda.NewEngine(panda.OptionLogger(logger))

	server := panda.NewServer(engine)
	server.Begin(panda.ArgsLen(0, 1))

	server.Handle("getUser", func(req *panda.Request) {

		id := req.Args.Int(0)

		user, found := users[id]
		if !found {
			req.Error("User with ID: %d not found!", id)
			return
		}

		req.Result(user)
	})

	log.Fatal(server.ListenAndServe("tcp4", "127.0.0.1:2222"))
}
