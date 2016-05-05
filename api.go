package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// StartAPIServer launches the API server
func StartAPIServer() error {
	if Config.LogLevel == 0 {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Next()
	})

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

	if err := router.Run(Config.API); err != nil {
		return err
	}

	log.Println("API server listening on", Config.API)

	return nil
}
