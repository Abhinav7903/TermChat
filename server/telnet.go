package server

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"termchat/factory"
	"termchat/pkg/users"
	"time"

)

// notifyChannel returns the per-user Redis channel for incoming notifications.
func notifyChannel(username string) string {
	return fmt.Sprintf("notify:%s", strings.ToLower(strings.TrimSpace(username)))
}

func handleTelnetClient(conn net.Conn, srv *Server) {
	defer conn.Close()

	// Unique session ID for this connection.
	// Embedded in every published message so the publisher can ignore
	// their own echoes even if they have multiple tabs open.
	// Use remote address + connection time as unique session ID (no extra deps needed)
	sessionID := fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), time.Now().UnixNano())

	conn.Write([]byte("Welcome to TermChat CLI over Telnet!\n"))
	conn.Write([]byte("Commands: /register <email> <username> <password>, /login <email> <password>, /chat <user>, /tempchat <user>, /send <user> <message>, /room, /search <prefix>, /exit\n"))

	reader := bufio.NewReader(conn)
	var currentUser *factory.User

	var notifyCancel context.CancelFunc
	stopNotify := func() {
		if notifyCancel != nil {
			notifyCancel()
			notifyCancel = nil
		}
	}
	defer stopNotify()

	for {
		conn.Write([]byte("> "))
		line, err := reader.ReadString('\n')
		if err != nil {
			srv.logger.Error("Client disconnected", "error", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		args := strings.SplitN(line, " ", 2)
		cmd := args[0]
		argLine := ""
		if len(args) > 1 {
			argLine = args[1]
		}

		switch cmd {

		// =====================================================
		// EXIT
		// =====================================================
		case "/exit":
			stopNotify()
			conn.Write([]byte("OK EXIT\n"))
			return

		// =====================================================
		// REGISTER
		// =====================================================
		case "/register":
			parts := strings.Fields(argLine)
			if len(parts) != 3 {
				conn.Write([]byte("ERR REGISTER invalid_arguments\n"))
				continue
			}
			email, username, password := parts[0], parts[1], parts[2]
			hashed, err := users.HashPassword(password)
			if err != nil {
				conn.Write([]byte("ERR REGISTER hash_failed\n"))
				continue
			}
			user := factory.User{
				Email:          email,
				Name:           username,
				HashedPassword: hashed,
			}
			if err := srv.user.CreateUser(user); err != nil {
				conn.Write([]byte(fmt.Sprintf("ERR REGISTER %s\n", err)))
			} else {
				conn.Write([]byte("OK REGISTER\n"))
			}

		// =====================================================
		// LOGIN
		// =====================================================
		case "/login":
			parts := strings.Fields(argLine)
			if len(parts) != 2 {
				conn.Write([]byte("ERR LOGIN invalid_arguments\n"))
				continue
			}
			email, password := parts[0], parts[1]
			user := factory.User{Email: email, Password: password}
			loggedInUser, err := srv.user.Login(user)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("ERR LOGIN %s\n", err)))
				continue
			}
			currentUser = &loggedInUser
			conn.Write([]byte(fmt.Sprintf("OK LOGIN %s\n", currentUser.Name)))

			// Start per-user notification listener
			stopNotify()
			{
				myName := currentUser.Name
				nCtx, nCancel := context.WithCancel(context.Background())
				notifyCancel = nCancel
				go func() {
					ch := notifyChannel(myName)
					ps := srv.redis.Client.Subscribe(nCtx, ch)
					defer ps.Close()
					mc := ps.Channel()
					for {
						select {
						case <-nCtx.Done():
							return
						case msg, ok := <-mc:
							if !ok {
								return
							}
							conn.Write([]byte("NOTIFY " + msg.Payload + "\n"))
						}
					}
				}()
			}

		// =====================================================
		// ROOM LIST
		// =====================================================
		case "/room":
			if currentUser == nil {
				conn.Write([]byte("ERR AUTH not_logged_in\n"))
				continue
			}
			rooms, err := srv.message.GetChatPartners(currentUser.ID)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("ERR ROOM %s\n", err)))
				continue
			}
			if len(rooms) == 0 {
				conn.Write([]byte("ROOM NONE\n"))
				continue
			}
			for _, name := range rooms {
				conn.Write([]byte(fmt.Sprintf("ROOM %s\n", name)))
			}

		// =====================================================
		// SEND DIRECT MESSAGE
		// =====================================================
		case "/send":
			if currentUser == nil {
				conn.Write([]byte("ERR AUTH not_logged_in\n"))
				continue
			}
			parts := strings.SplitN(argLine, " ", 2)
			if len(parts) != 2 {
				conn.Write([]byte("ERR SEND invalid_arguments\n"))
				continue
			}
			receiver, msg := parts[0], parts[1]
			if err := srv.message.SendPersonalMessage(currentUser.Name, receiver, msg); err != nil {
				conn.Write([]byte(fmt.Sprintf("ERR SEND %s\n", err)))
			} else {
				payload := fmt.Sprintf("MSG %s", currentUser.Name)
				_ = srv.redis.Client.Publish(context.Background(), notifyChannel(receiver), payload).Err()
				conn.Write([]byte("OK SEND\n"))
			}

		// =====================================================
		// CHAT — persistent with history + live Redis
		//
		// Payload format (Redis): <sessionID>|<sender>|<timestamp>|<content>
		// Client protocol:
		//   ← OK CHAT <partner>
		//   ← HIST <timestamp>|<sender>|<content>
		//   ← OK CHAT READY
		//   ← MSG <sender>|<timestamp>|<content>    (live)
		//   ← OK CHAT EXIT
		// =====================================================
		case "/chat":
			if currentUser == nil {
				conn.Write([]byte("ERR AUTH not_logged_in\n"))
				continue
			}
			chatPartner := strings.TrimSpace(argLine)
			if chatPartner == "" {
				conn.Write([]byte("ERR CHAT invalid_arguments\n"))
				continue
			}

			// Notify partner
			_ = srv.redis.Client.Publish(
				context.Background(),
				notifyChannel(chatPartner),
				fmt.Sprintf("CHAT %s", currentUser.Name),
			).Err()

			conn.Write([]byte(fmt.Sprintf("OK CHAT %s\n", chatPartner)))

			// History
			messages, err := srv.message.GetMessagesBetweenUsers(currentUser.Name, chatPartner)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("ERR CHAT history_failed %s\n", err)))
				continue
			}
			for _, m := range messages {
				conn.Write([]byte(fmt.Sprintf("HIST %s|%s|%s\n", m.SentAt, m.SenderName, m.Content)))
			}
			conn.Write([]byte("OK CHAT READY\n"))

			chatID, err := srv.message.GetChatID(currentUser.Name, chatPartner)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("ERR CHAT pubsub_failed %s\n", err)))
				continue
			}

			channelName := fmt.Sprintf("chat:%d", chatID)
			ctx := context.Background()
			pubsub := srv.redis.Client.Subscribe(ctx, channelName)
			msgChan := pubsub.Channel()

			done := make(chan struct{})
			var once sync.Once
			safeClose := func() { once.Do(func() { close(done) }) }

			mySessionID := sessionID // capture for goroutine

			// Goroutine: forward messages from OTHER sessions only.
			// Payload: <sessionID>|<sender>|<timestamp>|<content>
			go func() {
				defer pubsub.Close()
				for {
					select {
					case <-done:
						return
					case msg, ok := <-msgChan:
						if !ok {
							return
						}
						// Split into 4 parts: sessionID | sender | timestamp | content
						segs := strings.SplitN(msg.Payload, "|", 4)
						if len(segs) != 4 {
							continue
						}
						fromSession, sender, ts, content := segs[0], segs[1], segs[2], segs[3]
						// Skip our own session's messages (already echoed optimistically on client)
						if fromSession == mySessionID {
							continue
						}
						conn.Write([]byte(fmt.Sprintf("MSG %s|%s|%s\n", sender, ts, content)))
					}
				}
			}()

			for {
				select {
				case <-done:
					goto ChatExit
				default:
				}

				msgLine, err := reader.ReadString('\n')
				if err != nil {
					srv.logger.Error("Chat input failed", "error", err)
					safeClose()
					break
				}
				msgLine = strings.TrimSpace(msgLine)
				if msgLine == "" {
					continue
				}
				if msgLine == "/exit" {
					safeClose()
					break
				}

				senderName := currentUser.Name
				if err := srv.message.SendPersonalMessage(senderName, chatPartner, msgLine); err != nil {
					conn.Write([]byte(fmt.Sprintf("ERR CHAT send_failed %s\n", err)))
					continue
				}

				// Publish with session ID prefix so the goroutine above can skip it
				ts := time.Now().Format("2006-01-02 15:04:05")
				payload := fmt.Sprintf("%s|%s|%s|%s", mySessionID, senderName, ts, msgLine)
				_ = srv.redis.Client.Publish(ctx, channelName, payload).Err()
			}

		ChatExit:
			conn.Write([]byte("OK CHAT EXIT\n"))

		// =====================================================
		// TEMP CHAT — ephemeral, no DB save
		//
		// Payload format (Redis): <sessionID>|<sender>|<timestamp>|<content>
		//                    or:  <sessionID>|<sender>|/close
		// =====================================================
		case "/tempchat":
			if currentUser == nil {
				conn.Write([]byte("ERR AUTH not_logged_in\n"))
				continue
			}
			chatPartner := strings.TrimSpace(argLine)
			if chatPartner == "" {
				conn.Write([]byte("ERR TEMPCHAT invalid_arguments\n"))
				continue
			}

			// Notify partner
			_ = srv.redis.Client.Publish(
				context.Background(),
				notifyChannel(chatPartner),
				fmt.Sprintf("TEMPCHAT %s", currentUser.Name),
			).Err()

			channelName := makeTempChatChannel(currentUser.Name, chatPartner)
			conn.Write([]byte(fmt.Sprintf("OK TEMPCHAT %s\n", chatPartner)))

			ctx := context.Background()
			pubsub := srv.redis.Client.Subscribe(ctx, channelName)
			msgChan := pubsub.Channel()

			done := make(chan struct{})
			var once sync.Once
			safeClose := func() { once.Do(func() { close(done) }) }

			mySessionID := sessionID

			go func() {
				defer pubsub.Close()
				for {
					select {
					case <-done:
						return
					case msg, ok := <-msgChan:
						if !ok {
							return
						}
						// Payload: <sessionID>|<rest...>
						firstPipe := strings.Index(msg.Payload, "|")
						if firstPipe == -1 {
							continue
						}
						fromSession := msg.Payload[:firstPipe]
						rest := msg.Payload[firstPipe+1:]

						// Skip our own session
						if fromSession == mySessionID {
							continue
						}
						conn.Write([]byte("MSG " + rest + "\n"))
					}
				}
			}()

			senderName := currentUser.Name
			for {
				select {
				case <-done:
					goto TempExit
				default:
				}

				msgLine, err := reader.ReadString('\n')
				if err != nil {
					safeClose()
					break
				}
				msgLine = strings.TrimSpace(msgLine)
				if msgLine == "" {
					continue
				}

				if msgLine == "/exit" {
					// Publish close signal (with session prefix)
					payload := fmt.Sprintf("%s|%s|/close", mySessionID, senderName)
					_ = srv.redis.Client.Publish(ctx, channelName, payload).Err()
					safeClose()
					break
				}

				ts := time.Now().Format("2006-01-02 15:04:05")
				payload := fmt.Sprintf("%s|%s|%s|%s", mySessionID, senderName, ts, msgLine)
				_ = srv.redis.Client.Publish(ctx, channelName, payload).Err()
			}

		TempExit:
			conn.Write([]byte("OK TEMPCHAT EXIT\n"))

		// =====================================================
		// SEARCH
		// =====================================================
		case "/search":
			if currentUser == nil {
				conn.Write([]byte("ERR AUTH not_logged_in\n"))
				continue
			}
			keyword := strings.TrimSpace(argLine)
			if keyword == "" {
				conn.Write([]byte("ERR SEARCH invalid_arguments\n"))
				continue
			}
			usersFound, err := srv.user.SearchUsersByName(keyword)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("ERR SEARCH %s\n", err)))
				continue
			}
			if len(usersFound) == 0 {
				conn.Write([]byte("SEARCH NONE\n"))
				continue
			}
			for _, u := range usersFound {
				conn.Write([]byte(fmt.Sprintf("SEARCH %s %s\n", u.Name, u.Email)))
			}

		// =====================================================
		// UNKNOWN
		// =====================================================
		default:
			conn.Write([]byte("ERR UNKNOWN_COMMAND\n"))
		}
	}
}

func makeTempChatChannel(u1, u2 string) string {
	a := strings.ToLower(strings.TrimSpace(u1))
	b := strings.ToLower(strings.TrimSpace(u2))
	if a < b {
		return fmt.Sprintf("tempchat:%s:%s", a, b)
	}
	return fmt.Sprintf("tempchat:%s:%s", b, a)
}