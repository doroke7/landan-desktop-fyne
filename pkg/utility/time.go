package pkgUtility

import (
	"strconv"
	"time"
)

func Time[T interface{ ~string | ~int | ~int64 }](bState bool) T {

	now := time.Now().Unix()
	if bState {
		now = time.Now().UnixMilli()
	}
	var result any

	// 根據泛型 T 的種類來決定邏輯
	// 這裡使用實例化一個零值來判斷型別
	var t T
	switch any(t).(type) {
	case string:
		result = strconv.FormatInt(now, 10)
	case int:
		result = int(now)
	case int64:
		result = now
	}

	return result.(T)
}
