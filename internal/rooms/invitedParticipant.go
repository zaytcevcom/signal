package rooms

type InvitedParticipant struct {
	Room      *Room   `json:"-"`
	UserID    int64   `json:"userId"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	Status    *string `json:"status"`
	Photo     *string `json:"photo"`
}
