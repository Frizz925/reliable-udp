package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"reliable-udp/mux"
	"sync"

	log "github.com/sirupsen/logrus"
)

func main() {
	if err := start(); err != nil {
		log.Fatal(err)
	}
}

func start() error {
	laddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	ms, err := mux.New(laddr)
	if err != nil {
		return err
	}
	go server(wg, ms)

	mc, err := mux.New(laddr)
	if err != nil {
		return err
	}
	client(wg, mc, ms.LocalAddr())

	if err := mc.Close(); err != nil {
		return err
	}
	if err := ms.Close(); err != nil {
		return err
	}
	wg.Wait()

	return nil
}

func server(wg *sync.WaitGroup, m *mux.Mux) {
	defer wg.Done()
	log.Infof("Server listening at: %s", m.LocalAddr())
	for {
		s, err := m.Accept()
		if err != nil {
			if err != io.EOF {
				log.Errorf("Server error: %+v", err)
			}
			return
		}
		go serverSession(s)
	}
}

func serverSession(s *mux.Session) {
	log.Debugf("[%s] accept", s)
	for {
		conn, err := s.Accept()
		if err != nil {
			if err != io.EOF {
				log.Errorf("[%s] Server session error: %+v", s, err)
			}
			return
		}
		go serverStream(s, conn)
	}
}

func serverStream(s *mux.Session, conn *mux.Stream) {
	log.Debugf("[%s] accept: StreamID: %d", s, conn.StreamID())
	b := make([]byte, 65535)
	for {
		if err := handleStream(conn, b); err != nil {
			if err != io.EOF {
				log.Errorf("[%s] Server stream error: %+v", s, err)
			}
			return
		}
	}
}

func handleStream(s *mux.Stream, b []byte) error {
	r, err := s.Read(b)
	if err != nil {
		return err
	}
	raw := b[:r]
	log.Infof("server: %s", string(raw))
	w, err := s.Write(raw)
	if err != nil {
		return err
	}
	if w != r {
		return fmt.Errorf("server: read -> write bytes mismatch, read: %d write: %d", r, w)
	}
	return nil
}

func client(wg *sync.WaitGroup, m *mux.Mux, raddr *net.UDPAddr) {
	defer wg.Done()
	if err := handleClient(m, raddr); err != nil {
		if err != io.EOF {
			log.Errorf("Client error: %+v", err)
		}
	}
}

func handleClient(m *mux.Mux, raddr *net.UDPAddr) error {
	content, err := ioutil.ReadFile("data/fischl.txt")
	if err != nil {
		return err
	}

	s, err := m.Session(raddr)
	if err != nil {
		return err
	}

	conn, err := s.OpenStream()
	if err != nil {
		return err
	}
	defer s.Close()
	log.Infof("Running client: %s", conn)

	off := 0
	b := make([]byte, 512)
	bsize := len(b) - 30
	for off < len(content) {
		chunk := content[off:]
		if len(chunk) > bsize {
			chunk = chunk[:bsize]
		}
		chunkLen := len(chunk)

		w, err := conn.Write(chunk)
		if err != nil {
			return err
		}
		if w != chunkLen {
			return fmt.Errorf("client: written bytes mismatch, content: %d write: %d", chunkLen, w)
		}
		off += w

		r, err := conn.Read(b)
		if err != nil {
			return err
		}
		if r != w {
			return fmt.Errorf("client: write -> read bytes mismatch, write: %d read: %d", w, r)
		}
		raw := b[:r]
		log.Infof("client: %s", string(raw))
	}

	return nil
}

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}
