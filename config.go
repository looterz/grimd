package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// BuildVersion returns the build version of grimd, this should be incremented every new release
var BuildVersion = "1.0.5"

// ConfigVersion returns the version of grimd, this should be incremented every time the config changes so grimd presents a warning
var ConfigVersion = "1.0.2"

type config struct {
	Version           string
	Sources           []string
	SourceDirs        []string
	Log               string
	LogLevel          int
	Bind              string
	API               string
	Nullroute         string
	Nullroutev6       string
	Nameservers       []string
	Interval          int
	Timeout           int
	Expire            int
	Maxcount          int
	QuestionCacheCap  int
	TTL               uint32
	Blocklist         []string
	Whitelist         []string
	ToggleName        string
	ReactivationDelay uint
}

var defaultConfig = `# version this config was generated from
version = "%s"

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

# When this string is queried, toggle grimd on and off
togglename = ""

# If not zero, the delay in seconds before grimd automaticall reactivates after
# having been turned off.
reactivationdelay = 300
`

// Config is the global configuration
var Config config

// LoadConfig loads the given config file
func LoadConfig(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := generateConfig(path); err != nil {
			return err
		}
	}

	if _, err := toml.DecodeFile(path, &Config); err != nil {
		return fmt.Errorf("could not load config: %s", err)
	}

	if Config.Version != ConfigVersion {
		if Config.Version == "" {
			Config.Version = "none"
		}

		log.Printf("warning, grimd.toml is out of date!\nconfig v%s\ngrimd config v%s\ngrimd v%s\nplease update your config\n", Config.Version, ConfigVersion, BuildVersion)
	} else {
		log.Printf("grimd v%s\n", BuildVersion)
	}

	return nil
}

func generateConfig(path string) error {
	output, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not generate config: %s", err)
	}
	defer output.Close()

	r := strings.NewReader(fmt.Sprintf(defaultConfig, ConfigVersion))
	if _, err := io.Copy(output, r); err != nil {
		return fmt.Errorf("could not copy default config: %s", err)
	}

	if abs, err := filepath.Abs(path); err == nil {
		log.Printf("generated default config %s\n", abs)
	}

	return nil
}
