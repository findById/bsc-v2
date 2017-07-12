package main

import (
	"flag"
	"log"
	"net"
	"time"
)

var (
	server = flag.String("s", "", "server address")
	target = flag.String("t", "", "target service address")
)

func main() {
	flag.Parse()
	if *server == "" || *target == "" {
		flag.PrintDefaults()
		return
	}
	serverAddr, err := net.ResolveTCPAddr("tcp", *server)
	if err != nil {
		log.Println(err)
		return
	}
	targetAddr, err := net.ResolveTCPAddr("tcp", *target)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("server addr:", serverAddr.String())
	log.Println("target addr:", targetAddr.String())
	aliveConn := 0
	exit := make(chan (int), 10)
	go NewDataConn(serverAddr, targetAddr, exit).do(false)
	go func() {
		for _ = range time.Tick(time.Second * 15) {
			log.Println("alive conns:", exit)
		}
	}()
	for {
		aliveConn += <-exit
		if aliveConn < 1 {
			break
		}
	}
	log.Println("JOB DONE.")
}
