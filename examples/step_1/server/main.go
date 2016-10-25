package main

import (
	"github.com/geekypanda/panda"
	"github.com/geekypanda/panda/examples/shared"
	"log"
	"os"
)

var (
	users = make(map[int]*shared.User)
)

func init() {
	for i := 1; i <= 50; i++ {
		users[i] = shared.NewTestUser(i)
	}
}

func main() {
	logger := log.New(os.Stdout, "SERVER ", log.LstdFlags)
	engine := panda.NewEngine(panda.OptionLogger(logger))

	server := panda.NewServer(engine)
	server.Begin(panda.ArgsLen(0, 1))

	server.Handle("getUser", func(req *panda.Request) {

		id := req.Args.Int("id")

		user, found := users[id]
		if !found {
			req.Error("User with ID: %d not found!", id)
			return
		}

		req.Result(user)
	})

	log.Fatal(server.ListenAndServe("tcp4", "127.0.0.1:2222"))
}
