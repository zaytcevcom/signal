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
		Room         string  `json:"room"`
		Token        string  `json:"token"`
		UserID       int64   `json:"userId"`
		FirstName    string  `json:"firstName"`
		LastName     string  `json:"lastName"`
		Status       *string `json:"status"`
		Photo        *string `json:"photo"`
		IsHorizontal bool    `json:"isHorizontal"`
		IsMicroOn    bool    `json:"isMicroOn"`
		IsSpeakerOn  bool    `json:"isSpeakerOn"`
		CameraType   *string `json:"cameraType"`
		BatteryLife  int64   `json:"batteryLife"`
	} `json:"msg"`
}

type EventLeave struct {
	Message struct {
		Room   string `json:"room"`
		UserID int64  `json:"userId"`
	} `json:"msg"`
}

type EventChangeState struct {
	Message struct {
		Room        string  `json:"room"`
		UserID      int64   `json:"userId"`
		IsMicroOn   bool    `json:"isMicroOn"`
		IsSpeakerOn bool    `json:"isSpeakerOn"`
		CameraType  *string `json:"cameraType"`
		BatteryLife int64   `json:"batteryLife"`
	} `json:"msg"`
}

type EventInviteUsers struct {
	Message struct {
		Room         string                      `json:"room"`
		UserID       int64                       `json:"userId"`
		Participants []*rooms.InvitedParticipant `json:"participants"`
	} `json:"msg"`
}

type ResponseJoin struct {
	Action              string                      `json:"action"`
	Self                *rooms.Participant          `json:"self"`
	Participants        []*rooms.Participant        `json:"participants"`
	InvitedParticipants []*rooms.InvitedParticipant `json:"invitedParticipants"`
	StartedAt           *int64                      `json:"startedAt"`
}

type Tid struct {
	TID     string      `json:"tid"`
	Message interface{} `json:"msg"`
}
