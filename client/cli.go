package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

// SendCLI sends a message from the CLI/pipe to the server.
func SendCLI(host, port, email, password, to, msg string) error {
	addr := net.JoinHostPort(host, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Consume welcome message
	_, _ = reader.ReadString('\n')
	_, _ = reader.ReadString('\n')

	// 1. Login
	fmt.Fprintf(conn, "/login %s %s\n", email, password)
	resp, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(resp, "OK LOGIN") {
		return fmt.Errorf("login failed: %s", resp)
	}

	// 2. Determine target
	if msg == "" {
		// Read from stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			// Interactive terminal, ask for input
			fmt.Print("Enter message: ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				msg = scanner.Text()
			}
		} else {
			// Piped input
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			msg = string(data)
		}
	}

	if msg == "" {
		return fmt.Errorf("empty message")
	}

	// 3. Send
	if strings.HasPrefix(to, "@") {
		// Personal message
		target := strings.TrimPrefix(to, "@")
		fmt.Fprintf(conn, "/send %s %s\n", target, msg)
	} else {
		// Group message
		fmt.Fprintf(conn, "/group %s\n", to)
		resp, _ = reader.ReadString('\n') // OK GROUP ...
		if !strings.HasPrefix(resp, "OK GROUP") {
			return fmt.Errorf("failed to open group: %s", resp)
		}
		// Ready for history? skip or read?
		for {
			line, _ := reader.ReadString('\n')
			if strings.HasPrefix(line, "OK GROUP READY") {
				break
			}
		}
		fmt.Fprintf(conn, "%s\n", msg)
	}

	fmt.Println("Message sent successfully.")
	return nil
}
