package room

type Response struct {
	Message Message `json:"msg"`
}

type Message struct {
	Action       string         `json:"action"`
	Event        string         `json:"event"`
	Param        string         `json:"param,omitempty"`
	Data         string         `json:"data,omitempty"`
	Room         string         `json:"room"`
	Self         *Participant   `json:"self"`
	Peer         *Participant   `json:"peer"`
	Participants []*Participant `json:"participants"`
}
