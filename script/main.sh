#!/bin/sh
# docker compose 的 provider type 只能指向單一個執行檔,不能帶參數,所以用這支腳本轉接:
# docker compose 呼叫 `<type> compose ...`,這裡把它轉成 `bin/main desktop compose ...`
# (compose 命令掛在 desktop 底下,見 cmd/desktop/main.go)。
#
# up 時多加 --supervisor:背景執行,結束(崩潰或關掉視窗)就自動重啟。
# --supervisor 只有 up 認得,metadata/down 沒有這個旗標,所以不能對所有呼叫都加。
case " $* " in
	*" up "*) exec "$(dirname "$0")/../bin/main" desktop "$@" --supervisor ;;
	*) exec "$(dirname "$0")/../bin/main" desktop "$@" ;;
esac
