package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"reliable-udp/protocol/wire"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-ch
		log.Infof("Received signal %+v", sig)
		cancel()
	}()
	if err := serve(ctx); err != nil {
		log.Fatal(err)
	}
}

func serve(ctx context.Context) error {
	l, err := wire.Listen("")
	if err != nil {
		return err
	}
	cherr := make(chan error)
	go func() {
		<-ctx.Done()
		l.Close()
		cherr <- l.Close()
	}()
	for {
		p, err := l.Accept()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
	}
	return <-cherr
}
