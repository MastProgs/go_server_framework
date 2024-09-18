package handlers

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func SlowHandler(r *http.Request) (interface{}, error) {
	time.Sleep(2 * time.Second)
	data := map[string]string{"msg": "느린 작업 완료!"}
	return data, nil
}

func FastHandler(r *http.Request) (interface{}, error) {
	data := map[string]string{"msg": "빠른 응답!"}
	return data, nil
}

func GetProfileHandler(r *http.Request) (interface{}, error) {
	// GET 요청 처리 로직
	return map[string]string{"name": "John Doe", "email": "john@example.com"}, nil
}

func UpdateProfileHandler(r *http.Request) (interface{}, error) {
	// POST 요청 처리 로직
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	defer r.Body.Close()

	// body 출력
	fmt.Printf("Received body: %s\n", string(body))

	// 여기에 body 처리 로직 추가

	return map[string]string{"message": "Profile updated successfully"}, nil
}
