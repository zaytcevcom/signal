package rooms

import "context"

type Device struct {
	Room   *Room       `json:"-"`
	Out    chan []byte `json:"-"`
	UserID int64       `json:"-"`
	ID     string      `json:"id"`
	Status string      `json:"status"`
}

func (d *Device) HandleContextDone(ctx context.Context) {
	<-ctx.Done()
	if d == nil {
		return
	}

	d.Room.RemoveDevice(d)
}
