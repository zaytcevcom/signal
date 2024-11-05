package app

import "signal/internal/rooms"

type Action struct {
	TID     string `json:"tid"`
	Message struct {
		Action string `json:"action"`
	} `json:"msg"`
}

type EventPreconnect struct {
	Message struct {
		Room     string `json:"room"`
		Token    string `json:"token"`
		UserID   int64  `json:"userId"`
		DeviceID string `json:"deviceId"`
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
		Sex          *int64  `json:"sex"`
		Photo        *string `json:"photo"`
		IsHorizontal bool    `json:"isHorizontal"`
		IsMicroOn    bool    `json:"isMicroOn"`
		IsSpeakerOn  bool    `json:"isSpeakerOn"`
		CameraType   *string `json:"cameraType"`
		BatteryLife  float64 `json:"batteryLife"`
		IsReady      bool    `json:"isReady"`
	} `json:"msg"`
}

type EventPublish struct {
	Message struct {
		Room   string `json:"room"`
		UserID int64  `json:"userId"`
	} `json:"msg"`
}

type EventStreamPublish struct {
	Message struct {
		Room   string `json:"room"`
		UserID int64  `json:"userId"`
		SDP    string `json:"sdp"`
	} `json:"msg"`
}

type EventStreamPlay struct {
	Message struct {
		Room          string `json:"room"`
		UserID        int64  `json:"userId"`
		SDP           string `json:"sdp"`
		ParticipantID int64  `json:"participantId"`
	} `json:"msg"`
}

type EventReady struct {
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
		BatteryLife float64 `json:"batteryLife"`
	} `json:"msg"`
}

type EventSpeak struct {
	Message struct {
		Room   string  `json:"room"`
		UserID int64   `json:"userId"`
		Level  float64 `json:"level"`
	} `json:"msg"`
}

type EventInviteUsers struct {
	Message struct {
		Room         string                      `json:"room"`
		UserID       int64                       `json:"userId"`
		Participants []*rooms.InvitedParticipant `json:"participants"`
	} `json:"msg"`
}

type ResponsePreconnect struct {
	Action string        `json:"action"`
	Device *rooms.Device `json:"device"`
}

type ResponseJoin struct {
	Action              string                      `json:"action"`
	Self                *rooms.Participant          `json:"self"`
	Participants        []*rooms.Participant        `json:"participants"`
	InvitedParticipants []*rooms.InvitedParticipant `json:"invitedParticipants"`
	StartedAt           *int64                      `json:"startedAt"`
}

type ResponseStream struct {
	Code      int64  `json:"code"`
	Pid       string `json:"pid"`
	SDP       string `json:"sdp"`
	Server    string `json:"server"`
	Service   string `json:"service"`
	SessionID string `json:"sessionid"`
}

type Tid struct {
	TID     string      `json:"tid"`
	Message interface{} `json:"msg"`
}

type Stream struct {
	StreamURL string `json:"streamurl"`
	Sdp       string `json:"sdp"`
}
