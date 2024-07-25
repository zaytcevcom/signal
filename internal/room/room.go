package room

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ossrs/go-oryx-lib/errors"
	"github.com/ossrs/go-oryx-lib/logger"
)

type Room struct {
	Name         string         `json:"room"`
	Participants []*Participant `json:"participants"`
	Lock         sync.RWMutex   `json:"-"`
}

type State struct {
	IsMicroOn   bool  `json:"isMicroOn"`
	IsCameraOn  bool  `json:"isCameraOn"`
	BatteryLife int64 `json:"batteryLife"`
}

func (v *Room) String() string {
	return fmt.Sprintf("room=%v, participants=%v", v.Name, len(v.Participants))
}

func (v *Room) Add(p *Participant) error {
	v.Lock.Lock()
	defer v.Lock.Unlock()

	for _, r := range v.Participants {
		if r.Display == p.Display {
			return errors.Errorf("Participant %v exists in room %v", p.Display, v.Name)
		}
	}

	v.Participants = append(v.Participants, p)
	return nil
}

func (v *Room) Get(display string) *Participant {
	v.Lock.RLock()
	defer v.Lock.RUnlock()

	for _, r := range v.Participants {
		if r.Display == display {
			return r
		}
	}

	return nil
}

func (v *Room) ChangeState(display string, state State) *Participant {
	v.Lock.Lock()
	defer v.Lock.Unlock()

	for i, r := range v.Participants {
		if r.Display == display {
			v.Participants[i].IsMicroOn = state.IsMicroOn
			v.Participants[i].IsCameraOn = state.IsCameraOn
			v.Participants[i].BatteryLife = state.BatteryLife

			return r
		}
	}

	return nil
}

func (v *Room) Remove(p *Participant) {
	v.Lock.Lock()
	defer v.Lock.Unlock()

	for i, r := range v.Participants {
		if p == r {
			v.Participants = append(v.Participants[:i], v.Participants[i+1:]...)
			return
		}
	}
}

func (v *Room) Notify(ctx context.Context, peer *Participant, event, param, data string) {
	var participants []*Participant
	func() {
		v.Lock.RLock()
		defer v.Lock.RUnlock()
		participants = append(participants, v.Participants...)
	}()

	for _, r := range participants {
		if r == peer {
			continue
		}

		res := Response{
			Message{
				Action:       "notify",
				Event:        event,
				Param:        param,
				Data:         data,
				Room:         v.Name,
				Self:         r,
				Peer:         peer,
				Participants: participants,
			},
		}

		b, err := json.Marshal(res)
		if err != nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case r.Out <- b:
		}

		logger.Tf(ctx, "Notify %v about %v %v", r, peer, event)
	}
}
