package utils

func IsSuccessResponse(statusCode int, exceptCode ...int) bool {
	if 200 <= statusCode && statusCode < 300 {
		return true
	}
	for _, code := range exceptCode {
		if statusCode == code {
			return true
		}
	}
	return false
}
