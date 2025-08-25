package redis

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// Redis struct holds the Redis client
type Redis struct {
	Client *redis.Client // Exported field
}

var once sync.Once
var instance *Redis

// initialize initializes the Redis client with the provided environment type
func initialize(envType *string) *redis.Client {
	slog.Debug("Initializing Redis")
	var rdb *redis.Client
	if *envType == "dev" {
		rdb = redis.NewClient(&redis.Options{
			Addr: "localhost:6379", // Change to your Redis instance
			DB:   0,                // Use default DB
		})
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr: "localhost:6379", // Change to your Redis instance
			DB:   0,                // Use default DB
		})
	}
	return rdb
}

// Ping returns a simple PONG string from Redis to verify connection
func (r *Redis) Ping() string {
	return r.Client.Ping(context.Background()).Val()
}

// NewRedis initializes and returns a singleton Redis client
func NewRedis(envType *string) *Redis {
	once.Do(func() {
		client := initialize(envType)
		instance = &Redis{
			Client: client, // Exported field
		}
		slog.Debug("Connected with Redis!!!!!")
		// Call ping to verify connection
		pingResult := instance.Ping()
		if pingResult != "PONG" {
			slog.Error("Failed to connect to Redis", "ping", pingResult)
			panic(fmt.Sprintf("Failed to connect to Redis: %s", pingResult))
		}
		slog.Info(pingResult)
	})
	return instance
}

// StoreEmailHash stores a hashed version of the email in Redis
func (r *Redis) StoreEmailHash(email string) (string, error) {
	// Create a SHA256 hash of the email
	h := sha256.New()
	h.Write([]byte(email))
	bs := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Store the hash and email in Redis
	return string(bs), r.Client.Set(context.Background(), string(bs), email, 0).Err()
}

// GetEmailFromHash retrieves the email from Redis using the hash, and then deletes the key
func (r *Redis) GetEmailFromHash(hash string) (string, error) {
	ctx := context.Background()
	email, err := r.Client.Get(ctx, hash).Result()
	if err != nil || len(email) == 0 {
		return "", err
	}
	// Now delete the key from Redis
	err = r.Client.Del(ctx, hash).Err()
	return email, err
}

// GenerateToken generates a new token for a session (like a JWT or random string)
func (r *Redis) GenerateToken(email string) (string, error) {
	// Create a SHA256 hash of the email and current timestamp for uniqueness
	h := sha256.New()
	h.Write([]byte(email + time.Now().String())) // Include timestamp for uniqueness
	token := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Store the token in Redis with an expiration of 1 hour
	if err := r.Client.Set(context.Background(), token, email, time.Hour).Err(); err != nil {
		return "", fmt.Errorf("failed to store token in Redis: %w", err)
	}

	return token, nil
}

// CheckSession checks if a session (token) exists for the given email
func (r *Redis) CheckSession(email string) (bool, error) {
	// Redis key to check for session existence
	key := "session:" + email
	slog.Debug("Checking session for email", "email", email, "key", key)

	exists, err := r.Client.Exists(context.Background(), key).Result()
	if err != nil {
		slog.Error("Error checking session existence", "error", err)
		return false, err
	}

	slog.Debug("Session exists", "exists", exists)
	return exists == 1, nil
}

// StoreSession stores a session (token) in Redis
func (r *Redis) StoreSession(email, token string) error {
	key := "session:" + email
	return r.Client.Set(context.Background(), key, token, time.Hour).Err() // Session expires after 1 hour
}

// DeleteSession deletes the session (token) from Redis for a given email
func (r *Redis) DeleteSession(email string) error {
	key := "session:" + email
	return r.Client.Del(context.Background(), key).Err()
}
