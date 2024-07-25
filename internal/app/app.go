package app

import (
	"context"
	"encoding/json"
	"signal/internal/room"
	"sync"

	"github.com/ossrs/go-oryx-lib/errors"
	"github.com/ossrs/go-oryx-lib/logger"
	"golang.org/x/net/websocket"
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

func New(logger Logger) *App {
	return &App{
		logger: logger,
	}
}

func (a *App) Health(_ context.Context) []byte {
	return []byte("OK")
}

func (a *App) Version(_ context.Context) []byte {
	return []byte("1.0.8")
}

// RTC todo: можно ли тут знать о *websocket.Conn ?
func (a *App) RTC(ctx context.Context, c *websocket.Conn) {
	ctx, cancel := context.WithCancel(logger.WithContext(ctx))
	defer cancel()

	r := c.Request()
	logger.Tf(ctx, "Serve client %v at %v", r.RemoteAddr, r.RequestURI)
	defer func(c *websocket.Conn) {
		err := c.Close()
		if err != nil {
			logger.E(ctx, err.Error())
		}
	}(c)

	var self *room.Participant
	go func() {
		<-ctx.Done()
		if self == nil {
			return
		}

		// Notify other peers that we're quiting.
		// @remark The ctx(of self) is done, so we must use a new context.
		go self.Room.Notify(context.Background(), self, "leave", "", "")

		self.Room.Remove(self)
		logger.Tf(ctx, "Remove client %v", self)
	}()

	inMessages := make(chan []byte)
	go func() {
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
	}()

	outMessages := make(chan []byte)
	go func() {
		defer cancel()

		handleMessage := func(m []byte) error {
			action := Action{}
			if err := json.Unmarshal(m, &action); err != nil {
				return errors.Wrapf(err, "Unmarshal %s", m)
			}

			var res interface{}
			if action.Message.Action == "join" {
				obj := EventJoin{}
				if err := json.Unmarshal(m, &obj); err != nil {
					return errors.Wrapf(err, "Unmarshal %s", m)
				}

				r, _ := a.rooms.LoadOrStore(obj.Message.Room, &room.Room{Name: obj.Message.Room})
				p := &room.Participant{
					Room:        r.(*room.Room),
					Display:     obj.Message.Display,
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
				if err := r.(*room.Room).Add(p); err != nil {
					return errors.Wrapf(err, "join")
				}

				self = p
				logger.Tf(ctx, "Join %v ok", self)

				res = Res{
					action.Message.Action, obj.Message.Room, p, r.(*room.Room).Participants,
				}

				go r.(*room.Room).Notify(ctx, p, action.Message.Action, "", "")
			} else if action.Message.Action == "publish" {
				obj := EventLeave{}
				if err := json.Unmarshal(m, &obj); err != nil {
					return errors.Wrapf(err, "Unmarshal %s", m)
				}

				r, _ := a.rooms.LoadOrStore(obj.Message.Room, &room.Room{Name: obj.Message.Room})
				p := r.(*room.Room).Get(obj.Message.Display)

				// Now, the peer is publishing.
				p.Publishing = true

				go r.(*room.Room).Notify(ctx, p, action.Message.Action, "", "")
			} else if action.Message.Action == "changeState" {
				obj := EventChangeState{}
				if err := json.Unmarshal(m, &obj); err != nil {
					return errors.Wrapf(err, "Unmarshal %s", m)
				}

				r, _ := a.rooms.LoadOrStore(obj.Message.Room, &room.Room{Name: obj.Message.Room})
				p := r.(*room.Room).ChangeState(obj.Message.Display, room.State{
					IsMicroOn:   obj.Message.IsMicroOn,
					IsCameraOn:  obj.Message.IsCameraOn,
					BatteryLife: obj.Message.BatteryLife,
				})

				go r.(*room.Room).Notify(ctx, p, action.Message.Action, "", "")
			} else if action.Message.Action == "control" {
				obj := EventControl{}
				if err := json.Unmarshal(m, &obj); err != nil {
					return errors.Wrapf(err, "Unmarshal %s", m)
				}

				r, _ := a.rooms.LoadOrStore(obj.Message.Room, &room.Room{Name: obj.Message.Room})
				p := r.(*room.Room).Get(obj.Message.Display)

				go r.(*room.Room).Notify(ctx, p, action.Message.Action, obj.Message.Call, obj.Message.Data)
			} else {
				obj := EventCustom{}
				if err := json.Unmarshal(m, &obj); err != nil {
					return errors.Wrapf(err, "Unmarshal %s", m)
				}

				r, _ := a.rooms.LoadOrStore(obj.Message.Room, &room.Room{Name: obj.Message.Room})
				p := r.(*room.Room).Get(obj.Message.Display)

				go r.(*room.Room).Notify(ctx, p, action.Message.Action, "", "")
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
	}()

	for m := range outMessages {
		if _, err := c.Write(m); err != nil {
			logger.Wf(ctx, "Ignore err %v for %v", err, r.RemoteAddr)
			break
		}
	}
}
