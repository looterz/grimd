# grimd
[![Go Report Card](https://goreportcard.com/badge/github.com/looterz/grimd?style=flat-square)](https://goreportcard.com/report/github.com/looterz/grimd)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/looterz/grimd)
[![Release](https://github.com/looterz/grimd/actions/workflows/release.yaml/badge.svg)](https://github.com/looterz/grimd/releases)

:zap: Fast dns proxy that can run anywhere, built to black-hole internet advertisements and malware servers.

Based on [kenshinx/godns](https://github.com/kenshinx/godns) and [miekg/dns](https://github.com/miekg/dns).

# Installation
```
go install github.com/looterz/grimd@latest
```

You can also download one of the [releases](https://github.com/looterz/grimd/releases) or [docker images](https://github.com/looterz/grimd/pkgs/container/grimd). Detailed guides and resources can be found on the [wiki](https://github.com/looterz/grimd/wiki).

# Docker Installation
To quickly get grimd up and running with docker, run
```
docker run -d -p 53:53/udp -p 53:53/tcp -p 8080:8080/tcp ghcr.io/looterz/grimd:latest
```

Alternatively, download the [docker-compose.yml](https://raw.githubusercontent.com/looterz/grimd/master/docker-compose.yml) file and launch it using docker-compose.
```
docker-compose up -d
```

# Configuration
If ```grimd.toml``` is not found, it will be generated for you, below is the default configuration.
```toml
# version this config was generated from
version = "1.0.9"

# list of sources to pull blocklists from, stores them in ./sources
sources = [
"https://mirror1.malwaredomains.com/files/justdomains",
"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
"https://sysctl.org/cameleon/hosts",
"https://s3.amazonaws.com/lists.disconnect.me/simple_tracking.txt",
"https://s3.amazonaws.com/lists.disconnect.me/simple_ad.txt",
"https://gitlab.com/quidsup/notrack-blocklists/raw/master/notrack-blocklist.txt"
]

# list of locations to recursively read blocklists from (warning, every file found is assumed to be a hosts-file or domain list)
sourcedirs = [
"sources"
]

# log configuration
# format: comma separated list of options, where options is one of 
#   file:<filename>@<loglevel>
#   stderr>@<loglevel>
#   syslog@<loglevel>
# loglevel: 0 = errors and important operations, 1 = dns queries, 2 = debug
# e.g. logconfig = "file:grimd.log@2,syslog@1,stderr@2"
logconfig = "file:grimd.log@2,stderr@2"

# apidebug enables the debug mode of the http api library
apidebug = false

# enable the web interface by default
dashboard = true

# address to bind to for the DNS server
bind = "0.0.0.0:53"

# address to bind to for the API server
api = "127.0.0.1:8080"

# response to blocked queries with a NXDOMAIN
nxdomain = false

# ipv4 address to forward blocked queries to
nullroute = "0.0.0.0"

# ipv6 address to forward blocked queries to
nullroutev6 = "0:0:0:0:0:0:0:0"

# nameservers to forward queries to
nameservers = ["1.1.1.1:53", "1.0.0.1:53"]

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

# Drbl related settings
usedrbl = 0
drblpeersfilename = "drblpeers.yaml"
drblblockweight = 128
drbltimeout = 30
drbldebug = 0

# manual whitelist entries
whitelist = [
	"getsentry.com",
	"www.getsentry.com"
]

# manual custom dns entries
customdnsrecords = []

# When this string is queried, toggle grimd on and off
togglename = ""

# If not zero, the delay in seconds before grimd automaticall reactivates after
# having been turned off.
reactivationdelay = 300

#Dns over HTTPS provider to use.
DoH = "https://cloudflare-dns.com/dns-query"
```

# Building
Requires golang 1.7 or higher, you build grimd like any other golang application, for example to build for linux x64
```shell
env GOOS=linux GOARCH=amd64 go build -v github.com/looterz/grimd
```

# Building Docker
Run container and test
```shell
mkdir sources
docker build -t grimd:latest -f docker/Dockerfile . && \
docker run -v $PWD/sources:/sources --rm -it -P --name grimd-test grimd:latest --config /sources/grimd.toml --update
```

By default, if the program runs in a docker, it will automatically replace `127.0.0.1` in the default configuration with `0.0.0.0` to ensure that the API interface is available.

```shell
curl -H "Accept: application/json" http://127.0.0.1:55006/application/active
```

# Web API
A restful json api is exposed by default on the local interface, allowing you to build web applications that visualize requests, blocks and the cache. [reaper](https://github.com/looterz/reaper) is the default grimd web frontend.


If you want to enable the default dashboard, make sure the configuration file contains the following:

```toml
dashboard = true
```

![reaper-example](http://i.imgur.com/oXLtqSz.png)

# Speed
Incoming requests spawn a goroutine and are served concurrently, and the block cache resides in-memory to allow for rapid lookups, while answered queries are cached allowing grimd to serve thousands of queries at once while maintaining a memory footprint of under 15mb for 100,000 blocked domains!

# Daemonize
You can find examples of different daemon scripts for grimd on the [wiki](https://github.com/looterz/grimd/wiki/Daemon-Scripts).
