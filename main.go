package main

import (
	"flag"
	"github.com/elico/drbl-peer"
	"log"
	"os"
	"os/signal"
	"time"
	"strconv"
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

var drblPeers *drblpeer.DrblPeers

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
	if Config.LogDomainsToRedis > 0 {
		redisConn.Addr = Config.RedisAddress + ":" + Config.RedisPort
		redisConn.Db = Config.RedisDB
		log.Println("REDIS domain loging is ON at =>", Config.RedisAddress+":"+Config.RedisPort+"/"+strconv.Itoa(Config.RedisDB))
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
	quit_activation := make(chan bool)
	go grimdActivation.loop(quit_activation)

	server := &Server{
		host:     Config.Bind,
		rTimeout: 5 * time.Second,
		wTimeout: 5 * time.Second,
	}

	server.Run()

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
}
