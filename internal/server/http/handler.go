package internalhttp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

type handler struct {
	logger Logger
	app    Application
}

func NewHandler(logger Logger, app Application) http.Handler {
	h := &handler{
		logger: logger,
		app:    app,
	}

	r := mux.NewRouter()
	r.HandleFunc("/health", h.Health).Methods(http.MethodGet)
	r.HandleFunc("/sig/v1/version", h.Version).Methods(http.MethodGet)
	r.Handle("/sig/v1/rtc", websocket.Handler(h.RTC))
	r.MethodNotAllowedHandler = http.HandlerFunc(methodNotAllowedHandler)
	r.NotFoundHandler = http.HandlerFunc(methodNotFoundHandler)

	r.Handle("/sig/v1/rtc", websocket.Handler(func(c *websocket.Conn) {}))
	return r
}

func (s *handler) Health(w http.ResponseWriter, r *http.Request) {
	response := s.app.Health(r.Context())

	_, err := w.Write(response)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Health - response error: %s", err))
	}
}

func (s *handler) Version(w http.ResponseWriter, r *http.Request) {
	response := s.app.Version(r.Context())

	_, err := w.Write(response)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Version - response error: %s", err))
	}
}

func (s *handler) RTC(c *websocket.Conn) {
	s.app.RTC(context.Background(), c)
}

func methodNotAllowedHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
}

func methodNotFoundHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "404 Not Found", http.StatusNotFound)
}
