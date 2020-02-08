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

type ServHTTP struct {
	*log.Logger
	*http.Server

	timeoutServerShutdown time.Duration
}

type HandlerMap map[string]func(http.ResponseWriter, *http.Request)

func New(logger *log.Logger, listenAddr string, timeoutShutdown time.Duration) *ServHTTP {
	return &ServHTTP{
		Logger: logger,
		Server: &http.Server{
			Addr:         listenAddr,
			ErrorLog:     logger,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
		timeoutServerShutdown: timeoutShutdown,
	}
}

func NewHandler() HandlerMap {
	return make(map[string]func(http.ResponseWriter, *http.Request))
}

func (h HandlerMap) Add(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	if h == nil {
		h = NewHandler()
	}
	h[pattern] = handler
}

func (s *ServHTTP) Shutdown(cancel context.CancelFunc, stop <-chan error) error {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-stop:
		cancel()
		return fmt.Errorf("listen: %v\n", err)
	case <-quit:
	}

	s.Println("Shutdown Server ...")
	cancel()

	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), s.timeoutServerShutdown)
	defer cancelTimeout()

	if err := s.Server.Shutdown(ctxTimeout); err != nil {
		return fmt.Errorf("server shutdown: %s", err)
	}

	s.Println("Server exiting")
	return nil
}

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

func (s *ServHTTP) ServeAndShutdown(domains ...string) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan error, 1)

	go func() {
		stop <- s.ServeAutoCert(domains...)
	}()

	if err := s.Shutdown(cancel, stop); err != nil {
		s.Fatalln(err)
	}
}

func (s *ServHTTP) HandlersRegistration(handlers map[string]func(http.ResponseWriter, *http.Request)) {
	router := http.NewServeMux()

	for pattern, handler := range handlers {
		router.HandleFunc(pattern, handler)
	}
	s.Handler = router
}
