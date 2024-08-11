package rooms

type Response struct {
	Message Message `json:"msg"`
}

type Message struct {
	Action              string                `json:"action"`
	Event               string                `json:"event"`
	Self                *Participant          `json:"self"`
	Peer                *Participant          `json:"peer"`
	Participants        []*Participant        `json:"participants"`
	InvitedParticipants []*InvitedParticipant `json:"invitedParticipants"`
	StartedAt           *int64                `json:"startedAt"`
}
