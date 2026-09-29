#!/bin/sh
# docker compose 的 provider type 只能指向單一個執行檔,不能帶參數,所以用這支腳本轉接:
# docker compose 呼叫 `<type> compose ...`,這裡把它轉成 `bin/main desktop compose ...`
# (compose 命令掛在 desktop 底下,見 cmd/desktop/main.go)。
#
# up 時預設加 --launchd:交給 macOS launchd 執行,結束(崩潰或關掉視窗)就自動重啟。
# LANDAN_SUPERVISOR=go 改用 Go 版 supervisor:背景執行,結束(崩潰或關掉視窗)就自動重啟。
# --launchd / --supervisor 只有 up 認得,metadata/down 沒有這兩個旗標,所以不能對所有呼叫都加。
# down 兩種模式都會停,不用區分。
case " $* " in
	*" up "*)
		if [ "$LANDAN_SUPERVISOR" = "go" ]; then
			exec "$(dirname "$0")/../bin/main" desktop "$@" --supervisor
		fi
		exec "$(dirname "$0")/../bin/main" desktop "$@" --launchd ;;
	*) exec "$(dirname "$0")/../bin/main" desktop "$@" ;;
esac
