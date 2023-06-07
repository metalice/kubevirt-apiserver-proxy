package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/kubevirt-ui/kubevirt-proxy-data/handlers"
)

func main() {
	server := gin.Default()
	server.GET("/health", handlers.HealthHandler)
	server.GET("/apis/*path", handlers.RequestHandler)

	log.Println("listening for server 8080")

	err := server.RunTLS(":8080", "./cert/tls.crt", "./cert/tls.key") // listen and serve on 0.0.0.0:8080

	if err != nil {
		log.Println("Failed to start server: ", err.Error())
	}
}
