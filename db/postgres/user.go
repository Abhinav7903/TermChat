package postgres

import (
	"database/sql"
	"fmt"
	"strings"
	"termchat/factory"
	"termchat/pkg/users"
	"time"

	"github.com/lib/pq"
)

// CreateUser inserts a new user into the database

// CreateUser inserts a new user into the database
func (p *Postgres) CreateUser(user factory.User) error {
	query := `
		INSERT INTO users (email, username, password_hash)
		VALUES ($1, $2, $3)
	`
	_, err := p.DbConn.Exec(query, user.Email, user.Name, user.HashedPassword)
	if err != nil {
		// Detect unique constraint violations
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				if strings.Contains(pqErr.Message, "username") {
					return fmt.Errorf("username already taken")
				}
				if strings.Contains(pqErr.Message, "email") {
					return fmt.Errorf("email already registered")
				}
			}
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// Login verifies user credentials
func (p *Postgres) Login(data factory.User) (factory.User, error) {
	var user factory.User
	var hashedPassword string
	var createdAt time.Time

	query := `
		SELECT id, email, username, password_hash, created_at
		FROM users WHERE email = $1
	`
	err := p.DbConn.QueryRow(query, data.Email).Scan(
		&user.ID, &user.Email, &user.Name, &hashedPassword, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return factory.User{}, fmt.Errorf("user not found")
		}
		return factory.User{}, fmt.Errorf("login query failed: %w", err)
	}

	if !users.VerifyPassword(hashedPassword, data.Password) {
		return factory.User{}, fmt.Errorf("invalid password")
	}

	user.Created = createdAt.Format(time.RFC3339)
	return user, nil
}

// GetUser retrieves a user by email
func (p *Postgres) GetUser(email string) (factory.User, error) {
	var user factory.User
	var createdAt time.Time

	query := `
		SELECT id, email, username, created_at
		FROM users WHERE email = $1
	`
	err := p.DbConn.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.Name, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return factory.User{}, fmt.Errorf("user not found")
		}
		return factory.User{}, fmt.Errorf("get user query failed: %w", err)
	}

	user.Created = createdAt.Format(time.RFC3339)
	return user, nil
}

// GetAllUsers retrieves all users
func (p *Postgres) GetAllUsers() ([]factory.User, error) {
	query := `
		SELECT id, email, username, created_at
		FROM users ORDER BY created_at DESC
	`
	rows, err := p.DbConn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var usersList []factory.User
	for rows.Next() {
		var user factory.User
		var createdAt time.Time

		err := rows.Scan(&user.ID, &user.Email, &user.Name, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}

		user.Created = createdAt.Format(time.RFC3339)
		usersList = append(usersList, user)
	}
	return usersList, nil
}

// DeleteUser deletes a user by email
func (p *Postgres) DeleteUser(email string) error {
	query := `DELETE FROM users WHERE email = $1`
	res, err := p.DbConn.Exec(query, email)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("no user found with email %s", email)
	}
	return nil
}
