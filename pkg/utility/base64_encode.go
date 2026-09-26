package pkgUtility

import "encoding/base64"

func Base64Encode(input string) string {
	return base64.StdEncoding.EncodeToString([]byte(input))
}
