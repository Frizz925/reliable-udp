package peer

import (
	"bufio"
	"errors"
	"net"
	"os"
	"reliable-udp/mux"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var ErrInterrupted = errors.New("interrupted")

type Peer struct {
	mux *mux.Mux

	lastAddr string

	mu sync.RWMutex
	wg *sync.WaitGroup

	newAddr chan string
	die     chan struct{}
}

// Peer fully controls the mux and its lifecycle.
// The mux should not be used again after being used by peer.
func New(m *mux.Mux) *Peer {
	return &Peer{
		mux:     m,
		wg:      &sync.WaitGroup{},
		newAddr: make(chan string, 1),
		die:     make(chan struct{}),
	}
}

func (p *Peer) Start() error {
	raddr, message, err := promptForInputs()
	if err != nil {
		return err
	}

	p.wg.Add(3)
	go p.producerRoutine(raddr)
	go p.consumerRoutine(message)
	go p.listenRoutine()

	return nil
}

func (p *Peer) Stop() error {
	if err := p.mux.Close(); err != nil {
		return err
	}
	close(p.die)
	p.wg.Wait()
	return nil
}

func (p *Peer) producerRoutine(raddr *net.UDPAddr) {
	defer p.wg.Done()
	if err := p.handleProducer(raddr); err != nil {
		log.Errorf("Producer error: %+v", err)
	}
}

func (p *Peer) handleProducer(raddr *net.UDPAddr) error {
	session, err := p.mux.OpenSession(raddr)
	if err != nil {
		return err
	}
	defer session.Close()
	conn, err := session.OpenStream()
	if err != nil {
		return err
	}
	cmd := NewCommander(conn)
	if _, err := cmd.Call("set"); err != nil {
		return err
	}

	interval := 15 * time.Second
	for {
		addr, err := cmd.Call("get")
		if err != nil {
			return err
		}
		if addr == "" {
			time.Sleep(interval)
			continue
		}

		p.mu.RLock()
		lastAddr := p.lastAddr
		p.mu.RUnlock()

		// No new peer, just sleep
		if lastAddr == addr {
			time.Sleep(interval)
			continue
		}

		p.mu.Lock()
		p.lastAddr = addr
		p.mu.Unlock()

		select {
		case p.newAddr <- addr:
		case <-p.die:
			return ErrInterrupted
		}

		time.Sleep(interval)
	}
}

func (p *Peer) consumerRoutine(message string) {
	defer p.wg.Done()
	for {
		select {
		case addr := <-p.newAddr:
			p.wg.Add(1)
			go p.messageRoutine(addr, message)
		case <-p.die:
			log.Errorf("Consumer error: %+v", ErrInterrupted)
			return
		}
	}
}

func (p *Peer) messageRoutine(addr string, message string) {
	defer p.wg.Done()
	if err := p.sendMessage(addr, message); err != nil {
		log.Errorf("Peer error: %+v", err)
	}
}

func (p *Peer) sendMessage(addr string, message string) error {
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	session, err := p.mux.OpenSession(raddr)
	if err != nil {
		return err
	}
	defer session.Close()
	conn, err := session.OpenStream()
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(message)); err != nil {
		return err
	}
	log.Infof("Sent message to peer at %s", addr)
	return nil
}

func (p *Peer) listenRoutine() {
	defer p.wg.Done()
	for {
		conn, err := p.mux.Accept()
		if err != nil {
			log.Errorf("Listen error: %+v", err)
			return
		}
		log.Infof("Accepted new peer: %s", conn.RemoteAddr())
		p.wg.Add(1)
		go p.peerRoutine(conn)
	}
}

func (p *Peer) peerRoutine(conn net.Conn) {
	defer p.wg.Done()
	if err := p.handlePeer(conn); err != nil {
		log.Errorf("Peer error: %+v", err)
	}
}

func (p *Peer) handlePeer(conn net.Conn) error {
	buf := make([]byte, 65535)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	message := string(buf[:n])
	log.Infof("Received message from %s: %s", conn.RemoteAddr(), message)
	return nil
}

func promptForInputs() (*net.UDPAddr, string, error) {
	con := bufio.NewReadWriter(
		bufio.NewReader(os.Stdin),
		bufio.NewWriter(os.Stdout),
	)
	addr := prompt(con, "Enter the tracker address: ", "127.0.0.1:7500")
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, "", err
	}
	message := prompt(con, "Enter the message to send to your peers: ", "Hello, world!")
	return raddr, message, nil
}

func prompt(con *bufio.ReadWriter, message string, fallback string) string {
	if _, err := con.WriteString(message); err != nil {
		return fallback
	}
	if err := con.Flush(); err != nil {
		return fallback
	}
	read, err := con.ReadString('\n')
	if err != nil {
		con.WriteString(fallback + "\n")
		con.Flush()
		return fallback
	}
	return strings.TrimSpace(read)
}
