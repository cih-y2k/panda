package main

import (
	"fmt"
	"github.com/greekdev/panda"
	"github.com/greekdev/panda/examples/step_1/shared"
	"log"
	"os"
	"sync"
)

var (
	users = make(map[int]*shared.User)
	mu    sync.Mutex
)

func init() {
	for i := 0; i <= 50; i++ {
		users[i] = shared.NewTestUser(i)
	}
}

func main() {
	logger := log.New(os.Stdout, "SERVER", log.LstdFlags)
	engine := panda.NewEngine(panda.OptionLogger(logger))

	server := panda.NewServer(engine)

	server.Handle("getUser", func(c panda.Conn, args ...panda.Arg) (interface{}, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("Args are missing")
		}
		id := panda.MustDecodeInt(args[0])
		return getUser(id)
	})

	log.Fatal(server.ListenAndServe("tcp4", "127.0.0.1:2222"))
}

func getUser(id int) (*shared.User, error) {
	mu.Lock()
	defer mu.Unlock()
	user, found := users[id]
	if !found {
		return nil, fmt.Errorf("User with ID: %d not found!", id)
	}
	return user, nil
}
