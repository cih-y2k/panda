package main

import (
	"fmt"
	"github.com/greekdev/panda"
	"github.com/greekdev/panda/teststd/shared"
	"strconv"
	"time"
)

var users = make(map[int]shared.User)

func init() {
	// add some test users
	for i := 0; i < 20; i++ {
		users[i] = shared.User{Username: "greekdev" + strconv.Itoa(i)}
	}
}

func main() {
	engine := panda.NewEngine()

	s := panda.NewServer(engine)
	///TODO: hello with hello_struct, it returns the hello only, so it gets the first matching letters, find a way to fix that *

	s.Handle("hello_struct", func(c panda.Conn, args ...panda.Arg) (interface{}, error) {
		user := shared.User{}
		if len(args) == 0 {
			panic("expecting arguments but got zero!")
		}
		panda.DecodeResult(&user, args[0])
		//fmt.Printf("From hello_struct, decoded result: %#v. Full Argument: %#v", user, args[0])
		if user.ClientID == c.ID() {
			return "YOU", nil
		}
		return fmt.Sprintf("OTHER, ClientID: %d while Conn ID: %d\n", user.ClientID, c.ID()), nil
	})

	s.Handle("getUser", func(conn panda.Conn, args ...panda.Arg) (interface{}, error) {
		requestedID, _ := panda.DecodeInt(args[0])
		for k, v := range users {
			if k == int(requestedID) {
				return v, nil
			}
		}
		return nil, fmt.Errorf("Error while handle the getUser with ID: %s", args[0])
	})

	// test non struct returns
	s.Handle("getYear", func(c panda.Conn, args ...panda.Arg) (interface{}, error) {
		return time.Now().Year(), nil
	})

	go func() {
		time.Sleep(4 * time.Second)
		data, err := s.Do(1, "getWorkingDir") // the connection's id
		if err != nil {
			panic(err)
		}
		fmt.Printf("getWorkingDir from client: %s\n", data)
	}()

	s.Handle("requestClientHello", func(c panda.Conn, args ...panda.Arg) (interface{}, error) {
		//callFromConnID := int(args[0].(float64))
		// let's send a result from the client connection's statement handler itself!
		fmt.Printf("Server: requestClientHello ID %d\n", c.ID())
		return s.Do(c.ID(), "helloClient")
	})

	panic(s.ListenAndServe("tcp4", "127.0.0.1:25"))
}
