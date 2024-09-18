package middleware

import (
	"net/http"
	"sync"

	workerpool "go_server_framework/core"
	"go_server_framework/types"
)

func WorkerPoolMiddleware(pool *workerpool.WorkerPool, handler func() (interface{}, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var wg sync.WaitGroup
		wg.Add(1)

		pool.Submit(func() {
			defer wg.Done()
			data, err := handler()
			if err != nil {
				response := types.CreateErrorResponse(500, err.Error())
				types.SendJSONResponse(w, response)
				return
			}
			response := types.CreateSuccessResponse(data)
			types.SendJSONResponse(w, response)
		})

		wg.Wait()
	}
}
