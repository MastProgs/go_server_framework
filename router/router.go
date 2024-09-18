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

	// API 그룹
	apiHandler := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiHandler))

	// 인증이 필요한 API 그룹
	authApiHandler := http.NewServeMux()
	apiHandler.Handle("/auth/", middleware.JWTAuthMiddleware(http.StripPrefix("/auth", authApiHandler)))

	// 인증이 필요한 API 엔드포인트
	// /auth/profile
	authApiHandler.Handle("/profile", middleware.WorkerPoolMiddleware(Pool, middleware.MethodHandler{
		Get:  handlers.GetProfileHandler,
		Post: handlers.UpdateProfileHandler,
		// 필요한 다른 메서드들...
	}))

	// 공개 API 엔드포인트
	// /api/slow
	apiHandler.Handle("/slow", middleware.WorkerPoolMiddleware(Pool, middleware.MethodHandler{
		Get: handlers.SlowHandler,
	}))
	apiHandler.Handle("/fast", middleware.WorkerPoolMiddleware(Pool, middleware.MethodHandler{
		Get: handlers.FastHandler,
	}))

	// 인증 관련 엔드포인트
	// /api/login
	apiHandler.HandleFunc("/login", handlers.LoginHandler)
	apiHandler.HandleFunc("/logout", handlers.LogoutHandler)

	return mux
}
