package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"
)

type Proxy struct {
	Server   string
	Target   string
	User     string
	Password string
}

type Config struct {
	Proxies  []*Proxy
	User     string
	Password string
	Retry    int
	Interval int
	Debug    bool
	Nodelay  bool
}

var (
	conf    = flag.String("c", "config.json", "config file path,default config.json")
	install = flag.Bool("install", false, "install with systemd")
	profile = flag.Bool("p", false, "start profile http server @:6060")
	mode    = "c"

	connMonitor    = make(chan (int), 10)
	channelMonitor = make(chan (int), 20)
)

func loadConfig() (c *Config, err error) {
	f, err := os.Open(*conf)
	if err != nil {
		return
	}
	defer f.Close()
	dat, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}
	c = &Config{}
	err = json.Unmarshal(dat, c)
	return
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if *install {
		return
	}
	if *profile {
		go func() {
			http.ListenAndServe(":6060", nil)
		}()
	}
	aliveConn := 0
	aliveChannel := 0
	go func() {
		var n int
		for {
			select {
			case n = <-connMonitor:
				aliveConn += n
			case n = <-channelMonitor:
				aliveChannel += n
			}
		}
	}()
	go func() {
		for _ = range time.Tick(time.Second * 5) {
			log.Println("[G] alive conns:", aliveConn, "channels:", aliveChannel, "goroutine:", runtime.NumGoroutine())
		}
	}()

	conf, err := loadConfig()
	if err != nil {
		log.Println(err)
		return
	}
	runClient(conf)
}
