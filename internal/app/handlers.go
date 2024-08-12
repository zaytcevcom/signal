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
) (interface{}, error)

func handlePreconnect(
	ctx context.Context,
	a *App,
	m []byte,
	action Action,
	outMessages chan []byte,
) (interface{}, error) {
	obj := EventPreconnect{}
	if err := json.Unmarshal(m, &obj); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s", m)
	}

	r, loaded := a.rooms.Load(obj.Message.Room)
	if !loaded {
		return nil, errors.Errorf("room %s does not exist", obj.Message.Room)
	}

	d := &internalrooms.Device{
		Room:   r.(*internalrooms.Room),
		Out:    outMessages,
		UserID: obj.Message.UserID,
		ID:     obj.Message.DeviceID,
		Status: "",
	}

	err := r.(*internalrooms.Room).AddDevice(d)
	if err != nil {
		return nil, err
	}

	device, err := r.(*internalrooms.Room).GetDeviceHistory(obj.Message.UserID)
	if err != nil {
		return nil, err
	}

	go d.HandleContextDone(ctx)

	response := ResponsePreconnect{
		Action: action.Message.Action,
		Device: device,
	}

	return response, nil
}

func handleAccept(
	ctx context.Context,
	a *App,
	m []byte,
	action Action,
) (interface{}, error) {
	obj := EventPreconnect{}
	if err := json.Unmarshal(m, &obj); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s", m)
	}

	r, loaded := a.rooms.Load(obj.Message.Room)
	if !loaded {
		return nil, errors.Errorf("room %s does not exist", obj.Message.Room)
	}

	d, err := r.(*internalrooms.Room).Accept(obj.Message.UserID, obj.Message.DeviceID)
	if err != nil {
		return nil, err
	}

	go r.(*internalrooms.Room).NotifyPreconnect(ctx, d, action.Message.Action)

	return nil, nil
}

func handleDecline(
	ctx context.Context,
	a *App,
	m []byte,
	action Action,
) (interface{}, error) {
	obj := EventPreconnect{}
	if err := json.Unmarshal(m, &obj); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s", m)
	}

	r, loaded := a.rooms.Load(obj.Message.Room)
	if !loaded {
		return nil, errors.Errorf("room %s does not exist", obj.Message.Room)
	}

	d, err := r.(*internalrooms.Room).Decline(obj.Message.UserID, obj.Message.DeviceID)
	if err != nil {
		return nil, err
	}

	go r.(*internalrooms.Room).NotifyPreconnect(ctx, d, action.Message.Action)

	return nil, nil
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
		Publishing:   false,
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
) (interface{}, error) {
	obj := EventPublish{}
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
		invitedPeer := &internalrooms.InvitedParticipant{
			Room:      r.(*internalrooms.Room),
			UserID:    value.UserID,
			FirstName: value.FirstName,
			LastName:  value.LastName,
			Status:    value.Status,
			Photo:     value.Photo,
		}
		if err := r.(*internalrooms.Room).AddInvited(invitedPeer); err != nil {
			return nil, errors.Wrapf(err, "inviteUsers")
		}
	}

	go r.(*internalrooms.Room).Notify(ctx, p, action.Message.Action)

	return nil, nil
}
