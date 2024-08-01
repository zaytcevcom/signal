package rooms

import (
	"context"
	"fmt"

	"github.com/ossrs/go-oryx-lib/logger"
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
	IsCameraOn   bool        `json:"isCameraOn"`
	IsSpeakerOn  bool        `json:"isSpeakerOn"`
	BatteryLife  int64       `json:"batteryLife"`
}

func (p *Participant) String() string {
	return fmt.Sprintf("userID=%v, room=%v", p.UserID, p.Room.Name)
}

func (p *Participant) HandleContextDone(ctx context.Context) {
	<-ctx.Done()
	if p == nil {
		logger.Tf(ctx, "Empty Participant")
		return
	}

	go p.Room.Notify(context.Background(), p, "leave", "", "")

	p.Room.Remove(p)
	logger.Tf(ctx, "Remove client %v", p)
}
