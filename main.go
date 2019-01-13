package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"time"
	"github.com/elico/drbl-peer"
)

var (
	configPath      string
	forceUpdate     bool
	grimdActive     bool
	grimdActivation activationHandler

	// BlockCache contains all blocked domains
	BlockCache = &MemoryBlockCache{Backend: make(map[string]bool)}

	// QuestionCache contains all queries to the dns server
	QuestionCache = &MemoryQuestionCache{Backend: make([]QuestionCacheEntry, 0), Maxcount: 1000}
	drblPeers *drblpeer.DrblPeers
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

        if Config.UseDrbl > 0 {
                drblPeers, _ = drblpeer.NewPeerListFromYamlFile(Config.DrblPeersFilename, Config.DrblBlockWeight, Config.DrblTimeout, (Config.LogLevel > 0))
                if Config.DrblDebug > 0 {
                        log.Println("Drbl Debug is ON")
                        drblPeers.Debug = true
                        log.Println("Drbl Debug is ON", drblPeers.Debug)
                }
        }

	// delay updating the blocklists, cache until the server starts and can serve requests as the local resolver
	timer := time.NewTimer(time.Second * 1)
	go func() {
		<-timer.C
		if _, err := os.Stat("lists"); os.IsNotExist(err) || forceUpdate {
			if err := Update(); err != nil {
				log.Fatal(err)
			}
		}
		if err := UpdateBlockCache(); err != nil {
			log.Fatal(err)
		}
	}()

	grimdActive = true
	quitActivation := make(chan bool)
	go grimdActivation.loop(quitActivation)

	server := &Server{
		host:     Config.Bind,
		rTimeout: 5 * time.Second,
		wTimeout: 5 * time.Second,
	}

	server.Run()

	if Config.WebPanel {
		if err := StartWebServer(); err != nil {
			log.Fatal(err)
		}

		log.Printf("API server listening on http://%s/api\nFrontend server located at http://%s/frontend\n", Config.API, Config.API)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

forever:
	for {
		select {
		case <-sig:
			log.Printf("signal received, stopping\n")
			quitActivation <- true
			break forever
		}
	}
}

func init() {
	flag.StringVar(&configPath, "config", "grimd.toml", "location of the config file, if not found it will be generated (default grimd.toml)")
	flag.BoolVar(&forceUpdate, "update", false, "force an update of the blocklist database")

	rand.Seed(time.Now().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())
}
