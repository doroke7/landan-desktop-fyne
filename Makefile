# 桌面程式用 cgo(Fyne + OpenGL),只能在對應平台上編譯,這裡編本機版本。
# 由 docker compose 啟動宿主機上的程式(見 compose.yaml),不是容器。
# compose 只會呼叫 `<type> compose ...`,所以 type 指向轉接腳本 bin/desktop_compose_up.sh(見 compose.yaml),
# 由它轉成 `bin/main desktop compose ... --supervisor`:在背景啟動 supervisor,視窗結束(崩潰或關掉)就自動重啟。

.PHONY: help
help:
	@echo "make build   編譯 -> bin/main,並產生 bin/desktop_compose_up.sh"
	@echo "make run     直接執行(前景,關掉視窗才結束)"
	@echo "make up      編譯並用 docker compose 在背景啟動"
	@echo "make down    停止"
	@echo "make logs    看背景程式的日誌"
	@echo "make protoc  由 proto/ 產生 pb/"
	@echo "make clean   刪除 bin/"

.PHONY: build
build:
	go build -o bin/main .
	@printf '%s\n' '#!/bin/sh' 'exec "$$(dirname "$$0")/main" desktop "$$@" --supervisor' > bin/desktop_compose_up.sh
	@chmod +x bin/desktop_compose_up.sh

.PHONY: run
run: build
	./bin/main desktop

# 專案裡沒有容器時 docker compose up 一定回傳 1(程式其實已啟動),所以前面加 - 忽略結束碼;
# 真的失敗時,錯誤訊息仍會顯示在輸出裡。
# --progress=plain:預設的動畫進度畫面會吃掉 provider 回報的訊息。
.PHONY: up
up: build
	-docker compose --progress=plain up

.PHONY: down
down:
	docker compose --progress=plain down

.PHONY: logs
logs:
	tail -f runtime/desktop/desktop.log

.PHONY: protoc
protoc:
	@protoc \
	-I ./proto \
	--go_out=paths=source_relative:./pb \
	--go-grpc_out=paths=source_relative:./pb \
	$$(find ./proto/ir -name "*.proto")

.PHONY: clean
clean:
	rm -rf bin
