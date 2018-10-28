package main

import (
	"flag"
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
		logger.Fatal(err)
	}

	QuestionCache.Maxcount = Config.QuestionCacheCap

	logFiles, err := LoggerInit(Config.LogConfig)
	if err != nil {
		logger.Fatal(err)
	}
	defer func() {
		for _, f := range logFiles {
			f.Close()
		}
	}()

	// delay updating the blocklists, cache until the server starts and can serve requests as the local resolver
	startUpdate := make(chan bool, 1)

	//abort if the server does not come up in 10 seconds
	timer := time.NewTimer(time.Second * 10)
	go func() {
		<-timer.C
		startUpdate <- false
	}()

	go func() {
		run := <-startUpdate
		if !run {
			panic("The DNS server did not start in 10 seconds")
		}
		PerformUpdate(forceUpdate)
	}()

	grimdActive = true
	quitActivation := make(chan bool)
	go grimdActivation.loop(quitActivation)

	server := &Server{
		host:     Config.Bind,
		rTimeout: 5 * time.Second,
		wTimeout: 5 * time.Second,
	}

	server.Run(startUpdate)

	if err := StartAPIServer(); err != nil {
		logger.Fatal(err)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

forever:
	for {
		select {
		case <-sig:
			logger.Error("signal received, stopping\n")
			quitActivation <- true
			break forever
		}
	}
}

func init() {
	flag.StringVar(&configPath, "config", "grimd.toml", "location of the config file, if not found it will be generated (default grimd.toml)")
	flag.BoolVar(&forceUpdate, "update", false, "force an update of the blocklist database")

	runtime.GOMAXPROCS(runtime.NumCPU())
}
