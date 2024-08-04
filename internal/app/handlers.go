package app

import (
	"context"
	"encoding/json"

	"github.com/ossrs/go-oryx-lib/errors"
	"github.com/ossrs/go-oryx-lib/logger"
	internalrooms "signal/internal/rooms"
)

type ActionHandler func(
	ctx context.Context,
	a *App,
	m []byte,
	action Action,
	outMessage chan []byte,
) (interface{}, error)

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