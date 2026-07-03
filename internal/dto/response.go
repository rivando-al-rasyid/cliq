package dto

type Response struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Success bool   `json:"isSuccess"`
	Error   string `json:"error,omitempty"`
}

// NewSuccess builds a successful response with a payload.
func NewSuccess(message string, data any) Response {
	return Response{
		Message: message,
		Data:    data,
		Success: true,
	}
}

// NewSuccessNoData builds a successful response without a payload.
func NewSuccessNoData(message string) Response {
	return Response{
		Message: message,
		Success: true,
	}
}

// NewError builds a failed response extracting string from native error interface.
func NewError(message string, err error) Response {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	return Response{
		Message: message,
		Success: false,
		Error:   errStr,
	}
}
