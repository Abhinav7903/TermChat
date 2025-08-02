package factory

type Message struct {
	ID           int    `json:"id"`
	SenderID     int    `json:"sender_id"`
	SenderName   string `json:"sender_name"`
	RecevierName string `json:"receiver_name"`
	ChatID       int    `json:"chat_id"`
	Content      string `json:"content"` // decrypted text
	SentAt       string `json:"sent_at"` // formatted timestamp
	ChatType     string `json:"chat_type"`
}
