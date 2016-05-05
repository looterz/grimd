# grimd
[![Build Status](https://travis-ci.org/looterz/grimd.svg)](https://travis-ci.org/looterz/grimd)
[![Go Report Card](https://goreportcard.com/badge/github.com/looterz/grimd)](https://goreportcard.com/report/github.com/looterz/grimd)
[![Code Climate](https://codeclimate.com/github/looterz/grimd/badges/gpa.svg)](https://codeclimate.com/github/looterz/grimd)
[![GoDoc](https://godoc.org/github.com/looterz/grimd?status.svg)](http://godoc.org/github.com/looterz/grimd)

:zap: Fast dns proxy that can run anywhere, built to black-hole internet advertisements and malware servers.

Based on [kenshinx/godns](https://github.com/kenshinx/godns) and [miekg/dns](https://github.com/miekg/dns).

# Installation
```
go get github.com/looterz/grimd
```

You can also download one of the [releases](https://github.com/looterz/grimd/releases), detailed guides and resources can be found on the [wiki](https://github.com/looterz/grimd/wiki).

# Configuration
If ```grimd.toml``` is not found, it will be generated for you, below is the default configuration.
```toml
# list of sources to pull blocklists from, stores them in ./sources
sources = [
"http://mirror1.malwaredomains.com/files/justdomains",
"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
"http://sysctl.org/cameleon/hosts",
"https://zeustracker.abuse.ch/blocklist.php?download=domainblocklist",
"https://s3.amazonaws.com/lists.disconnect.me/simple_tracking.txt",
"https://s3.amazonaws.com/lists.disconnect.me/simple_ad.txt",
"http://hosts-file.net/ad_servers.txt",
"https://raw.githubusercontent.com/quidsup/notrack/master/trackers.txt"
]

# list of locations to recursively read blocklists from (warning, every file found is assumed to be a hosts-file or domain list)
sourcedirs = [
"sources"
]

# location of the log file
log = "grimd.log"

# what kind of information should be logged, 0 = errors and important operations, 1 = dns queries, 2 = debug
loglevel = 0

# address to bind to for the DNS server
bind = "0.0.0.0:53"

# address to bind to for the API server
api = "127.0.0.1:8080"

# ipv4 address to forward blocked queries to
nullroute = "0.0.0.0"

# ipv6 address to forward blocked queries to
nullroutev6 = "0:0:0:0:0:0:0:0"

# nameservers to forward queries to
nameservers = ["8.8.8.8:53", "8.8.4.4:53"]

# concurrency interval for lookups in miliseconds
interval = 200

# query timeout for dns lookups in seconds
timeout = 5

# cache entry lifespan in seconds
expire = 600

# cache capacity, 0 for infinite
maxcount = 0

# question cache capacity, 0 for infinite but not recommended (this is used for storing logs)
questioncachecap = 5000

# manual blocklist entries
blocklist = []

# manual whitelist entries
whitelist = [
	"getsentry.com",
	"www.getsentry.com"
]
```

# Building
Requires golang 1.6, you build grimd like any other golang application, for example to build for linux x64
```shell
env GOOS=linux GOARCH=amd64 go build -v github.com/looterz/grimd
```

# Web API
A restful json api is exposed by default on the local interface, allowing you to build web applications that visualize requests, blocks and the cache. [reaper](https://github.com/looterz/reaper) is the default grimd web frontend, and will likely get a makeover when the [Buoy](http://jessecha.se/work/buoy.html) framework releases later this year.

![reaper-example](http://i.imgur.com/UW1uvOC.png)

# Speed
Incoming requests spawn a goroutine and are served concurrently, and the block cache resides in-memory to allow for rapid lookups, allowing grimd to serve thousands of queries at once while maintaining a memory footprint of under 15mb for 100,000 blocked domains!

# Daemonize
You can find examples of different daemon scripts for grimd on the [wiki](https://github.com/looterz/grimd/wiki/Daemon-Scripts).
