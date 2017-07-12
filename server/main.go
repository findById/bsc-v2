package main

import (
	"flag"
	"log"
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
	server := NewProxyServer()
	server.Start(*dataPort, *userPort)
	select {
	}
}
