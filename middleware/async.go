package middleware

import (
	"net/http"

	workerpool "go_server_framework/core"
	"go_server_framework/types"
)

type HandlerFunc func(*http.Request) (interface{}, error)

type MethodHandler struct {
	Get    HandlerFunc
	Post   HandlerFunc
	Put    HandlerFunc
	Delete HandlerFunc
}

func WorkerPoolMiddleware(pool *workerpool.WorkerPool, handlers MethodHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var handler HandlerFunc

		switch r.Method {
		case http.MethodGet:
			handler = handlers.Get
		case http.MethodPost:
			handler = handlers.Post
		case http.MethodPut:
			handler = handlers.Put
		case http.MethodDelete:
			handler = handlers.Delete
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if handler == nil {
			http.Error(w, "Method not implemented", http.StatusNotImplemented)
			return
		}

		// 채널을 생성하여 작업 완료를 기다립니다.
		done := make(chan struct{})

		pool.Submit(func() {
			defer close(done)
			data, err := handler(r)
			if err != nil {
				response := types.CreateErrorResponse(500, err.Error())
				types.SendJSONResponse(w, response)
				return
			}
			response := types.CreateSuccessResponse(data)
			types.SendJSONResponse(w, response)
		})

		// 작업이 완료될 때까지 기다립니다.
		<-done
	}
}
