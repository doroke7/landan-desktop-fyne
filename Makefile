# 桌面程式用 cgo(Fyne + OpenGL),只能在對應平台上編譯,這裡編本機版本。

.PHONY: help
help:
	@echo "make build   編譯 -> bin/main"
	@echo "make run     直接執行(前景,關掉視窗才結束)"
	@echo "make bundle  打包成 .app(macOS),icon 才會生效"
	@echo "make protoc  由 proto/ 產生 pb/"
	@echo "make clean   刪除 bin/ 與打包產物"

.PHONY: build
build:
	go build -o bin/main .

.PHONY: start
start: build
	./bin/main desktop

# 打包成 .app(macOS)/ .exe 安裝檔等,讓 icon 真正生效(單純 go build 的執行檔沒有
# app bundle,系統圖示、Dock icon 都不會套用)。用 go run 指定版本執行 fyne CLI,
# 不需要額外 go install。
.PHONY: bundle
bundle: build
	go run fyne.io/fyne/v2/cmd/fyne@v2.8.1 package \
		-name landan-desktop-fyne \
		-appID com.landan.desktop \
		-icon asset/icon/icon.jpeg \
		-exe bin/main

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
	rm -rf landan-desktop-fyne.app
