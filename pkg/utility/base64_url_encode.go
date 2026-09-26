package pkgUtility

import "encoding/base64"

func Base64UrlEncode(input string) string {
	return base64.URLEncoding.EncodeToString([]byte(input))
}
