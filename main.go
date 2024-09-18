package main

import (
	"fmt"
	"log"
	"net/http"

	"go_server_framework/config"
	"go_server_framework/router"
)

func main() {
	cfg := config.GetConfig()

	r := router.SetupRouter()

	// 워커 풀 시작
	router.Pool.Start()
	defer router.Pool.Stop()

	port := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("Server is running on %s port...\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
