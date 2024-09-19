package handlers

import (
	"fmt"
	"io"
	"net/http"
)

func PingHandler(r *http.Request) (interface{}, error) {
	data := map[string]string{"msg": "pong"}
	return data, nil
}

func PostPingHandler(r *http.Request) (interface{}, error) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	defer r.Body.Close()

	// body 출력
	fmt.Printf("Received body: %s\n", string(body))

	data := map[string]string{"msg": "pong"}
	return data, nil
}

func GetProfileHandler(r *http.Request) (interface{}, error) {
	// GET 요청 처리 로직
	return map[string]string{"name": "John Doe", "email": "john@example.com"}, nil
}

func UpdateProfileHandler(r *http.Request) (interface{}, error) {

	// 여기에 body 처리 로직 추가
	return map[string]string{"msg": "Profile updated successfully"}, nil
}
