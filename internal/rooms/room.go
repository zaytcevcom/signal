package rooms

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Room struct {
	Name                string                `json:"room"`
	Token               string                `json:"-"`
	Participants        []*Participant        `json:"participants"`
	InvitedParticipants []*InvitedParticipant `json:"invitedParticipants"`
	StartedAt           *int64                `json:"startedAt"`
	Lock                sync.RWMutex          `json:"-"`
}

type State struct {
	IsMicroOn   bool    `json:"isMicroOn"`
	IsSpeakerOn bool    `json:"isSpeakerOn"`
	CameraType  *string `json:"cameraType"`
	BatteryLife int64   `json:"batteryLife"`
}

func (r *Room) String() string {
	return fmt.Sprintf("room=%v, participants=%v", r.Name, len(r.Participants))
}

func (r *Room) Add(p *Participant) error {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	for i, participant := range r.InvitedParticipants {
		if participant.UserID == p.UserID {
			r.InvitedParticipants = append(r.InvitedParticipants[:i], r.InvitedParticipants[i+1:]...)
			break
		}
	}

	for _, participant := range r.Participants {
		if participant.UserID == p.UserID {
			return fmt.Errorf("participant %v exists in room %v", p.UserID, r.Name)
		}
	}

	r.Participants = append(r.Participants, p)

	if len(r.Participants) == 2 {
		unixTime := time.Now().Unix()
		r.StartedAt = &unixTime
	}

	return nil
}

func (r *Room) AddInvited(p *InvitedParticipant) error {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	for _, participant := range r.Participants {
		if participant.UserID == p.UserID {
			return nil
		}
	}

	for _, participant := range r.InvitedParticipants {
		if participant.UserID == p.UserID {
			return nil
		}
	}

	r.InvitedParticipants = append(r.InvitedParticipants, p)
	return nil
}

func (r *Room) Get(userID int64) (*Participant, error) {
	r.Lock.RLock()
	defer r.Lock.RUnlock()

	for _, participant := range r.Participants {
		if participant.UserID == userID {
			return participant, nil
		}
	}

	return nil, fmt.Errorf("participant %v does not exist in room %v", userID, r.Name)
}

func (r *Room) ChangePublishing(p *Participant, publishing bool) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	p.Publishing = publishing
}

func (r *Room) ChangeState(p *Participant, state State) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	p.IsMicroOn = state.IsMicroOn
	p.IsSpeakerOn = state.IsSpeakerOn
	p.CameraType = state.CameraType
	p.BatteryLife = state.BatteryLife
}

func (r *Room) Remove(p *Participant) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	for i, participant := range r.Participants {
		if p == participant {
			r.Participants = append(r.Participants[:i], r.Participants[i+1:]...)
			break
		}
	}
}

func (r *Room) Notify(ctx context.Context, peer *Participant, event string) {
	var participants []*Participant
	var invitedParticipants []*InvitedParticipant
	func() {
		r.Lock.RLock()
		defer r.Lock.RUnlock()
		participants = append(participants, r.Participants...)
		invitedParticipants = append(invitedParticipants, r.InvitedParticipants...)
	}()

	for _, participant := range participants {
		if participant == peer {
			continue
		}

		response := Response{
			Message{
				Action:              "notify",
				Event:               event,
				Self:                participant,
				Peer:                peer,
				Participants:        participants,
				InvitedParticipants: invitedParticipants,
				StartedAt:           r.StartedAt,
			},
		}

		message, err := json.Marshal(response)
		if err != nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case participant.Out <- message:
		}
	}
}
