package router

import (
	"net/http"

	workerpool "go_server_framework/core"
	"go_server_framework/handlers"
	"go_server_framework/middleware"
)

var Pool *workerpool.WorkerPool

func SetupRouter() *http.ServeMux {
	if Pool == nil {
		Pool = workerpool.NewWorkerPool()
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/slow", middleware.WorkerPoolMiddleware(Pool, handlers.SlowHandler))
	mux.HandleFunc("/fast", middleware.WorkerPoolMiddleware(Pool, handlers.FastHandler))

	return mux
}
