package factory

type User struct {
	ID             int    `json:"id"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	Password       string `json:"password,omitempty"`
	HashedPassword string `json:"hashed_password,omitempty"`
	Created        string `json:"created"`
}
