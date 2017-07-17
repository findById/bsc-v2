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
	debug = flag.Bool("d", false, "debug, default false")
	username = flag.String("u", "", "username")
	password = flag.String("p", "", "password")
)

func main() {
	flag.Parse()
	if *dataPort == "" || *userPort == "" {
		flag.PrintDefaults()
		return
	}

	log.Println("Accepting data connections at:", *dataPort)
	log.Println("Accepting user connections at:", *userPort)

	h := md5.New().Sum([]byte(*username + ":" + *password))
	b := base64.StdEncoding.EncodeToString(h)

	server := NewProxyServer(b, *debug)
	server.Start(*dataPort, *userPort)
	select {
	}
}
