package peer

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
)

type Commander struct {
	conn   net.Conn
	mu     sync.Mutex
	buffer []byte
}

func NewCommander(conn net.Conn) *Commander {
	return &Commander{
		conn:   conn,
		buffer: make([]byte, 65535),
	}
}

func (c *Commander) Call(method string, params ...string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cmd := method
	if len(params) > 0 {
		cmd = fmt.Sprintf("%s %s", method, strings.Join(params, " "))
	}

	if _, err := c.conn.Write([]byte(cmd)); err != nil {
		return "", err
	}
	n, err := c.conn.Read(c.buffer)
	if err != nil {
		return "", err
	}

	resp := string(c.buffer[:n])
	if strings.HasPrefix(resp, "error") {
		message := strings.TrimSpace(resp[5:])
		return "", errors.New(message)
	} else if strings.HasPrefix(resp, "ok") {
		result := strings.TrimSpace(resp[2:])
		return result, nil
	}
	return "", fmt.Errorf("unknown response: %s", resp)
}
