package main

import (
	"flag"
	"log"
	"crypto/md5"
	"encoding/base64"
)

var (
	dataPort = flag.String("dp", "", "data port")
	userPort = flag.String("up", "", "user port")

	username = flag.String("u", "", "username")
	password = flag.String("p", "", "password")
)

func main() {
	flag.Parse()
	if *dataPort == "" || *userPort == "" {
		flag.PrintDefaults()
		return
	}

	log.Println(*dataPort)
	log.Println(*userPort)

	h := md5.New().Sum([]byte(*username + ":" + *password))
	b := base64.StdEncoding.EncodeToString(h)

	server := NewProxyServer(b)
	server.Start(*dataPort, *userPort)
	select {
	}
}
