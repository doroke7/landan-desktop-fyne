package pkgUtility

import "encoding/json"

func JsonDecode[T any](sString string) (T, error) {
	var tResult T
	// 将字符串转为字节流并反序列化
	oErr := json.Unmarshal([]byte(sString), &tResult)
	return tResult, oErr
}
