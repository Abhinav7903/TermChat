package client

import (
	"bufio"
	"net"
)

// Connect opens a TCP connection to the TermChat server.
func Connect(host, port string) (net.Conn, error) {
	return net.Dial("tcp", host+":"+port)
}

// ReadLoop reads newline-delimited lines from the server and sends each one
// to inCh. It closes inCh when the connection is closed or an error occurs.
func ReadLoop(conn net.Conn, inCh chan<- string) {
	defer close(inCh)
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		inCh <- scanner.Text()
	}
}

// Write sends a single line (with trailing newline) to the server.
func Write(conn net.Conn, line string) error {
	_, err := conn.Write([]byte(line + "\n"))
	return err
}
