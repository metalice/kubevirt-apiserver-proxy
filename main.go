package main

import (
	"log"
	"time"

	cache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/handlers"
)

var HEALTH_CACHE_TIME = 30 * time.Second
var API_CACHE_TIME = 15 * time.Second

func main() {
	server := gin.Default()
	memoryStore := persist.NewMemoryStore(1 * time.Minute)

	server.Use(gzip.Gzip(gzip.DefaultCompression))

	server.GET("/health", cache.CacheByRequestURI(memoryStore, HEALTH_CACHE_TIME), handlers.HealthHandler)
	server.GET("/apis/*path", cache.CacheByRequestURI(memoryStore, API_CACHE_TIME), handlers.RequestHandler)

	log.Printf("listening for server 8080 - v0.0.8 - API cache time: %v", API_CACHE_TIME)

	err := server.RunTLS(":8080", "./cert/tls.crt", "./cert/tls.key") // listen and serve on 0.0.0.0:8080

	if err != nil {
		log.Println("Failed to start server: ", err.Error())
	}
}
