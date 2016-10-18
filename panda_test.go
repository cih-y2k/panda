package panda_test

import (
	//	"fmt"
	"github.com/greekdev/panda"
	"net/http"
	"net/http/httptest"
	//"sync"
	"testing"
	"time"
)

type testStruct struct {
	Name     string
	ClientID uint64
}

func BenchmarkSimpleOnlyHTTPRecorder(b *testing.B) {
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		b.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("Hello"))
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(rr, req)
		/*
			if status := rr.Code; status != http.StatusOK {
				b.Errorf("handler returned wrong status code: got %v want %v",
					status, http.StatusOK)
			}

			expected := `Hello`
			if rr.Body.String() != expected {
				b.Errorf("handler returned unexpected body: got %v want %v",
					rr.Body.String(), expected)
			}*/

	}
	b.StopTimer()
	b.ReportAllocs()
}

/*
func BenchmarkSimpleOnlyHTTPRaw(b *testing.B) {
	mux := http.NewServeMux()
	mux.Handle("/hello", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("Hello"))
	}))

	s := http.Server{Addr: "127.0.0.1:128", Handler: mux}
	// no http test recorder
	go s.ListenAndServe()
	client := &http.Client{}
	time.Sleep(200 * time.Millisecond) // http server takes longer to startup*

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Get("http://127.0.0.1:128/hello")
		if err != nil {
			b.Fatal(err)
		}
		//res.Body.Close()

	}
	b.StopTimer()
	b.ReportAllocs()
}*/

func BenchmarkHandleDo(b *testing.B) {
	srvEngine := panda.NewEngine(panda.OptionBuffer(uint64(b.N / 3)))
	srv := panda.NewServer(srvEngine)

	srv.Handle("hello", func(req *panda.Request) {
		req.Result("Hello")
	})

	clientEngine := panda.NewEngine(panda.OptionBuffer(uint64(b.N / 3)))
	client := panda.NewClient(clientEngine)
	defer func() {
		client.Close()
		srv.Close()
	}()
	go func() {
		if err := srv.ListenAndServe("tcp4", "127.0.0.1:128"); err != nil {
			panic(err)
		}
	}()
	time.Sleep(50 * time.Millisecond)
	err := client.Dial("tcp4", "127.0.0.1:128")
	if err != nil {
		b.Fatal(err)
	}

	//	wg := &sync.WaitGroup{}
	//wg.Add(b.N)
	//fmt.Printf("\nExecuting %d Operations", b.N)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		//	go func() {
		res, err := client.Do("hello")
		if err != nil {
			b.Fatal(err)
		}
		if res.(string) != "Hello" {
			b.Fatal("Expecting 'Hello' but got %#v", res)
		}
		//	wg.Done()
		//	}()

	}

	//	wg.Wait()

	b.StopTimer()
	b.ReportAllocs()

}

/*
func BenchmarkTwoWayHandleDo(b *testing.B) {
	srv := panda.NewServer(panda.NewEngine())

	srv.Handle("hello1_string_no_args", func(panda.Conn, ...panda.Arg) (interface{}, error) {
		//println("do 1")
		return "Hello", nil
	})

	srv.Handle("hello2_string", func(c panda.Conn, args ...panda.Arg) (interface{}, error) {
		//	println("do 2")
		if len(args) == 0 {
			return nil, fmt.Errorf("expecting arguments on hello_struct_no_args but got zero!")
		}
		return "Hello " + args[0].(string), nil
	})

	srv.Handle("hello3_struct_no_args", func(c panda.Conn, args ...panda.Arg) (interface{}, error) {
		//	println("do 3")

		ts := testStruct{ClientID: c.ID(), Name: "something"}
		return ts, nil
	})

	srv.Handle("hello4_struct", func(c panda.Conn, args ...panda.Arg) (interface{}, error) {
		//	println("do 4")
		user := testStruct{}
		if len(args) == 0 {
			return nil, fmt.Errorf("expecting arguments on hello4_struct but got zero!")
		}

		panda.DecodeResult(&user, args[0])
		if user.ClientID == c.ID() {
			return "YOU", nil
		}
		return fmt.Sprintf("OTHER, ClientID: %d while Conn ID: %d", user.ClientID, c.ID()), nil
	})

	client := panda.NewClient(panda.NewEngine())

	go func() {
		if err := srv.ListenAndServe("tcp4", "127.0.0.1:125"); err != nil {
			panic(err)
		}
	}()
	time.Sleep(1 * time.Second)
	err := client.Dial("tcp4", "127.0.0.1:125")
	if err != nil {
		b.Fatal(err)
	}

	defer func() {
		// if srv and client doesn't ends the benchmark
		client.Close()
		srv.Close()
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		_, err := client.Do("hello1_string_no_args")
		if err != nil {
			b.Fatal(err)
		}
		_, err = client.Do("hello2_string", "kataras")
		if err != nil {
			b.Fatal(err)
		}
		_, err = client.Do("hello3_struct_no_args")
		if err != nil {
			b.Fatal(err)
		}

		res, err := client.Do("hello4_struct", testStruct{ClientID: client.Conn().ID(), Name: "kataras"})
		if err != nil {
			b.Fatal(err)
		} else if res.(string) != "YOU" {
			b.Fatalf("Expecting YOU but got %s. **Local Client ID: %d\n", res, client.Conn().ID())
		}
	}
	b.StopTimer()
	b.ReportAllocs()
	client.Close()
	srv.Close()

}*/
