package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gopkg.in/gin-contrib/cors.v1"
)

// StartAPIServer launches the API server
func StartAPIServer() error {
	if Config.LogLevel == 0 {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(cors.Default())

	router.GET("/blockcache", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"length": BlockCache.Length(), "items": BlockCache.Backend})
	})

	router.GET("/blockcache/exists/:key", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"exists": BlockCache.Exists(c.Param("key"))})
	})

	router.GET("/blockcache/get/:key", func(c *gin.Context) {
		if ok, _ := BlockCache.Get(c.Param("key")); !ok {
			c.IndentedJSON(http.StatusOK, gin.H{"error": c.Param("key") + " not found"})
		} else {
			c.IndentedJSON(http.StatusOK, gin.H{"success": ok})
		}
	})

	router.GET("/blockcache/length", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"length": BlockCache.Length()})
	})

	router.GET("/blockcache/remove/:key", func(c *gin.Context) {
		// Removes from BlockCache only. If the domain has already been queried and placed into MemoryCache, will need to wait until item is expired.
		BlockCache.Remove(c.Param("key"))
		c.IndentedJSON(http.StatusOK, gin.H{"success": true})
	})

	router.GET("/blockcache/set/:key", func(c *gin.Context) {
		// MemoryBlockCache Set() always returns nil, so ignoring response.
		_ = BlockCache.Set(c.Param("key"), true)
		c.IndentedJSON(http.StatusOK, gin.H{"success": true})
	})

	router.GET("/questioncache", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"length": QuestionCache.Length(), "items": QuestionCache.Backend})
	})

	router.GET("/questioncache/length", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"length": QuestionCache.Length()})
	})

	router.GET("/questioncache/clear", func(c *gin.Context) {
		QuestionCache.Clear()
		c.IndentedJSON(http.StatusOK, gin.H{"success": true})
	})

	router.GET("/questioncache/client/:client", func(c *gin.Context) {
		var filteredCache []QuestionCacheEntry

		QuestionCache.mu.RLock()
		for _, entry := range QuestionCache.Backend {
			if entry.Remote == c.Param("client") {
				filteredCache = append(filteredCache, entry)
			}
		}
		QuestionCache.mu.RUnlock()

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
				timeout_string := c.DefaultQuery("timeout", "300")
				timeout, err := strconv.ParseUint(timeout_string, 0, 0)
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
		PerformUpdate(true)
		c.AbortWithStatus(http.StatusOK)
	})

	if err := router.Run(Config.API); err != nil {
		return err
	}

	logger.Critical("API server listening on", Config.API)

	return nil
}
