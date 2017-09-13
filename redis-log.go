package main

import (
	"github.com/elico/goredis"
	"log"
)

var redisConn goredis.Client

func logDomainIntoRedis(domain string) {
	res, err := redisConn.Incr(domain)
	if err != nil {
		log.Println(err)
    return
	}
  if Config.LogLevel > 0 {
    log.Println("Logged to redis server =>", domain, "Result =>", res)
  }
}
