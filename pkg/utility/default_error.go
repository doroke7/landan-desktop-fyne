package pkgUtility

type DefaultError struct {
	Code    int16
	Status  int16
	Message string // 給前端或用戶看的訊息
}

func NewDefaultError(sMessage string, iCode int16, iStatus int16) *DefaultError {
	return &DefaultError{
		Code:    iCode,
		Status:  iStatus,
		Message: sMessage,
	}
}

func (oSelf *DefaultError) Error() string {
	return oSelf.Message
}
