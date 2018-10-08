package main

import (
	"context"
	"flag"
	"net/http"
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

	flag.Parse()

	config, err := LoadConfig(configPath)
	if err != nil {
		logger.Fatal(err)
	}

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

	grimdActive = true
	quitActivation := make(chan bool)
	go grimdActivation.loop(quitActivation, config.ReactivationDelay)

	server := &Server{
		host:     config.Bind,
		rTimeout: 5 * time.Second,
		wTimeout: 5 * time.Second,
	}

	// BlockCache contains all blocked domains
	blockCache := &MemoryBlockCache{Backend: make(map[string]bool)}
	// QuestionCache contains all queries to the dns server
	questionCache := &MemoryQuestionCache{Backend: make([]QuestionCacheEntry, 0), Maxcount: 1000}
	questionCache.Maxcount = config.QuestionCacheCap

	reloadChan := make(chan bool)

	var apiServer *http.Server
	go func() {
		run := <-startUpdate
		if !run {
			panic("The DNS server did not start in 10 seconds")
		}
		reloadChan <- true
	}()

	// The server will start with an empty blockcache and then trigger an update
	// via the `startUpdate` channel.
	server.Run(startUpdate, config, blockCache, questionCache)

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

forever:
	for {
		select {
		case <-sig:
			logger.Error("signal received, stopping\n")
			quitActivation <- true
			break forever
		case <-reloadChan:
			logger.Debug("Reloading the blockcache")
			blockCache = PerformUpdate(config, true)
			server.Stop()
			if apiServer != nil {
				apiServer.Shutdown(context.Background())
			}
			server.Run(startUpdate, config, blockCache, questionCache)
			apiServer, err = StartAPIServer(config, reloadChan, blockCache, questionCache)
			if err != nil {
				logger.Fatal(err)
			}
		}
	}
}

func init() {
	flag.StringVar(&configPath, "config", "grimd.toml", "location of the config file, if not found it will be generated (default grimd.toml)")
	flag.BoolVar(&forceUpdate, "update", false, "force an update of the blocklist database")

	runtime.GOMAXPROCS(runtime.NumCPU())
}
