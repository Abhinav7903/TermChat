package users

import "termchat/factory"

type Repository interface {
	CreateUser(user factory.User) error
	Login(data factory.User) (factory.User, error)
	GetUser(email string) (factory.User, error)
	SearchUsersByName(name string) ([]factory.User, error)
}
