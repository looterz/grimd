package main

import (
	"net"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gopkg.in/gin-contrib/cors.v1"
)

// StartAPIServer starts the API server
func StartAPIServer(config *Config,
	reloadChan chan bool,
	blockCache *MemoryBlockCache,
	questionCache *MemoryQuestionCache) (*http.Server, error) {
	if !config.APIDebug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	server := &http.Server{
		Addr:    config.API,
		Handler: router,
	}

	router.Use(cors.Default())

	router.GET("/blockcache", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"length": blockCache.Length(), "items": blockCache.Backend})
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

	router.GET("/blockcache/set/:key", func(c *gin.Context) {
		// MemoryBlockCache Set() always returns nil, so ignoring response.
		_ = blockCache.Set(c.Param("key"), true)
		c.IndentedJSON(http.StatusOK, gin.H{"success": true})
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
