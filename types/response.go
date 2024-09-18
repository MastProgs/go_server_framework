package types

import (
	"encoding/json"
	"net/http"
)

type Result struct {
	Success     bool   `json:"success"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

type Response struct {
	Result Result      `json:"result"`
	Data   interface{} `json:"data"`
}

func CreateSuccessResponse(data interface{}) Response {
	return Response{
		Result: Result{
			Success:     true,
			ErrorCode:   0,
			Description: "success",
		},
		Data: data,
	}
}

func CreateErrorResponse(errorCode int, description string) Response {
	return Response{
		Result: Result{
			Success:     false,
			ErrorCode:   errorCode,
			Description: description,
		},
		Data: nil,
	}
}

func SendJSONResponse(w http.ResponseWriter, response Response) {
	status := getHTTPStatus(response.Result)

	// Content-Type 헤더 설정
	w.Header().Set("Content-Type", "application/json")

	// 응답 인코딩
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		// 인코딩 실패 시 에러 응답
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode response"})
		return
	}

	// 상태 코드 설정 (한 번만 호출)
	w.WriteHeader(status)

	// JSON 응답 쓰기
	w.Write(jsonResponse)
}

func getHTTPStatus(result Result) int {
	if result.Success {
		return http.StatusOK
	}
	// 에러 코드에 따른 HTTP 상태 코드 매핑
	// 여기에 필요한 매핑을 추가할 수 있습니다.
	return http.StatusInternalServerError
}
