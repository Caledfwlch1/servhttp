package servhttp_test

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/caledfwlch1/servhttp"
)

func ExampleServHTTP_ServeAndShutdown() {
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	srv := servhttp.New(logger, ":4443", time.Minute)

	srv.AddHandler("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	srv.ServeAndShutdown()
}
