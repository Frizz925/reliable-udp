package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"reliable-udp/client"
	"reliable-udp/mux"
	"reliable-udp/protocol"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-ch
		log.Infof("Received signal %+v", sig)
		cancel()
	}()

	if err := start(ctx); err != nil {
		log.Fatal(err)
	}
}

func start(ctx context.Context) error {
	pk, err := protocol.ReadPrivateKey(nil)
	if err != nil {
		return err
	}
	m := mux.New(pk)

	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	go m.Listen(conn)

	saddr := conn.LocalAddr().(*net.UDPAddr)
	if err := startClient(saddr, pk); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func startClient(raddr *net.UDPAddr, pk protocol.PrivateKey) error {
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	c, err := client.New(conn, pk)
	if err != nil {
		return err
	}
	content := []byte("Hello, world!")
	return c.Send(protocol.NewRaw(protocol.FrameRaw, content), raddr)
}

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}
