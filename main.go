package main

import (
	"log"
	"time"

	cache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-gonic/gin"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/handlers"
)

func main() {
	server := gin.Default()
	memoryStore := persist.NewMemoryStore(1 * time.Minute)
	server.GET("/health", cache.CacheByRequestURI(memoryStore, 5*time.Second), handlers.HealthHandler)
	server.GET("/apis/*path", cache.CacheByRequestURI(memoryStore, 2*time.Second), handlers.RequestHandler)
	log.Println("listening for server 8080")

	err := server.RunTLS(":8080", "./cert/tls.crt", "./cert/tls.key") // listen and serve on 0.0.0.0:8080

	if err != nil {
		log.Println("Failed to start server: ", err.Error())
	}
}
