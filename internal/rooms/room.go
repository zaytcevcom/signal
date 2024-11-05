package rooms

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ossrs/go-oryx-lib/logger"
)

type Room struct {
	Name                string                `json:"-"`
	Token               string                `json:"-"`
	Devices             []*Device             `json:"-"`
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

	for _, device := range r.Devices {
		if device.ID == d.ID {
			return fmt.Errorf("device %v exists in room %v", d.ID, r.Name)
		}
	}

	r.Devices = append(r.Devices, d)
	return nil
}

func (r *Room) RemoveDevice(d *Device) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	for i, device := range r.Devices {
		if device == d {
			logger.Tf(context.Background(), "Remove device: %v", d)
			r.Devices = append(r.Devices[:i], r.Devices[i+1:]...)
			break
		}
	}
}

func (r *Room) GetDeviceHistory(userID int64) (*Device, error) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	for _, device := range r.Devices {
		if (device.Status == DeclineStatus || device.Status == BusyStatus) && device.UserID != userID {
			return device, nil
		}
	}

	for _, device := range r.Devices {
		if device.Status != "" && device.UserID == userID {
			return device, nil
		}
	}

	return nil, nil
}

func (r *Room) Accept(deviceID string) (*Device, error) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	for _, device := range r.Devices {
		if device.ID == deviceID {
			device.Status = AcceptStatus
			return device, nil
		}
	}

	return nil, fmt.Errorf("(accept) device %v not found", deviceID)
}

func (r *Room) Decline(deviceID string) (*Device, error) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	for _, device := range r.Devices {
		if device.ID == deviceID {
			device.Status = DeclineStatus
			return device, nil
		}
	}

	return nil, fmt.Errorf("(decline) device %v not found", deviceID)
}

func (r *Room) Busy(deviceID string) (*Device, error) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	for _, device := range r.Devices {
		if device.ID == deviceID {
			device.Status = BusyStatus
			return device, nil
		}
	}

	return nil, fmt.Errorf("(busy) device %v not found", deviceID)
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

func (r *Room) Ready(p *Participant) {
	r.Lock.Lock()
	defer r.Lock.Unlock()

	p.IsReady = true
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

		devices = append(devices, r.Devices...)
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

	logger.Tf(ctx, "Count participants: %d, peerId: %d", len(participants), peer.UserID)

	for _, participant := range participants {
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

		logger.Tf(ctx, "Notify: %v", response)

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

func (r *Room) NotifySpeak(ctx context.Context, userID int64, level float64, event string) {
	var participants []*Participant
	var invitedParticipants []*InvitedParticipant
	func() {
		r.Lock.RLock()
		defer r.Lock.RUnlock()
		participants = append(participants, r.Participants...)
		invitedParticipants = append(invitedParticipants, r.InvitedParticipants...)
	}()

	response := NotifySpeakResponse{
		NotifySpeakMessage{
			Action: "notify",
			Event:  event,
			UserID: userID,
			Level:  level,
		},
	}

	message, err := json.Marshal(response)
	if err != nil {
		return
	}

	for _, participant := range participants {
		if participant.UserID == userID {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case participant.Out <- message:
		}
	}
}
