package main

import (
	"github.com/geekypanda/panda"
	"net/http"
)

const remote = "127.0.0.1:2222"

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(res http.ResponseWriter, req *http.Request) {
		c := panda.NewClient(panda.NewEngine())
		c.Connect(remote)
		id := req.URL.Query().Get("id")
		data, err := c.DoRaw("getUser", id)
		if err == nil {
			res.Header().Set("Content-Type", "application/json; charset=utf-8")
			res.WriteHeader(200)

			res.Write(data)

			return
		}
		http.Error(res, err.Error(), http.StatusServiceUnavailable)

	})
	srv := &http.Server{Addr: "127.0.0.1:80", Handler: mux}
	srv.ListenAndServe()
}
