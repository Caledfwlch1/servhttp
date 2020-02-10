package servhttp_test

import (
	"net/http"

	"github.com/caledfwlch1/servhttp"
)

func ExampleServHTTP_ServeAndShutdown() {
	srv := servhttp.New(":4443")

	srv.AddHandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("root OK"))
	})

	srv.AddHandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("new OK"))
	})

	srv.ServeAndShutdown("example.com")
}
