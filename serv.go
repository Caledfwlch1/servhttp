// The servhttp package simplifies the organization of starting and shutting down an HTTP server.
// This package automatically receives Let's Encrypt certificates.
// For more information, visit https://letsencrypt.org.
package servhttp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// HTTP server type with a logger.
type ServHTTP struct {
	*log.Logger
	*http.Server

	authFunc              func(r *http.Request) bool
	redirectUrl           string
	router                *http.ServeMux
	timeoutServerShutdown time.Duration
}

// This function creates a new HTTP server with an empty handler.
func New(logger *log.Logger, listenAddr string, timeoutShutdown time.Duration) *ServHTTP {
	router := http.NewServeMux()
	return &ServHTTP{
		Logger: logger,
		Server: &http.Server{
			Addr:         listenAddr,
			Handler:      router,
			ErrorLog:     logger,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
		router:                router,
		timeoutServerShutdown: timeoutShutdown,
	}
}

// Adds new handler to the ServHTTP
func (s *ServHTTP) AddHandler(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.router.HandleFunc(pattern, handler)
}

// AddAuthFunc adds middleware authentication.
func (s *ServHTTP) AddAuthFunc(f func(r *http.Request) bool, redirectUrl string) {
	s.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !f(r) {
			http.Redirect(w, r, redirectUrl, http.StatusTemporaryRedirect)
			return
		}
		// Assuming authentication passed, run the original handler
		//s.Handler.ServeHTTP(w, r)
	})
}

// Graceful shutdown method
func (s *ServHTTP) Shutdown(cancel context.CancelFunc, stop <-chan error) error {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-stop:
		cancel()
		return fmt.Errorf("listen: %v\n", err)
	case <-quit:
	}

	s.Println("server shutdown ...")
	cancel()

	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), s.timeoutServerShutdown)
	defer cancelTimeout()

	if err := s.Server.Shutdown(ctxTimeout); err != nil {
		return fmt.Errorf("server shutdown: %s", err)
	}

	s.Println("server shut down")
	return nil
}

// ServeAutoCert runs http.ListenAndServe if the domains slice is empty.
// Otherwise, it runs http.ListenAndServeTLS with a list of domains using Let's Encrypt.
func (s *ServHTTP) ServeAutoCert(domains ...string) error {
	if len(domains) == 0 {
		return s.ListenAndServe()
	}

	m := &autocert.Manager{
		Cache:      autocert.DirCache("golang-autocert"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domains...),
	}

	s.Server.TLSConfig = m.TLSConfig()

	return s.ListenAndServeTLS("", "")
}

// This method combines the methods of ServeAutoCert and Shutdown.
func (s *ServHTTP) ServeAndShutdown(domains ...string) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan error, 1)

	go func() {
		stop <- s.ServeAutoCert(domains...)
	}()

	s.Println("server started")

	if err := s.Shutdown(cancel, stop); err != nil {
		s.Fatalln(err)
	}
}
