package rooms

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Room struct {
	Name                string                `json:"-"`
	Token               string                `json:"-"`
	UserDevices         sync.Map              `json:"-"` // todo: тут не нужно типизировать?
	Participants        []*Participant        `json:"participants"`
	InvitedParticipants []*InvitedParticipant `json:"invitedParticipants"`
	StartedAt           *int64                `json:"startedAt"`
	Lock                sync.RWMutex          `json:"-"`
}

type State struct {
	IsMicroOn   bool    `json:"isMicroOn"`
	IsSpeakerOn bool    `json:"isSpeakerOn"`
	CameraType  *string `json:"cameraType"`
	BatteryLife float64 `json:"batteryLife"`
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

func (r *Room) AddDevice(d *Device) error {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	value, _ := r.UserDevices.LoadOrStore(d.UserID, make([]*Device, 0))

	devices, ok := value.([]*Device)
	if !ok {
		return fmt.Errorf("couldn't load devices for user: %d", d.UserID)
	}

	for _, device := range devices {
		if device.ID == d.ID {
			return nil
		}
	}

	devices = append(devices, d)
	r.UserDevices.Store(d.UserID, devices)
	return nil
}

func (r *Room) RemoveDevice(d *Device) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	value, _ := r.UserDevices.LoadOrStore(d.UserID, make([]*Device, 0))

	devices, ok := value.([]*Device)
	if !ok {
		return
	}

	for i, device := range devices {
		if d == device {
			devices = append(devices[:i], devices[i+1:]...)
			r.UserDevices.Store(d.UserID, devices)
			return
		}
	}
}

func (r *Room) GetDeviceHistory(userID int64) (*Device, error) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	value, _ := r.UserDevices.LoadOrStore(userID, make([]*Device, 0))

	devices, ok := value.([]*Device)
	if !ok {
		return nil, fmt.Errorf("couldn't parse devices for user: %d", userID)
	}

	for _, device := range devices {
		if device.Status != "" {
			return device, nil
		}
	}

	return nil, nil
}

func (r *Room) Accept(userID int64, deviceID string) (*Device, error) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	value, _ := r.UserDevices.LoadOrStore(userID, make([]*Device, 0))

	devices, ok := value.([]*Device)
	if !ok {
		return nil, fmt.Errorf("couldn't load devices for user: %d", userID)
	}

	for _, device := range devices {
		if device.ID == deviceID {
			device.Status = "accept"
			return device, nil
		}
	}

	return nil, fmt.Errorf("device not found for user: %d", userID)
}

func (r *Room) Decline(userID int64, deviceID string) (*Device, error) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	value, _ := r.UserDevices.LoadOrStore(userID, make([]*Device, 0))

	devices, ok := value.([]*Device)
	if !ok {
		return nil, fmt.Errorf("couldn't load devices for user: %d", userID)
	}

	for _, device := range devices {
		if device.ID == deviceID {
			device.Status = "decline"
			return device, nil
		}
	}

	return nil, fmt.Errorf("device not found for user: %d", userID)
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

func (r *Room) NotifyPreconnect(ctx context.Context, d *Device, event string) {
	var devices []*Device
	func() {
		r.Lock.RLock()
		defer r.Lock.RUnlock()

		value, _ := r.UserDevices.LoadOrStore(d.UserID, make([]*Device, 0))

		items, ok := value.([]*Device)
		if !ok {
			return
		}

		devices = append(devices, items...)
	}()

	for _, device := range devices {
		if device == d {
			continue
		}

		response := NotifyPreconnectResponse{
			NotifyPreconnectMessage{
				Action:   "notify",
				Event:    event,
				UserID:   d.UserID,
				DeviceID: d.ID,
			},
		}

		message, err := json.Marshal(response)
		if err != nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case device.Out <- message:
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

		response := NotifyResponse{
			NotifyMessage{
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
