package internalhttp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type handler struct {
	logger Logger
	app    Application
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

func NewHandler(logger Logger, app Application) http.Handler {
	h := &handler{
		logger: logger,
		app:    app,
	}

	r := mux.NewRouter()
	r.HandleFunc("/health", h.Health).Methods(http.MethodGet)
	r.HandleFunc("/version", h.Version).Methods(http.MethodGet)
	r.HandleFunc("/sig/v1/rtc", h.RTC)
	r.MethodNotAllowedHandler = http.HandlerFunc(methodNotAllowedHandler)
	r.NotFoundHandler = http.HandlerFunc(methodNotFoundHandler)

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

func (s *handler) RTC(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error(fmt.Sprintf("RTC - response error: %s", err))
		return
	}

	s.logger.Info(fmt.Sprintf("Serve client %v at %v", r.RemoteAddr, r.RequestURI))

	s.app.RTC(context.Background(), conn)
}

func methodNotAllowedHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
}

func methodNotFoundHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "404 Not Found", http.StatusNotFound)
}
