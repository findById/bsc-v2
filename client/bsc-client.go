package main

import (
	"crypto/md5"
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"
)

var (
	server  = flag.String("s", "", "server address")
	target  = flag.String("t", "", "target service address")
	user    = flag.String("u", "", "user name")
	token   = flag.String("p", "", "auth token")
	install = flag.Bool("i", false, "install with systemd")
	debug   = flag.Bool("d", false, "debug mode,default false")
	nodelay = flag.Bool("nodelay", true, "tcp nodelay,default true")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
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
	log.Printf("[G] PID:%v UID: %v\n", os.Getpid(), os.Getuid())
	log.Println("[G] DEBUG  MODE:", *debug)
	log.Println("[G] TCP NODELAY:", *nodelay)
	log.Println("[G] SERVER ADDR:", serverAddr.String())
	log.Println("[G] TARGET ADDR:", targetAddr.String())
	hash := md5.New()
	authToken := hash.Sum([]byte(*user + ":" + *token))
	go func() {
		http.ListenAndServe(":6060", nil)
	}()
	aliveConn := 0
	go func() {
		for _ = range time.Tick(time.Second * 5) {
			log.Println("[G] alive conns:", aliveConn, "goroutine:", runtime.NumGoroutine())
		}
	}()
	for {
		log.Println("[G] START WORK.")
		exit := make(chan (int), 10)
		go NewDataConn(serverAddr, targetAddr, authToken, *nodelay, *debug, exit).do(false)
		for n := range exit {
			aliveConn = aliveConn + n
			if aliveConn < 1 {
				break
			}
		}
		log.Println("[G] JOB DONE.")
		_ = <-time.Tick(time.Second * 30)
	}
}
