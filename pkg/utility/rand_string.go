package pkgUtility

import (
	"crypto/rand"
	"encoding/base64"
)

func RandString(iN int) string {

	// 1. 使用 16長度的 byte 資料
	oByteValue := make([]byte, iN)
	rand.Read(oByteValue)

	// 2. 直接編碼
	sResult := base64.RawURLEncoding.EncodeToString(oByteValue)

	// 3. 回傳前 n 位
	return sResult[:iN]
}

/*
// ASCII 跟 byte 問題
一個 Byte 的範圍是 $0$ 到 $255$。
ASCII 的限制：只有 $32$ 到 $126$ 是人類可讀的字元（字母、數字、標點）。
控制字元：$0$ 到 $31$ 是不可見的（例如「換行」、「響鈴」或「結束符號」）。
亂碼問題：如果隨機抽到 $130$，它在標準 ASCII 中根本沒有對應字元，在螢幕上會顯示成亂碼或問號。

*/

/**
為什麼 Base64 長度會變多？

也就是任意 N 長度的 ASCII string ，他的電腦底層都可以表示成 N 個 byte 的連接
（其中 1byte = 8 個bits）
然後他把 6 個bit 一組 切分（而剛好 6bit 2^6 等於 64） ，
所以我們就可以用 base64 表示剛才 的 N 個byte 連接，
然而 6 bit 是 比 8bit 小的，所以從新劃分後，字串會變長。

*/
