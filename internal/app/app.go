package app

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/ossrs/go-oryx-lib/errors"
	"github.com/ossrs/go-oryx-lib/logger"
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
func (a *App) RTC(ctx context.Context, conn *websocket.Conn) {
	ctx, cancel := context.WithCancel(logger.WithContext(ctx))
	defer a.closeConnection(ctx, cancel, conn)

	// todo: проблема с удалением пользователя из комнаты при потери сети

	inMessages := make(chan []byte)
	go a.handleInMessages(ctx, cancel, conn, inMessages)

	outMessages := make(chan []byte)
	go a.handleOutMessages(ctx, cancel, inMessages, outMessages)

	for m := range outMessages {
		if err := conn.WriteMessage(websocket.TextMessage, m); err != nil {
			logger.Wf(ctx, "[RTC] Ignore err %v for %v", err, conn.RemoteAddr())
			break
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
			_ = conn.Close() // todo: нужно или закроется позже?
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
	outMessages chan []byte,
) {
	defer cancel()

	handleMessage := func(m []byte) error {
		action := Action{}
		if err := json.Unmarshal(m, &action); err != nil {
			return errors.Wrapf(err, "Unmarshal %s", m)
		}

		var response interface{}
		actionType := action.Message.Action
		handler, ok := handlers[actionType]
		if !ok {
			handler = handlers["default"]
		}

		response, err := handler(ctx, a, m, action, outMessages)
		if err != nil {
			return err
		}

		b, err := json.Marshal(Tid{action.TID, response})
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
		Room:         r.(*internalrooms.Room),
		Out:          outMessages,
		UserID:       obj.Message.UserID,
		FirstName:    obj.Message.FirstName,
		LastName:     obj.Message.LastName,
		Status:       obj.Message.Status,
		Photo:        obj.Message.Photo,
		IsHorizontal: obj.Message.IsHorizontal,
		IsMicroOn:    obj.Message.IsMicroOn,
		IsCameraOn:   obj.Message.IsCameraOn,
		IsSpeakerOn:  obj.Message.IsSpeakerOn,
		CameraType:   obj.Message.CameraType,
		BatteryLife:  obj.Message.BatteryLife,
	}
	if err := r.(*internalrooms.Room).Add(p); err != nil {
		return nil, errors.Wrapf(err, "join")
	}

	go p.HandleContextDone(ctx)
	logger.Tf(ctx, "Join %v ok", p)

	response := ResponseJoin{
		Action:              action.Message.Action,
		Room:                obj.Message.Room,
		Self:                p,
		Participants:        r.(*internalrooms.Room).Participants,
		InvitedParticipants: r.(*internalrooms.Room).InvitedParticipants,
		StartedAt:           r.(*internalrooms.Room).StartedAt,
	}

	go r.(*internalrooms.Room).Notify(ctx, p, action.Message.Action, "", "")

	return response, nil
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
			IsSpeakerOn: obj.Message.IsSpeakerOn,
			CameraType:  obj.Message.CameraType,
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
			UserID:    value.UserID,
			FirstName: value.FirstName,
			LastName:  value.LastName,
			Status:    value.Status,
			Photo:     value.Photo,
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
