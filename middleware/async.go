package middleware

import (
	"net/http"
	"sync"

	workerpool "go_server_framework/core"
)

func WorkerPoolMiddleware(pool *workerpool.WorkerPool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 응답 작성을 동기화하기 위한 WaitGroup 사용
		var wg sync.WaitGroup
		wg.Add(1)

		pool.Submit(func() {
			defer wg.Done()
			next.ServeHTTP(w, r)
		})

		// 작업이 완료될 때까지 대기
		wg.Wait()
	}
}
