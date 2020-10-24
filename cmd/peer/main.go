package main

import (
	"os"
	"reliable-udp/mux"
	"reliable-udp/peer"
	"reliable-udp/protocol"
	"reliable-udp/util"

	log "github.com/sirupsen/logrus"
)

func main() {
	if err := start(); err != nil {
		log.Fatal(err)
	}
}

func start() error {
	priv, err := protocol.ReadPrivateKey(nil)
	if err != nil {
		return err
	}
	m, err := mux.New(nil, priv)
	if err != nil {
		return err
	}
	p := peer.New(m)
	if err := p.Start(); err != nil {
		return err
	}
	defer p.Stop()
	log.Infof("Received signal %+v", util.WaitForSignal())
	return nil
}

func init() {
	log.SetOutput(os.Stderr)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}
