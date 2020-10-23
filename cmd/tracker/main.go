package main

import (
	"net"
	"reliable-udp/mux"
	"reliable-udp/protocol"
	"reliable-udp/tracker"
	"reliable-udp/util"

	log "github.com/sirupsen/logrus"
)

func main() {
	if err := startTracker(); err != nil {
		log.Fatal(err)
	}
}

func startTracker() error {
	priv, err := protocol.ReadPrivateKey(nil)
	if err != nil {
		return err
	}
	laddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:7500")
	if err != nil {
		return err
	}
	m, err := mux.New(laddr, priv)
	if err != nil {
		return err
	}
	defer m.Close()
	go tracker.Serve(m)
	log.Infof("Tracker serving at %s", m.Addr())
	log.Infof("Received signal %+v", util.WaitForSignal())
	return nil
}

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}
