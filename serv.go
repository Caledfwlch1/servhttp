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

	// Server stop channel.
	Stop chan error

	authFunc        func(r *http.Request) bool
	redirectUrl     string
	router          *http.ServeMux
	timeoutShutdown time.Duration
}

// This function creates a new HTTP server with an empty handler.
func New(listenAddr string) *ServHTTP {
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
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
		router:          router,
		Stop:            make(chan error, 1),
		timeoutShutdown: time.Minute,
	}
}

// AddHandle registers the handler for the given pattern in the ServHTTP.
func (s *ServHTTP) AddHandle(pattern string, handler http.Handler) {
	s.router.Handle(pattern, handler)
}

// Adds a new handler to the ServHTTP for the given pattern.
func (s *ServHTTP) AddHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.router.HandleFunc(pattern, handler)
}

// AddAuthFunc adds middleware authentication.
func (s *ServHTTP) AddAuthFunc(f func(r *http.Request) bool, redirectUrl string) {
	handler := s.Handler
	s.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != redirectUrl && !f(r) {
			http.Redirect(w, r, redirectUrl, http.StatusTemporaryRedirect)
			return
		}
		// Assuming authentication passed, run the original handler
		handler.ServeHTTP(w, r)
	})
}

// Using the Config command, you can change the server settings.
// If you want to set one of the parameters, you can set the second with a default value for the corresponding type.
// Example:
// srv.Config(nil, time.Minute * 2) sets only the server timeout
// or
// srv.Config(logger, 0) sets only logger
func (s *ServHTTP) Config(logger *log.Logger, timeout time.Duration) {
	if logger != nil {
		s.Logger = logger
	}

	if timeout != 0 {
		s.timeoutShutdown = timeout
	}
}

// Graceful shutdown method.
func (s *ServHTTP) Shutdown() error {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-s.Stop:
		return fmt.Errorf("listen: %v\n", err)
	case <-quit:
	}

	s.Println("server shutdown ...")

	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), s.timeoutShutdown)
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
	go func() {
		s.Stop <- s.ServeAutoCert(domains...)
	}()

	s.Println("server started")

	if err := s.Shutdown(); err != nil {
		s.Fatalln(err)
	}
}
