package server

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"termchat/factory"
	"termchat/pkg/users"
	"time"
)

func handleTelnetClient(conn net.Conn, srv *Server) {
	defer conn.Close()

	conn.Write([]byte("Welcome to TermChat CLI over Telnet!\n"))
	conn.Write([]byte("Commands: /register <email> <username> <password>, /login <email> <password>, /chat <user>, /send <user> <message>, /room, /exit\n"))

	reader := bufio.NewReader(conn)
	var currentUser *factory.User

	for {
		conn.Write([]byte("> "))
		line, err := reader.ReadString('\n')
		if err != nil {
			srv.logger.Error("Client disconnected", "error", err)
			return
		}
		line = strings.TrimSpace(line)

		args := strings.SplitN(line, " ", 2)
		cmd := args[0]
		argLine := ""
		if len(args) > 1 {
			argLine = args[1]
		}

		switch cmd {
		case "/exit":
			conn.Write([]byte("Bye!\n"))
			return

		case "/register", "/signup", "/create", "/new", "/add":
			parts := strings.Split(argLine, " ")
			if len(parts) != 3 {
				conn.Write([]byte("Usage: /register <email> <username> <password>\n"))
				continue
			}
			email, username, password := parts[0], parts[1], parts[2]
			hashed, err := users.HashPassword(password)
			if err != nil {
				conn.Write([]byte("Failed to hash password\n"))
				continue
			}
			user := factory.User{Email: email, Name: username, HashedPassword: hashed}
			err = srv.user.CreateUser(user)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("Register failed: %s\n", err)))
			} else {
				conn.Write([]byte("User registered successfully\n"))
			}

		case "/login", "/signin":
			parts := strings.Split(argLine, " ")
			if len(parts) != 2 {
				conn.Write([]byte("Usage: /login <email> <password>\n"))
				continue
			}
			email, password := parts[0], parts[1]
			user := factory.User{Email: email, Password: password}
			loggedInUser, err := srv.user.Login(user)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("Login failed: %s\n", err)))
			} else {
				currentUser = &loggedInUser
				conn.Write([]byte(fmt.Sprintf("Welcome %s!\n", currentUser.Name)))
			}
			continue

			// case "/chat":
			if currentUser == nil {
				conn.Write([]byte("Please login first\n"))
				continue
			}
			chatPartner := strings.TrimSpace(argLine)
			if chatPartner == "" {
				conn.Write([]byte("Usage: /chat <username>\n"))
				continue
			}

			messages, err := srv.message.GetMessagesBetweenUsers(currentUser.Name, chatPartner)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("Failed to get messages: %s\n", err)))
				continue
			}

			conn.Write([]byte(fmt.Sprintf("----- Chat with %s -----\n", chatPartner)))
			var lastSeen time.Time
			for _, m := range messages {
				prefix := m.SenderName
				if m.SenderID == currentUser.ID {
					prefix = "You"
				}
				conn.Write([]byte(fmt.Sprintf("[%s] %s: %s\n", m.SentAt, prefix, m.Content)))
				parsedTime, _ := time.Parse("2006-01-02 15:04:05", m.SentAt)
				if parsedTime.After(lastSeen) {
					lastSeen = parsedTime
				}
			}
			conn.Write([]byte("Type your message. Use /exit to leave chat.\n"))

			chatID, err := srv.message.GetChatID(currentUser.Name, chatPartner)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("Unable to get chat ID: %s\n", err)))
				continue
			}

			done := make(chan struct{})
			pubsub := srv.redis.Client.Subscribe(context.Background(), fmt.Sprintf("chat:%d", chatID))
			msgChan := pubsub.Channel()

			go func() {
				for {
					select {
					case <-done:
						pubsub.Close()
						return
					case msg := <-msgChan:
						parts := strings.SplitN(msg.Payload, "|", 3)
						if len(parts) != 3 {
							continue
						}
						username, timestamp, content := parts[0], parts[1], parts[2]
						if username != currentUser.Name {
							conn.Write([]byte(fmt.Sprintf("\n[%s] %s: %s\n", timestamp, username, content)))
							conn.Write([]byte(fmt.Sprintf("[%s]> ", chatPartner)))
						}
					}
				}
			}()

			for {
				conn.Write([]byte(fmt.Sprintf("[%s]> ", chatPartner)))
				msgLine, err := reader.ReadString('\n')
				if err != nil {
					srv.logger.Error("Chat input failed", "error", err)
					break
				}
				msgLine = strings.TrimSpace(msgLine)
				if msgLine == "/exit" {
					conn.Write([]byte("Exiting chat...\n"))
					done <- struct{}{}
					break
				}
				if msgLine != "" {
					err := srv.message.SendPersonalMessage(currentUser.Name, chatPartner, msgLine)
					if err != nil {
						conn.Write([]byte(fmt.Sprintf("Send failed: %s\n", err)))
					}
				}
			}

		case "/send", "/message", "/msg", "/dm", "/direct":
			if currentUser == nil {
				conn.Write([]byte("Please login first\n"))
				continue
			}
			parts := strings.SplitN(argLine, " ", 2)
			if len(parts) != 2 {
				conn.Write([]byte("Usage: /send <receiver_username> <message>\n"))
				continue
			}
			receiver, msg := parts[0], parts[1]
			err := srv.message.SendPersonalMessage(currentUser.Name, receiver, msg)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("Failed to send message: %s\n", err)))
			} else {
				conn.Write([]byte("Message sent.\n"))
			}

		case "/room", "/rooms", "/chatrooms", "/recent":
			if currentUser == nil {
				conn.Write([]byte("Please login first\n"))
				continue
			}
			rooms, err := srv.message.GetChatPartners(currentUser.ID)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("Failed to retrieve chat rooms: %s\n", err)))
				continue
			}
			if len(rooms) == 0 {
				conn.Write([]byte("No chat rooms found.\n"))
			} else {
				conn.Write([]byte("Chatting with:\n"))
				for _, name := range rooms {
					conn.Write([]byte("- " + name + "\n"))
				}
			}

		case "/last", "/history", "/last5":
			if currentUser == nil {
				conn.Write([]byte("Please login first\n"))
				continue
			}
			partner := strings.TrimSpace(argLine)
			if partner == "" {
				conn.Write([]byte("Usage: /last <username>\n"))
				continue
			}

			msgs, err := srv.message.GetLastMessagesBetweenUsers(currentUser.Name, partner, 5)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("Failed to fetch messages: %s\n", err)))
				continue
			}

			conn.Write([]byte(fmt.Sprintf("Last 5 messages with %s:\n", partner)))
			for _, m := range msgs {
				prefix := m.SenderName
				if m.SenderID == currentUser.ID {
					prefix = "You"
				}
				conn.Write([]byte(fmt.Sprintf("[%s] %s: %s\n", m.SentAt, prefix, m.Content)))
			}

		case "/help":
			conn.Write([]byte("Available commands:\n"))
			conn.Write([]byte("/register <email> <username> <password> - Register a new user\n"))
			conn.Write([]byte("/login <email> <password> - Login to your account\n"))
			conn.Write([]byte("/chat <username> - Start a chat with a user\n"))
			conn.Write([]byte("/send <username> <message> - Send a message to a user\n"))
			conn.Write([]byte("/room - List your chat rooms\n"))
			conn.Write([]byte("/last <username> - Show last 5 messages with a user\n"))
			conn.Write([]byte("/exit - Exit the CLI\n"))

		case "/clear":
			if currentUser == nil {
				conn.Write([]byte("Please login first\n"))
				continue
			}
			conn.Write([]byte("\033[H\033[2J")) // ANSI escape code to clear terminal
			conn.Write([]byte("Terminal cleared.\n"))

		case "/ping":
			conn.Write([]byte("PONG\n"))

		case "/whoami":
			if currentUser == nil {
				conn.Write([]byte("Please login first\n"))
				continue
			}
			conn.Write([]byte(fmt.Sprintf("You are logged in as: %s \n", currentUser.Name)))

		case "/version":
			conn.Write([]byte("TermChat CLI over Telnet v1.0\n"))

		case "/about":
			conn.Write([]byte("TermChat is a terminal-based chat application.\n"))
			conn.Write([]byte("It allows users to register, log in, and chat with others in real time via a simple Telnet interface.\n"))
			conn.Write([]byte("This project is open-source and intended for learning, experimentation, or lightweight internal use.\n"))
			conn.Write([]byte("Developed by Abhinav.\n"))
			conn.Write([]byte("Use /help to view available commands.\n"))

		default:
			conn.Write([]byte("Unknown command\n" +
				"Type /help for a list of commands.\n"))
		}
	}
}
