#!/bin/sh
# docker compose 的 provider type 只能指向單一個執行檔,不能帶參數,所以用這支腳本轉接:
# docker compose 呼叫 `<type> compose ...`,這裡把它轉成 `bin/main desktop compose ...`
# (compose 命令掛在 desktop 底下,見 cmd/desktop/main.go)。
exec "$(dirname "$0")/../bin/main" desktop "$@"
