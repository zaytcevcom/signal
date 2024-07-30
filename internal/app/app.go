package app

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/ossrs/go-oryx-lib/errors"
	"github.com/ossrs/go-oryx-lib/logger"
	"golang.org/x/net/websocket"
	internalrooms "signal/internal/rooms"
)

type App struct {
	logger Logger
	rooms  sync.Map
}

type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

type ActionHandler func(
	ctx context.Context,
	a *App,
	m []byte,
	action Action,
	outMessage chan []byte,
) (interface{}, error)

var handlers map[string]ActionHandler

func init() {
	handlers = map[string]ActionHandler{
		"join":        handleJoin,
		"publish":     handlePublish,
		"changeState": handleChangeState,
		"control":     handleControl,
		"inviteUsers": handleInviteUsers,
		"default":     handleDefault,
	}
}

func New(logger Logger) *App {
	return &App{
		logger: logger,
	}
}

func (a *App) Health(_ context.Context) []byte {
	return []byte("OK")
}

func (a *App) Version(_ context.Context) []byte {
	return []byte("1.0.1")
}

// RTC todo: можно ли тут знать о *websocket.Conn ?
func (a *App) RTC(ctx context.Context, c *websocket.Conn) {
	ctx, cancel := context.WithCancel(logger.WithContext(ctx))
	defer a.closeConnection(ctx, cancel, c)

	r := c.Request()

	logger.Tf(ctx, "Serve client %v at %v", r.RemoteAddr, r.RequestURI)

	inMessages := make(chan []byte)
	go a.handleInMessages(ctx, cancel, c, r, inMessages)

	outMessages := make(chan []byte)
	go a.handleOutMessages(ctx, cancel, inMessages, outMessages)

	for m := range outMessages {
		if _, err := c.Write(m); err != nil {
			logger.Wf(ctx, "Ignore err %v for %v", err, r.RemoteAddr)
			break
		}
	}
}

func (a *App) closeConnection(ctx context.Context, cancel context.CancelFunc, c *websocket.Conn) {
	err := c.Close()
	if err != nil {
		logger.E(ctx, err.Error())
	}

	cancel()
}

func (a *App) handleInMessages(
	ctx context.Context,
	cancel context.CancelFunc,
	c *websocket.Conn,
	r *http.Request,
	inMessages chan []byte,
) {
	defer cancel()

	buf := make([]byte, 16384)
	for {
		n, err := c.Read(buf)
		if err != nil {
			logger.Wf(ctx, "Ignore err %v for %v", err, r.RemoteAddr)
			break
		}

		select {
		case <-ctx.Done():
		case inMessages <- buf[:n]:
		}
	}
}

func (a *App) handleOutMessages(
	ctx context.Context,
	cancel context.CancelFunc,
	inMessages chan []byte,
	outMessages chan []byte,
) {
	defer cancel()

	handleMessage := func(m []byte) error {
		action := Action{}
		if err := json.Unmarshal(m, &action); err != nil {
			return errors.Wrapf(err, "Unmarshal %s", m)
		}

		var res interface{}
		actionType := action.Message.Action
		handler, ok := handlers[actionType]
		if !ok {
			handler = handlers["default"]
		}

		res, err := handler(ctx, a, m, action, outMessages)
		if err != nil {
			return err
		}

		b, err := json.Marshal(Tid{action.TID, res})
		if err != nil {
			return errors.Wrapf(err, "marshal")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case outMessages <- b:
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

func handleJoin(
	ctx context.Context,
	a *App,
	m []byte,
	action Action,
	outMessages chan []byte,
) (interface{}, error) {
	obj := EventJoin{}
	if err := json.Unmarshal(m, &obj); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s", m)
	}

	r, _ := a.rooms.LoadOrStore(obj.Message.Room, &internalrooms.Room{Name: obj.Message.Room})
	p := &internalrooms.Participant{
		Room:        r.(*internalrooms.Room),
		Out:         outMessages,
		UserID:      obj.Message.UserID,
		FirstName:   obj.Message.FirstName,
		LastName:    obj.Message.LastName,
		Status:      obj.Message.Status,
		Photo:       nil,
		IsMicroOn:   obj.Message.IsMicroOn,
		IsCameraOn:  obj.Message.IsCameraOn,
		BatteryLife: obj.Message.BatteryLife,
	}
	if err := r.(*internalrooms.Room).Add(p); err != nil {
		return nil, errors.Wrapf(err, "join")
	}

	go p.HandleContextDone(ctx)
	logger.Tf(ctx, "Join %v ok", p)

	res := Res{
		Action:       action.Message.Action,
		Room:         obj.Message.Room,
		Self:         p,
		Participants: r.(*internalrooms.Room).Participants,
	}

	go r.(*internalrooms.Room).Notify(ctx, p, action.Message.Action, "", "")

	return res, nil
}

func handlePublish(
	ctx context.Context,
	a *App,
	m []byte,
	action Action,
	_ chan []byte,
) (interface{}, error) {
	obj := EventLeave{}
	if err := json.Unmarshal(m, &obj); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s", m)
	}

	r, _ := a.rooms.LoadOrStore(obj.Message.Room, &internalrooms.Room{Name: obj.Message.Room})

	p, err := r.(*internalrooms.Room).Get(obj.Message.UserID)
	if err != nil {
		return nil, errors.Wrapf(err, "publish")
	}

	p.Publishing = true

	go r.(*internalrooms.Room).Notify(ctx, p, action.Message.Action, "", "")

	return nil, nil
}

func handleChangeState(
	ctx context.Context,
	a *App,
	m []byte,
	action Action,
	_ chan []byte,
) (interface{}, error) {
	obj := EventChangeState{}
	if err := json.Unmarshal(m, &obj); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s", m)
	}

	r, _ := a.rooms.LoadOrStore(obj.Message.Room, &internalrooms.Room{Name: obj.Message.Room})

	p := r.(*internalrooms.Room).ChangeState(
		obj.Message.UserID,
		internalrooms.State{
			IsMicroOn:   obj.Message.IsMicroOn,
			IsCameraOn:  obj.Message.IsCameraOn,
			BatteryLife: obj.Message.BatteryLife,
		},
	)

	go r.(*internalrooms.Room).Notify(ctx, p, action.Message.Action, "", "")

	return nil, nil
}

func handleControl(
	ctx context.Context,
	a *App,
	m []byte,
	action Action,
	_ chan []byte,
) (interface{}, error) {
	obj := EventControl{}
	if err := json.Unmarshal(m, &obj); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s", m)
	}

	r, _ := a.rooms.LoadOrStore(obj.Message.Room, &internalrooms.Room{Name: obj.Message.Room})

	p, err := r.(*internalrooms.Room).Get(obj.Message.UserID)
	if err != nil {
		return nil, errors.Wrapf(err, "control")
	}

	go r.(*internalrooms.Room).Notify(ctx, p, action.Message.Action, obj.Message.Call, obj.Message.Data)

	return nil, nil
}

func handleInviteUsers(
	ctx context.Context,
	a *App,
	m []byte,
	action Action,
	_ chan []byte,
) (interface{}, error) {
	obj := EventInviteUsers{}
	if err := json.Unmarshal(m, &obj); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s", m)
	}

	r, _ := a.rooms.LoadOrStore(obj.Message.Room, &internalrooms.Room{Name: obj.Message.Room})

	for _, value := range obj.Message.Participants {
		p := &internalrooms.InvitedParticipant{
			Room:      r.(*internalrooms.Room),
			UserID:    obj.Message.UserID,
			FirstName: value.FirstName,
			LastName:  value.LastName,
			Status:    value.Status,
			Photo:     nil,
		}
		if err := r.(*internalrooms.Room).AddInvited(p); err != nil {
			return nil, errors.Wrapf(err, "inviteUsers")
		}
	}

	p, err := r.(*internalrooms.Room).Get(obj.Message.UserID)
	if err != nil {
		return nil, errors.Wrapf(err, "inviteUsers")
	}

	go r.(*internalrooms.Room).Notify(ctx, p, action.Message.Action, "", "")

	return nil, nil
}

func handleDefault(
	ctx context.Context,
	a *App,
	m []byte,
	action Action,
	_ chan []byte,
) (interface{}, error) {
	obj := EventCustom{}
	if err := json.Unmarshal(m, &obj); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s", m)
	}

	r, _ := a.rooms.LoadOrStore(obj.Message.Room, &internalrooms.Room{Name: obj.Message.Room})

	p, err := r.(*internalrooms.Room).Get(obj.Message.UserID)
	if err != nil {
		return nil, errors.Wrapf(err, "default")
	}

	go r.(*internalrooms.Room).Notify(ctx, p, action.Message.Action, "", "")

	return nil, nil
}
