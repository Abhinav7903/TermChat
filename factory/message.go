package factory

type Message struct {
	ID           int               `json:"id"`
	SenderID     int               `json:"sender_id"`
	SenderName   string            `json:"sender_name"`
	RecevierName string            `json:"receiver_name"`
	ChatID       int               `json:"chat_id"`
	Content      string            `json:"content"` // decrypted text
	SentAt       string            `json:"sent_at"` // formatted timestamp
	ChatType     string            `json:"chat_type"`
	Reactions    map[string]string `json:"reactions"` // username -> emoji
}

type GroupChat struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     int    `json:"owner_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	IsGlobal    bool   `json:"is_global"`
}

type GroupMember struct {
	GroupID  int    `json:"group_id"`
	UserID   int    `json:"user_id"`
	JoinedAt string `json:"joined_at"`
	Role     string `json:"role"`
}
