package dto

type Response struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
	Success bool   `json:"isSuccess"`
	Error   string `json:"error"`
}

// NewSuccess builds a successful response with a payload.
func NewSuccess(message string, data any) Response {
	return Response{
		Message: message,
		Data:    data,
		Success: true,
		Error:   "",
	}
}

// NewSuccessNoData builds a successful response without a payload.
func NewSuccessNoData(message string) Response {
	return NewSuccess(message, nil)
}

// NewError builds a failed response.
func NewError(message, errDetail string) Response {
	return Response{
		Message: message,
		Data:    nil,
		Success: false,
		Error:   errDetail,
	}
}
