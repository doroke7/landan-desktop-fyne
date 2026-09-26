package pkgUtility

type DefaultStatus struct {
	Code    int16
	Status  int16
	Message string // 給前端或用戶看的訊息
}

func NewDefaultStatus(sMessage string, iCode int16, iStatus int16) *DefaultStatus {
	return &DefaultStatus{
		Code:    iCode,
		Status:  iStatus,
		Message: sMessage,
	}
}

func (oSelf *DefaultStatus) Error() string {
	return oSelf.Message
}
