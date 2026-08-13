package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/coocood/freecache"
	"github.com/gin-gonic/gin"
)

// Cache instance with 256MB limit
var cache = freecache.NewCache(256 * 1024 * 1024)

func main() {
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// API info endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":    "HASkinProxy API",
			"version": "1.0.0",
		})
	})

	// Cache set endpoint
	r.POST("/cache/:key", func(c *gin.Context) {
		key := c.Param("key")
		var value map[string]interface{}
		if err := c.ShouldBindJSON(&value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ttl := 3600 // default 1 hour
		if t, ok := c.GetQuery("ttl"); ok {
			fmt.Sscanf(t, "%d", &ttl)
		}
		data, _ := json.Marshal(value)
		cache.Set([]byte(key), data, ttl)
		c.JSON(http.StatusOK, gin.H{"message": "cached", "key": key, "ttl": ttl})
	})

	// Cache get endpoint
	r.GET("/cache/:key", func(c *gin.Context) {
		key := c.Param("key")
		val, err := cache.Get([]byte(key))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
			return
		}
		var value map[string]interface{}
		json.Unmarshal(val, &value)
		c.JSON(http.StatusOK, gin.H{"key": key, "value": value})
	})

	// Cache delete endpoint
	r.DELETE("/cache/:key", func(c *gin.Context) {
		key := c.Param("key")
		cache.Del([]byte(key))
		c.JSON(http.StatusOK, gin.H{"message": "deleted", "key": key})
	})

	// Cache clear endpoint
	r.DELETE("/cache", func(c *gin.Context) {
		cache.Clear()
		c.JSON(http.StatusOK, gin.H{"message": "cache cleared"})
	})

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
