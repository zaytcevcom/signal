package rooms

import (
	"context"
	"fmt"
)

type InvitedParticipant struct {
	Room      *Room   `json:"-"`
	UserID    int64   `json:"userId"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	Status    *string `json:"status"`
	Photo     *string `json:"photo"`
}

type Participant struct {
	Room         *Room       `json:"-"`
	Publishing   bool        `json:"publishing"`
	Out          chan []byte `json:"-"`
	UserID       int64       `json:"userId"`
	FirstName    string      `json:"firstName"`
	LastName     string      `json:"lastName"`
	Status       *string     `json:"status"`
	Photo        *string     `json:"photo"`
	IsHorizontal bool        `json:"isHorizontal"`
	IsMicroOn    bool        `json:"isMicroOn"`
	IsSpeakerOn  bool        `json:"isSpeakerOn"`
	CameraType   *string     `json:"cameraType"`
	BatteryLife  int64       `json:"batteryLife"`
}

func (p *Participant) String() string {
	return fmt.Sprintf("userID=%v, room=%v", p.UserID, p.Room.Name)
}

// HandleContextDone Todo: возможно есть лучше варианты, как удалить комнату если из нее вышли все участники?
func (p *Participant) HandleContextDone(ctx context.Context, emptyRooms chan<- string) {
	<-ctx.Done()
	if p == nil {
		return
	}

	p.Room.Remove(p)
	p.Room.Notify(context.Background(), p, "leave")

	if len(p.Room.Participants) == 0 {
		emptyRooms <- p.Room.Name
	}
}
