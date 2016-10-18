package main

import (
	"fmt"
	"github.com/greekdev/panda"
	"github.com/greekdev/panda/teststd/shared"
	"os"
	"time"
)

func main() {
	s := panda.NewClient(panda.NewEngine())
	s.Dial("tcp4", "127.0.0.1:25")

	fmt.Printf("Client is ready with conn ID: %d\n", s.Conn().ID())
	now := time.Now()
	// test client-side events
	s.Handle("getWorkingDir", func(panda.Conn, ...panda.Arg) (interface{}, error) {
		return os.Getwd()
	})

	// test act like server  2
	s.Handle("helloClient", func(c panda.Conn, args ...panda.Arg) (interface{}, error) {
		// because it's client-side acts like server, the connection will be always to this connection itself
		// unless the server acts like a bridge between client-to-client communication, you can do that with server side: .Do(GetConn(2),"getUser",...)
		return fmt.Sprintf("Hello from connection with ID: %d\n", c.ID()), nil
	})

	msg, err := s.Do("hello_struct", shared.User{ClientID: s.Conn().ID(), Username: "kataras"})
	if err != nil {
		panic(err)
	} else if msg.(string) != "YOU" {
		fmt.Printf("Expecting YOU but got %s. **Local Client ID: %d\n", msg, s.Conn().ID())
	}

	// test async
	for arg := 10; arg < 20; arg++ {

		ch := s.DoAsync("getUser", arg)
		go func(ch panda.Response) {
			time.Sleep(150 * time.Millisecond) // ONLY FOR VISUALIZE THE DELAY
			response := <-ch

			if response.Error != "" {
				fmt.Printf("Error on client's DoAsync getUser: %s\n", response.Error)
			}

			user := &shared.User{}
			response.Decode(user)
			fmt.Printf("Receive user with Username: %s\n", user.Username)
		}(ch)

	}

	for arg := 0; arg < 10; arg++ {

		user := panda.Expect(&shared.User{}, s.Do)("getUser", arg).(*shared.User)
		/*	res, err := s.Do("getUser", arg)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Receive: %#v\n", res)*/
		fmt.Printf("User %s received (2nd)\n", user.Username)
		time.Sleep(50 * time.Millisecond) // ONLY FOR VISUALIZE THE DELAY
	}

	// test act like server 1
	s.Handle("do not do that", func(c panda.Conn, args ...panda.Arg) (interface{}, error) {
		time.Sleep(time.Duration(panda.MustDecodeInt(args[0])) * time.Second) // to simiulate the 'local' delay
		return shared.User{Username: "DONT DO THIS"}, nil
	})

	// async local which acts like SYNC for your own good ( on remote it's really async works the same for remote)
	// it doesn't sends to the server, it's checking local methods first , it's working but do not do that:)
	channel := s.DoAsync("do not do that", 2) //
	go func() {
		response := <-channel
		fmt.Printf("Got the 'Do not do this' result async : %#v\n", response.Data)
	}()
	// the same but sync
	// it doesn't sends to the server, it's checking local methods first , it's working but do not do that:)
	res, err := s.Do("do not do that", 3)
	fmt.Printf("Sync: Do not do this, result: %#v\n", res)
	fmt.Printf("Sync: Do not do this, error: %v\n", err)

	// so they should received the same time almost
	// 5 secs in order to not exit
	//time.Sleep(time.Duration(5) * time.Second)

	// test non- structs returns
	data, err := s.Do("getYear")
	if err == nil {
		fmt.Printf("Got server's year: %d\n", panda.MustDecodeInt(data))
	}

	//TODO: here it blocks while server receives the request message...
	hello, _ := s.Do("requestClientHello")
	fmt.Printf("From hello: %s\n", hello)

	fmt.Printf("\n\nTOOK:\n%d", time.Now().Sub(now))
}
