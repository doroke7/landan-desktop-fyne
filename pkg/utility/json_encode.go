package pkgUtility

import "encoding/json"

func JsonEncode[T any](oVal T) (string, error) {
	oByteVal, oErr := json.Marshal(oVal)
	sVal := string(oByteVal)
	return sVal, oErr
}
