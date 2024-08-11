package internalhttp

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type Server struct {
	server *http.Server
	logger Logger
}

type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

type Application interface {
	Health(ctx context.Context) []byte
	Version(ctx context.Context) []byte
	WS(ctx context.Context, conn *websocket.Conn)
}

func New(logger Logger, app Application, host string, port int) *Server {
	servers := &http.Server{
		Addr:         net.JoinHostPort(host, strconv.Itoa(port)),
		Handler:      NewHandler(logger, app),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &Server{
		server: servers,
		logger: logger,
	}
}

func (s *Server) Start(ctx context.Context) error {
	err := s.server.ListenAndServe()
	if err != nil {
		return err
	}

	<-ctx.Done()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
