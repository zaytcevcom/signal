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

	r, loaded := a.rooms.Load(obj.Message.Room)

	if !loaded {
		r = &internalrooms.Room{
			Name:  obj.Message.Room,
			Token: obj.Message.Token,
		}

		a.rooms.Store(obj.Message.Room, r)
	} else if r.(*internalrooms.Room).Token != obj.Message.Token {
		return nil, errors.Errorf("Invalid token for room %s", obj.Message.Room)
	}

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
		IsSpeakerOn:  obj.Message.IsSpeakerOn,
		CameraType:   obj.Message.CameraType,
		BatteryLife:  obj.Message.BatteryLife,
	}
	if err := r.(*internalrooms.Room).Add(p); err != nil {
		return nil, errors.Wrapf(err, "join")
	}

	go p.HandleContextDone(ctx, a.emptyRooms)
	logger.Tf(ctx, "Join %v ok", p)

	response := ResponseJoin{
		Action:              action.Message.Action,
		Self:                p,
		Participants:        r.(*internalrooms.Room).Participants,
		InvitedParticipants: r.(*internalrooms.Room).InvitedParticipants,
		StartedAt:           r.(*internalrooms.Room).StartedAt,
	}

	go r.(*internalrooms.Room).Notify(ctx, p, action.Message.Action)

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

	r, loaded := a.rooms.Load(obj.Message.Room)
	if !loaded {
		return nil, errors.Errorf("room %s does not exist", obj.Message.Room)
	}

	p, err := r.(*internalrooms.Room).Get(obj.Message.UserID)
	if err != nil {
		return nil, errors.Wrapf(err, "publish")
	}

	r.(*internalrooms.Room).ChangePublishing(p, true)

	go r.(*internalrooms.Room).Notify(ctx, p, action.Message.Action)

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

	r, loaded := a.rooms.Load(obj.Message.Room)
	if !loaded {
		return nil, errors.Errorf("room %s does not exist", obj.Message.Room)
	}

	p, err := r.(*internalrooms.Room).Get(obj.Message.UserID)
	if err != nil {
		return nil, errors.Wrapf(err, "changeState")
	}

	r.(*internalrooms.Room).ChangeState(
		p,
		internalrooms.State{
			IsMicroOn:   obj.Message.IsMicroOn,
			IsSpeakerOn: obj.Message.IsSpeakerOn,
			CameraType:  obj.Message.CameraType,
			BatteryLife: obj.Message.BatteryLife,
		},
	)

	go r.(*internalrooms.Room).Notify(ctx, p, action.Message.Action)

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

	r, loaded := a.rooms.Load(obj.Message.Room)
	if !loaded {
		return nil, errors.Errorf("room %s does not exist", obj.Message.Room)
	}

	p, err := r.(*internalrooms.Room).Get(obj.Message.UserID)
	if err != nil {
		return nil, errors.Wrapf(err, "inviteUsers")
	}

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

	go r.(*internalrooms.Room).Notify(ctx, p, action.Message.Action)

	return nil, nil
}

func handleDefault(
	_ context.Context,
	_ *App,
	_ []byte,
	_ Action,
	_ chan []byte,
) (interface{}, error) {
	return nil, errors.Errorf("unknown action")
}
