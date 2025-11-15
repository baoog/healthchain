package server

import (
	"consentcc/config"
	"context"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"time"
)

type Server struct {
	e    *echo.Echo
	addr string
}

type EchoServer interface {
	Echo() *echo.Echo
	StartAsync()
	Shutdown(ctx context.Context) error
}

func New(cfg *config.Config) *Server {
	e := echo.New()
	s := &Server{e: e, addr: cfg.HTTPAddr}
	return s
}

func (s *Server) Echo() *echo.Echo { return s.e }

func (s *Server) StartAsync() {
	go func() {
		log.Printf("HTTP listening on %s", s.addr)
		if err := s.e.Start(s.addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http start: %v", err)
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.e.Shutdown(ctx)
}

func GracefulWait(ctx context.Context, cancel func()) {
	<-ctx.Done()
	cancel()
	time.Sleep(200 * time.Millisecond)
}
