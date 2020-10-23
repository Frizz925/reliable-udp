package tracker

import (
	"fmt"
	"net"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Tracker struct {
	mu       sync.RWMutex
	lastAddr string
}

func New() *Tracker {
	return &Tracker{}
}

func (t *Tracker) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		log.Infof("Received connection from %s", conn.RemoteAddr())
		go t.clientWorker(conn)
	}
}

func (t *Tracker) clientWorker(conn net.Conn) {
	if err := t.handleClient(conn); err != nil {
		log.Errorf("Client error: %+v", err)
	}
}

func (t *Tracker) handleClient(conn net.Conn) error {
	buf := make([]byte, 65535)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}
		b := t.handlePacket(conn.RemoteAddr(), buf[:n])
		if _, err := conn.Write(b); err != nil {
			return err
		}
	}
}

func (t *Tracker) handlePacket(addr net.Addr, b []byte) []byte {
	method := strings.SplitN(string(b), " ", 2)[0]
	resp, err := t.handleRequest(addr, method)
	if err != nil {
		resp = fmt.Sprintf("error %s", err)
	}
	return []byte(resp)
}

func (t *Tracker) handleRequest(addr net.Addr, method string) (string, error) {
	log.Infof("Handling request from %s: %s", addr, method)
	switch method {
	case "get":
		t.mu.RLock()
		lastAddr := t.lastAddr
		t.mu.RUnlock()
		if lastAddr == addr.String() {
			return "ok", nil
		}
		return fmt.Sprintf("ok %s", lastAddr), nil
	case "set":
		t.mu.Lock()
		t.lastAddr = addr.String()
		t.mu.Unlock()
		return "ok", nil
	case "keepalive":
		return "ok", nil
	default:
		return "", fmt.Errorf("unknown method: %s", method)
	}
}

func Serve(l net.Listener) error {
	return New().Serve(l)
}
