package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"time"
)

var (
	configPath      string
	forceUpdate     bool
	grimdActive     bool
	grimdActivation ActivationHandler

	// BlockCache contains all blocked domains
	BlockCache = &MemoryBlockCache{Backend: make(map[string]bool)}

	// QuestionCache contains all queries to the dns server
	QuestionCache = &MemoryQuestionCache{Backend: make([]QuestionCacheEntry, 0), Maxcount: 1000}
)

func main() {

	flag.Parse()

	if err := LoadConfig(configPath); err != nil {
		log.Fatal(err)
	}

	QuestionCache.Maxcount = Config.QuestionCacheCap

	logFile, err := LoggerInit(Config.Log)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	// delay updating the blocklists, cache until the server starts and can serve requests as the local resolver
	start_update := make(chan bool, 1)

	//abort if the server does not come up in 10 seconds
	timer := time.NewTimer(time.Second * 10)
	go func() {
		<-timer.C
		start_update <- false
	}()

	go func() {
		run := <-start_update
		if !run {
			panic("The DNS server did not start in 10 seconds")
		}
		PerformUpdate(forceUpdate)
	}()

	grimdActive = true
	quit_activation := make(chan bool)
	go grimdActivation.loop(quit_activation)

	server := &Server{
		host:     Config.Bind,
		rTimeout: 5 * time.Second,
		wTimeout: 5 * time.Second,
	}

	server.Run(start_update)

	if err := StartAPIServer(); err != nil {
		log.Fatal(err)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

forever:
	for {
		select {
		case <-sig:
			log.Printf("signal received, stopping\n")
			quit_activation <- true
			break forever
		}
	}
}

func init() {
	flag.StringVar(&configPath, "config", "grimd.toml", "location of the config file, if not found it will be generated (default grimd.toml)")
	flag.BoolVar(&forceUpdate, "update", false, "force an update of the blocklist database")

	runtime.GOMAXPROCS(runtime.NumCPU())
}
