package types

type Result struct {
	Success     bool   `json:"success"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

type Response struct {
	Result Result      `json:"result"`
	Data   interface{} `json:"data"`
}
