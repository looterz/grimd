package main

import (
	"bufio"
	"embed"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/gin-contrib/cors.v1"
)

//go:embed dashboard/reaper
var dashboardAssets embed.FS

func isRunningInDockerContainer() bool {
	// slightly modified from blog: https://paulbradley.org/indocker/
	// docker creates a .dockerenv file at the root
	// of the directory tree inside the container.
	// if this file exists then the viewer is running
	// from inside a container so return true

	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	return false
}

// StartAPIServer starts the API server
func StartAPIServer(config *Config,
	reloadChan chan bool,
	blockCache *MemoryBlockCache,
	questionCache *MemoryQuestionCache) (*http.Server, error) {
	if !config.APIDebug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Automatically replace the default listening address in the docker with `0.0.0.0`
	if isRunningInDockerContainer() {
		const localhost = "127.0.0.1:"
		if strings.HasPrefix(config.API, localhost) {
			config.API = strings.Replace(config.API, localhost, "0.0.0.0:", 1)
		}
	}

	server := &http.Server{
		Addr:    config.API,
		Handler: router,
	}

	router.Use(cors.Default())

	// Serves only if the user configuration enables the dashboard
	if config.Dashboard {
		router.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusTemporaryRedirect, "/dashboard")
			c.Abort()
		})

		dashboardAssets, _ := fs.Sub(dashboardAssets, "dashboard/reaper")
		router.StaticFS("/dashboard", http.FS(dashboardAssets))
	}

	router.GET("/blockcache", func(c *gin.Context) {
		special := make([]string, 0, len(blockCache.Special))
		for k := range blockCache.Special {
			special = append(special, k)
		}
		c.IndentedJSON(http.StatusOK, gin.H{"length": blockCache.Length(), "items": blockCache.Backend, "special": special})
	})

	router.GET("/blockcache/exists/:key", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"exists": blockCache.Exists(c.Param("key"))})
	})

	router.GET("/blockcache/get/:key", func(c *gin.Context) {
		if ok, _ := blockCache.Get(c.Param("key")); !ok {
			c.IndentedJSON(http.StatusOK, gin.H{"error": c.Param("key") + " not found"})
		} else {
			c.IndentedJSON(http.StatusOK, gin.H{"success": ok})
		}
	})

	router.GET("/blockcache/length", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"length": blockCache.Length()})
	})

	router.GET("/blockcache/remove/:key", func(c *gin.Context) {
		// Removes from BlockCache only. If the domain has already been queried and placed into MemoryCache, will need to wait until item is expired.
		blockCache.Remove(c.Param("key"))
		c.IndentedJSON(http.StatusOK, gin.H{"success": true})
	})

	router.GET("/blockcache/personal", func(c *gin.Context) {
		filePath := filepath.FromSlash("sources/personal.list")
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_RDONLY, 0600)
		if err != nil {
			logger.Critical(err)
		}

		defer func() {
			if err := f.Close(); err != nil {
				logger.Criticalf("Error closing file: %s\n", err)
			}
		}()

		var personalBlockList []string

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			personalBlockList = append(personalBlockList, line)
		}
		c.IndentedJSON(http.StatusOK, gin.H{"personalBlockList": personalBlockList})
	})

	router.GET("/blockcache/set/:key", func(c *gin.Context) {
		filePath := filepath.FromSlash("sources/personal.list")
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			logger.Critical(err)
		}

		defer func() {
			if err := f.Close(); err != nil {
				logger.Criticalf("Error closing file: %s\n", err)
			}
		}()

		_, err = blockCache.Get(c.Param("key"))
		if err == (KeyNotFound{c.Param("key")}) {
			// MemoryBlockCache Set() always returns nil, so ignoring response.
			_ = blockCache.Set(c.Param("key"), true)
			c.IndentedJSON(http.StatusOK, gin.H{"success": true})

			// Add domain to user block list
			if _, err := f.WriteString(c.Param("key") + "\n"); err != nil {
				logger.Critical(err)
			}
		} else {
			//_ = blockCache.Set(c.Param("key"), false)
			blockCache.Remove(c.Param("key"))
			c.IndentedJSON(http.StatusOK, gin.H{"success": true})

			personalBlockList := ""
			// Remove domain from user block list
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Replace(line, "\n", "", 1) != c.Param("key") {
					personalBlockList = personalBlockList + "\n" + line
				}
			}
			if scanner.Err() != nil {
				logger.Critical("error while reading personal block list")
				return
			}
			err := f.Truncate(0)
			if err != nil {
				logger.Error(err)
			}
			_, err = f.Write([]byte(personalBlockList))
			if err != nil {
				logger.Error(err)
			}
		}
	})

	router.GET("/questioncache", func(c *gin.Context) {
		highWater, err := strconv.ParseInt(c.DefaultQuery("highWater", "-1"), 10, 64)
		if err != nil {
			highWater = -1
		}
		c.IndentedJSON(http.StatusOK, gin.H{
			"length": questionCache.Length(),
			"items":  questionCache.GetOlder(highWater),
		})
	})

	router.GET("/questioncache/length", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"length": questionCache.Length()})
	})

	router.GET("/questioncache/clear", func(c *gin.Context) {
		questionCache.Clear()
		c.IndentedJSON(http.StatusOK, gin.H{"success": true})
	})

	router.GET("/questioncache/client/:client", func(c *gin.Context) {
		var filteredCache []QuestionCacheEntry

		questionCache.mu.RLock()
		for _, entry := range questionCache.Backend {
			if entry.Remote == c.Param("client") {
				filteredCache = append(filteredCache, entry)
			}
		}
		questionCache.mu.RUnlock()

		c.IndentedJSON(http.StatusOK, filteredCache)
	})

	router.GET("/questioncache/client", func(c *gin.Context) {
		clientList := make(map[string]bool)
		questionCache.mu.RLock()
		for _, entry := range questionCache.Backend {
			if _, ok := clientList[entry.Remote]; !ok {
				clientList[entry.Remote] = true
			}
		}
		questionCache.mu.RUnlock()
		var clients []string
		for client := range clientList {
			clients = append(clients, client)
		}
		c.IndentedJSON(http.StatusOK, clients)
	})

	router.OPTIONS("/application/active", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusOK)
	})

	router.GET("/application/active", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"active": grimdActive})
	})

	// Handle the setting of active state.
	// Possible values for state:
	// On
	// Off
	// Snooze: off for `timeout` seconds; timeout defaults to 300
	router.PUT("/application/active", func(c *gin.Context) {
		active := c.Query("state")
		version := c.Query("v")
		if version != "1" {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "Illegal value for 'version'"})
		} else {
			switch active {
			case "On":
				grimdActivation.set(true)
				c.IndentedJSON(http.StatusOK, gin.H{"active": grimdActive})
			case "Off":
				grimdActivation.set(false)
				c.IndentedJSON(http.StatusOK, gin.H{"active": grimdActive})
			case "Snooze":
				timeoutString := c.DefaultQuery("timeout", "300")
				timeout, err := strconv.ParseUint(timeoutString, 0, 0)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Illegal value for 'timeout'"})
				} else {
					grimdActivation.toggleOff(uint(timeout))
					c.IndentedJSON(http.StatusOK, gin.H{
						"active":  grimdActive,
						"timeout": timeout,
					})
				}
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": "Illegal value for 'state'"})
			}
		}
	})

	router.POST("/blocklist/update", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusOK)
		// Send reload trigger to chan in background goroutine so does not hang
		go func(reloadChan chan bool) {
			reloadChan <- true
		}(reloadChan)
	})

	listener, err := net.Listen("tcp", config.API)
	if err != nil {
		return nil, err
	}
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			logger.Fatal(err)
		}
	}()

	logger.Criticalf("API server listening on %s", config.API)
	return server, err
}
