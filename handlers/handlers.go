package handlers

import (
	"time"
)

func SlowHandler() (interface{}, error) {
	time.Sleep(2 * time.Second)
	data := map[string]string{"msg": "느린 작업 완료!"}
	return data, nil
}

func FastHandler() (interface{}, error) {
	data := map[string]string{"msg": "빠른 응답!"}
	return data, nil
}
