package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"go_server_framework/types"
)

func SlowHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(2 * time.Second)
	response := types.Response{
		Result: types.Result{
			Success:     true,
			ErrorCode:   0,
			Description: "Slow operation completed successfully",
		},
		Data: "작업 완료!",
	}
	sendJSONResponse(w, response)
}

func FastHandler(w http.ResponseWriter, r *http.Request) {
	response := types.Response{
		Result: types.Result{
			Success:     true,
			ErrorCode:   0,
			Description: "Fast operation completed successfully",
		},
		Data: "빠른 응답!",
	}
	sendJSONResponse(w, response)
}

func sendJSONResponse(w http.ResponseWriter, response types.Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(getHTTPStatus(response))
	json.NewEncoder(w).Encode(response)
}

func getHTTPStatus(response types.Response) int {
	if !response.Result.Success {
		return response.Result.ErrorCode
	}
	return http.StatusOK
}
