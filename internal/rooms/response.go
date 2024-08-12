package rooms

type NotifyResponse struct {
	Message NotifyMessage `json:"msg"`
}

type NotifyMessage struct {
	Action              string                `json:"action"`
	Event               string                `json:"event"`
	Self                *Participant          `json:"self"`
	Peer                *Participant          `json:"peer"`
	Participants        []*Participant        `json:"participants"`
	InvitedParticipants []*InvitedParticipant `json:"invitedParticipants"`
	StartedAt           *int64                `json:"startedAt"`
}

type NotifyPreconnectResponse struct {
	Message NotifyPreconnectMessage `json:"msg"`
}

type NotifyPreconnectMessage struct {
	Action   string `json:"action"`
	Event    string `json:"event"`
	UserID   int64  `json:"userId"`
	DeviceID string `json:"deviceId"`
}
