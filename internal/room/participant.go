package room

import "fmt"

type Participant struct {
	Room        *Room       `json:"-"`
	Display     string      `json:"display"`
	Publishing  bool        `json:"publishing"`
	Out         chan []byte `json:"-"`
	UserID      int64       `json:"userId"`
	FirstName   string      `json:"firstName"`
	LastName    string      `json:"lastName"`
	Status      *string     `json:"status"`
	Photo       *string     `json:"photo"`
	IsMicroOn   bool        `json:"isMicroOn"`
	IsCameraOn  bool        `json:"isCameraOn"`
	BatteryLife int64       `json:"batteryLife"`
}

func (v *Participant) String() string {
	return fmt.Sprintf("display=%v, room=%v", v.Display, v.Room.Name)
}
