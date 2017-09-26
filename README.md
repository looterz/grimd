# grimd
[![Travis](https://img.shields.io/travis/looterz/grimd.svg?style=flat-square)](https://travis-ci.org/looterz/grimd)
[![Go Report Card](https://goreportcard.com/badge/github.com/looterz/grimd?style=flat-square)](https://goreportcard.com/report/github.com/looterz/grimd)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/looterz/grimd)

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
# version this config was generated from
version = "1.0.8"

# list of sources to pull blocklists from, stores them in ./sources

useblocklist = 0

sources = [
"http://mirror1.malwaredomains.com/files/justdomains",
"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
"http://sysctl.org/cameleon/hosts",
"https://zeustracker.abuse.ch/blocklist.php?download=domainblocklist",
"https://s3.amazonaws.com/lists.disconnect.me/simple_tracking.txt",
"https://s3.amazonaws.com/lists.disconnect.me/simple_ad.txt",
"http://hosts-file.net/ad_servers.txt",
"https://raw.githubusercontent.com/quidsup/notrack/master/trackers.txt",
"http://moodle.ngtech.co.il/ngtech-ads-hosts-only.txt"
]

# list of locations to recursively read blocklists from (warning, every file found is assumed to be a hosts-file or domain list)
sourcedirs = [
 "/var/grim/sources"
]

# location of the log file
log = "/var/log/grimd.log"

# what kind of information should be logged, 0 = errors and important operations, 1 = dns queries, 2 = debug
loglevel = 0

# address to bind to for the DNS server
bind = ":53"

# address to bind to for the API server
api = "127.0.0.1:28080"

# ipv4 address to forward blocked queries to
nullroute = "213.151.33.115"

# ipv6 address to forward blocked queries to
nullroutev6 = "2a01:6500:1:1000::114:100"

# nameservers to forward queries to
nameservers = ["8.8.8.8:53", "8.8.4.4:53"]

# concurrency interval for lookups in miliseconds
interval = 200

# query timeout for dns lookups in seconds
timeout = 5

# disable dns caching
disablecache = 1

# disable dns queries cache
disablequestioncache = 1

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
	"ngtech.co.il",
	"www.ngtech.co.il"
]

# When this string is queried, toggle grimd on and off
togglename = ""

# If not zero, the delay in seconds before grimd automaticall reactivates after
# having been turned off.
reactivationdelay = 300

# Drbl related settings
usedrbl = 1
drblpeersfilename = "/etc/drblpeers.yaml"
drblblockweight = 128
drbltimeout = 30
drbldebug = 0

# Log domains to redis DB
logdomainstoredis = 0
redisaddress = "127.0.0.1"
redisport = "6379"
redisdb = 0
```

# Building
Requires golang 1.7 or higher, you build grimd like any other golang application, for example to build for linux x64
```shell
env GOOS=linux GOARCH=amd64 go build -v github.com/looterz/grimd
```

# Web API
A restful json api is exposed by default on the local interface, allowing you to build web applications that visualize requests, blocks and the cache. [reaper](https://github.com/looterz/reaper) is the default grimd web frontend.

![reaper-example](http://i.imgur.com/oXLtqSz.png)

# Speed
Incoming requests spawn a goroutine and are served concurrently, and the block cache resides in-memory to allow for rapid lookups, while answered queries are cached allowing grimd to serve thousands of queries at once while maintaining a memory footprint of under 15mb for 100,000 blocked domains!

# Daemonize
You can find examples of different daemon scripts for grimd on the [wiki](https://github.com/looterz/grimd/wiki/Daemon-Scripts).
