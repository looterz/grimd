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
)

func main() {
	// BlockCache contains all blocked domains
	blockCache := &MemoryBlockCache{Backend: make(map[string]bool)}
	// QuestionCache contains all queries to the dns server
	questionCache := &MemoryQuestionCache{Backend: make([]QuestionCacheEntry, 0), Maxcount: 1000}

	flag.Parse()

	config, err := LoadConfig(configPath)
	if err != nil {
		logger.Fatal(err)
	}

	questionCache.Maxcount = config.QuestionCacheCap

	logFile, err := LoggerInit(config.LogLevel, config.Log)
	if err != nil {
		logger.Fatal(err)
	}
	defer logFile.Close()

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
		PerformUpdate(forceUpdate, config)
	}()

	grimdActive = true
	quitActivation := make(chan bool)
	go grimdActivation.loop(quitActivation, config.ReactivationDelay)

	server := &Server{
		host:     config.Bind,
		rTimeout: 5 * time.Second,
		wTimeout: 5 * time.Second,
	}

	server.Run(startUpdate, config, blockCache, questionCache)

	if err := StartAPIServer(config, blockCache, questionCache); err != nil {
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
