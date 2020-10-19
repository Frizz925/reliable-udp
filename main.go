package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"

	log "github.com/sirupsen/logrus"
)

func main() {
	if err := start(); err != nil {
		log.Fatal(err)
	}
}

func start() error {
	privKey, err := rsa.GenerateKey(rand.Reader, 128)
	if err != nil {
		return err
	}
	rsa.VerifyPKCS1v15()
	rsa.SignPKCS1v15()
	x509.MarshalPKCS1PublicKey()
}
