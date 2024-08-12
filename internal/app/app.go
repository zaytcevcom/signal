package app

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ossrs/go-oryx-lib/errors"
	"github.com/ossrs/go-oryx-lib/logger"
)

type App struct {
	logger     Logger
	rooms      sync.Map // todo: тут не нужно типизировать?
	emptyRooms chan string
}

type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

var handlers map[string]ActionHandler

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 30 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

func init() {
	handlers = map[string]ActionHandler{
		"accept":      handleAccept,
		"decline":     handleDecline,
		"publish":     handlePublish,
		"changeState": handleChangeState,
		"inviteUsers": handleInviteUsers,
	}
}

func New(logger Logger) *App {
	a := &App{
		logger:     logger,
		emptyRooms: make(chan string),
	}

	// todo: все ок с местом запуска горутины?
	ctx, cancel := context.WithCancel(context.Background())
	go a.manageRooms(ctx, cancel)

	return a
}

func (a *App) Health(_ context.Context) []byte {
	return []byte("OK")
}

func (a *App) Version(_ context.Context) []byte {
	return []byte("1.0.1")
}

// WS todo: можно ли тут знать о *websocket.Conn ?
func (a *App) WS(ctx context.Context, conn *websocket.Conn) {
	ctx, cancel := context.WithCancel(logger.WithContext(ctx))
	defer a.closeConnection(ctx, cancel, conn)

	a.heartbeat(ctx, cancel, conn)

	inMessages := make(chan []byte)
	go a.handleInMessages(ctx, cancel, conn, inMessages)

	preconnectMessages := make(chan []byte)
	outMessages := make(chan []byte)
	go a.handleOutMessages(ctx, cancel, inMessages, preconnectMessages, outMessages)

	for {
		select {
		case <-ctx.Done():
		case m := <-preconnectMessages:
			if err := conn.WriteMessage(websocket.TextMessage, m); err != nil {
				logger.Wf(ctx, "[WS preconect] Ignore err %v for %v", err, conn.RemoteAddr())
				break
			}
		case m := <-outMessages:
			if err := conn.WriteMessage(websocket.TextMessage, m); err != nil {
				logger.Wf(ctx, "[WS main] Ignore err %v for %v", err, conn.RemoteAddr())
				break
			}
		}
	}
}

func (a *App) manageRooms(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case roomID := <-a.emptyRooms:
			a.rooms.Delete(roomID)
		}
	}
}

func (a *App) closeConnection(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn) {
	err := conn.Close()
	if err != nil {
		logger.E(ctx, err.Error())
	}

	cancel()
}

func (a *App) heartbeat(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn) {
	err := conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		logger.E(ctx, err.Error())
		return
	}

	conn.SetPongHandler(func(string) error {
		err := conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			logger.E(ctx, err.Error())
			return err
		}
		return nil
	})

	ticker := time.NewTicker(pingPeriod)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
					cancel()
					return
				}
			}
		}
	}()
}

func (a *App) handleInMessages(
	ctx context.Context,
	cancel context.CancelFunc,
	conn *websocket.Conn,
	inMessages chan []byte,
) {
	defer cancel()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			logger.Wf(ctx, "[InMessages] Ignore err %v", err)
			break
		}

		select {
		case <-ctx.Done():
			return // todo: нужно?
		case inMessages <- message:
		}
	}
}

func (a *App) handleOutMessages(
	ctx context.Context,
	cancel context.CancelFunc,
	inMessages chan []byte,
	preconnectMessages chan []byte,
	outMessages chan []byte,
) {
	defer cancel()

	handleMessage := func(m []byte) error {
		action := Action{}
		if err := json.Unmarshal(m, &action); err != nil {
			return errors.Wrapf(err, "Unmarshal %s", m)
		}

		var response interface{}
		var err error

		actionType := action.Message.Action

		switch actionType {
		case "preconnect":
			response, err = handlePreconnect(ctx, a, m, action, preconnectMessages)
			if err != nil {
				return err
			}
		case "join":
			response, err = handleJoin(ctx, a, m, action, outMessages)
			if err != nil {
				return err
			}
		default:
			handler, ok := handlers[actionType]
			if !ok {
				return errors.Errorf("unknown action")
			}

			response, err = handler(ctx, a, m, action)
			if err != nil {
				return err
			}
		}

		message, err := json.Marshal(Tid{action.TID, response})
		if err != nil {
			return errors.Wrapf(err, "marshal")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case outMessages <- message:
		}

		return nil
	}

	for m := range inMessages {
		if err := handleMessage(m); err != nil {
			logger.Wf(ctx, "Handle %s err %v", m, err)
			break
		}
	}
}
