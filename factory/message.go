package factory

type Message struct {
	ID       int    `json:"id"`
	SenderID int    `json:"sender_id"`
	ChatID   int    `json:"chat_id"`
	Content  string `json:"content"` // decrypted text
	SentAt   string `json:"sent_at"` // formatted timestamp
	ChatType string `json:"chat_type"`
}
