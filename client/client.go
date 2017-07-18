package main

import (
	"crypto/md5"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

func newProxy(wg *sync.WaitGroup, p *Proxy, debug, noDelay bool, retry, interval int) {
	defer wg.Done()
	log.Println("[G] PROXY START WORK ", p.Target, "->", p.Server)
	hash := md5.New()
	authToken := hash.Sum([]byte(p.User + ":" + p.Password))
	for retry != 0 {
		serverAddr, err := net.ResolveTCPAddr("tcp", p.Server)
		if err != nil {
			log.Println(err)
		} else {
			targetAddr, err := net.ResolveTCPAddr("tcp", p.Target)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("[G] SERVER ADDR:", serverAddr.String())
				log.Println("[G] TARGET ADDR:", targetAddr.String())
				log.Println("[G] START PROXY:", targetAddr.String(), "->", serverAddr.String())
				NewDataConn(serverAddr, targetAddr, authToken, noDelay, debug, &connMonitor).do(false)
				log.Println("[G] PROXY JOB DONE.")
			}
		}
		if retry > 0 {
			retry--
		}
		_ = <-time.Tick(time.Second * time.Duration(interval))

		log.Println("[G] RETRY ", p.Target, "->", p.Server)
	}
}

func runClient(conf *Config) {
	log.Printf("[G] PID:%v UID: %v\n", os.Getpid(), os.Getuid())
	log.Println("[G] DEBUG  MODE:", conf.Debug)
	log.Println("[G] TCP NODELAY:", conf.Nodelay)
	log.Println("[G] PROXY COUNT:", len(conf.Proxies))
	wg := &sync.WaitGroup{}
	for _, proxy := range conf.Proxies {
		if proxy.User == "" {
			proxy.User = conf.User
		}
		if proxy.Password == "" {
			proxy.Password = conf.Password
		}
		wg.Add(1)
		go newProxy(wg, proxy, conf.Debug, conf.Nodelay, conf.Retry, conf.Interval)
	}
	wg.Wait()
	log.Println("[G] CLIENT JOB DONE.")
}
