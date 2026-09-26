package pkgUtility

import "encoding/base64"

func Base64Decode(encodedStr string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedStr)
	if err != nil {
		return "", err // 解碼失敗，回傳錯誤
	}
	return string(decodedBytes), nil // 將 []byte 轉回字串成功回傳
}
