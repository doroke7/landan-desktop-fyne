package pkgUtility

import (
	"crypto/md5"
	"encoding/hex"
)

func Md5(sString string) string {

	oHash := md5.Sum([]byte(sString))

	sResult := hex.EncodeToString(oHash[:])

	return sResult
}
