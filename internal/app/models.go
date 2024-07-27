package app

import "signal/internal/rooms"

type Action struct {
	TID     string `json:"tid"`
	Message struct {
		Action string `json:"action"`
	} `json:"msg"`
}

type EventJoin struct {
	Message struct {
		Room        string  `json:"room"`
		UserID      int64   `json:"userId"`
		FirstName   string  `json:"firstName"`
		LastName    string  `json:"lastName"`
		Status      *string `json:"status"`
		Photo       *string `json:"photo"`
		IsMicroOn   bool    `json:"isMicroOn"`
		IsCameraOn  bool    `json:"isCameraOn"`
		BatteryLife int64   `json:"batteryLife"`
	} `json:"msg"`
}

type EventLeave struct {
	Message struct {
		Room    string `json:"room"`
		Display string `json:"display"` // todo: Delete
		UserID  int64  `json:"userId"`
	} `json:"msg"`
}

type EventChangeState struct {
	Message struct {
		Room        string `json:"room"`
		UserID      int64  `json:"userId"`
		IsMicroOn   bool   `json:"isMicroOn"`
		IsCameraOn  bool   `json:"isCameraOn"`
		BatteryLife int64  `json:"batteryLife"`
	} `json:"msg"`
}

type EventControl struct {
	Message struct {
		Room   string `json:"room"`
		UserID int64  `json:"userId"`
		Call   string `json:"call"`
		Data   string `json:"data"`
	} `json:"msg"`
}

type EventCustom struct {
	Message struct {
		Room   string `json:"room"`
		UserID int64  `json:"userId"`
	} `json:"msg"`
}

type Res struct {
	Action       string               `json:"action"`
	Room         string               `json:"room"`
	Self         *rooms.Participant   `json:"self"`
	Participants []*rooms.Participant `json:"participants"`
}

type Tid struct {
	TID     string      `json:"tid"`
	Message interface{} `json:"msg"`
}
