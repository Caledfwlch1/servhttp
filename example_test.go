package servhttp_test

import (
	"github.com/caledfwlch1/servhttp"
	"log"
	"net/http"
	"os"
	"time"
)

func ExampleServHTTP_ServeAndShutdown() {
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	srv := servhttp.New(logger, ":4443", time.Minute)

	handlers := servhttp.NewHandler()
	handlers.Add("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	srv.HandlersRegistration(handlers)

	srv.ServeAndShutdown("example.com")
}
