package main

import (
	"github.com/geekypanda/panda"
	"net/http"
	"strconv"
)

const remote = "127.0.0.1:2222"

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(res http.ResponseWriter, req *http.Request) {
		c := panda.NewClient(panda.NewEngine())
		c.Connect(remote)
		params := req.URL.Query()
		if params != nil {
			idStr := params.Get("id")
			if idStr != "" {
				id, serr := strconv.Atoi(idStr)
				if serr != nil {

					http.Error(res, serr.Error(), http.StatusServiceUnavailable)
					return
				}
				data, err := c.DoRaw("getUser", id)
				if err == nil {
					res.Header().Set("Content-Type", "application/json; charset=utf-8")
					res.WriteHeader(200)

					res.Write(data)

					return
				}
			}

		}
		http.NotFound(res, req)

	})
	srv := &http.Server{Addr: "127.0.0.1:80", Handler: mux}
	srv.ListenAndServe()
}
