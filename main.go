package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"reliable-udp/mux"
	"reliable-udp/protocol"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func main() {
	if err := start(); err != nil {
		log.Fatal(err)
	}
}

func start() error {
	ms, err := startServer(nil)
	if err != nil {
		return err
	}
	defer ms.Close()
	go listenWorker(ms)

	port := ms.Addr().(*net.UDPAddr).Port
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	mc, err := startClient()
	if err != nil {
		return err
	}
	defer mc.Close()
	clientWorker(mc, raddr)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	sig := <-ch
	log.Infof("Received signal %+v", sig)

	return nil
}

func startServer(laddr *net.UDPAddr) (*mux.Mux, error) {
	priv, _, err := generateKeyPair()
	if err != nil {
		return nil, err
	}
	m, err := mux.New(laddr, priv)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func listenWorker(m *mux.Mux) {
	for {
		session, err := m.AcceptSession()
		if err != nil {
			log.Errorf("Listen error: %+v", err)
			return
		}
		sessionWorker(session)
	}
}

func sessionWorker(session *mux.Session) {
	defer session.Close()
	conn, err := session.Accept()
	if err != nil {
		log.Errorf("Session error: %+v", err)
		return
	}
	streamWorker(conn)
}

func streamWorker(conn *mux.Stream) {
	defer conn.Close()
	if err := serve(conn); err != nil {
		log.Errorf("Serve error: %+v", err)
	}
}

func serve(conn net.Conn) error {
	defer conn.Close()
	buf := make([]byte, 65535)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	content := buf[:n]
	log.Infof("Server received: %s", string(content))
	log.Infof("Server sending: %s", string(content))
	if _, err := conn.Write(content); err != nil {
		return err
	}
	return nil
}

func startClient() (*mux.Mux, error) {
	priv, _, err := generateKeyPair()
	if err != nil {
		return nil, err
	}
	m, err := mux.New(nil, priv)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func clientWorker(m *mux.Mux, raddr *net.UDPAddr) {
	if err := runClient(m, raddr); err != nil {
		log.Errorf("Client error: %+v", err)
	}
}

func runClient(m *mux.Mux, raddr *net.UDPAddr) error {
	session, err := m.OpenSession(raddr)
	if err != nil {
		return err
	}
	defer session.Close()
	conn, err := session.OpenStream()
	if err != nil {
		return err
	}
	defer conn.Close()

	message := "Hello, world!"
	log.Infof("Client sending: %s", message)
	if _, err := conn.Write([]byte(message)); err != nil {
		return err
	}

	b := make([]byte, 65535)
	n, err := conn.Read(b)
	if err != nil {
		return err
	}
	content := b[:n]
	log.Infof("Client received: %s", string(content))

	return nil
}

func generateKeyPair() (priv protocol.PrivateKey, pub protocol.PublicKey, err error) {
	priv, err = protocol.ReadPrivateKey(nil)
	if err != nil {
		return
	}
	pub, err = priv.PublicKey()
	if err != nil {
		return
	}
	return priv, pub, nil
}

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}
