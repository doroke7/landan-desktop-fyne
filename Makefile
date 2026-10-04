# 桌面程式用 cgo(Fyne + OpenGL),只能在對應平台上編譯,這裡編本機版本。

# 簽 .app 用的憑證名稱,由 make cert 建立。
SIGN_IDENTITY ?= landan-desktop-dev
APP := landan-desktop-fyne.app

.PHONY: help
help:
	@echo "make build   編譯 -> bin/main"
	@echo "make run     直接執行(前景,關掉視窗才結束)"
	@echo "make cert    建立簽章用的自簽憑證(只需一次)"
	@echo "make bundle  打包成 .app 並簽章(macOS),icon、攝影機授權才會生效"
	@echo "make protoc  由 proto/ 產生 pb/"
	@echo "make clean   刪除 bin/ 與打包產物"

.PHONY: build
build:
	go build -o bin/main .

.PHONY: start
start: build
	./bin/main desktop

.PHONY: cert
cert:
	SIGN_IDENTITY="$(SIGN_IDENTITY)" ./script/make-cert.sh

# 打包成 .app(macOS)/ .exe 安裝檔等,讓 icon 真正生效(單純 go build 的執行檔沒有
# app bundle,系統圖示、Dock icon 都不會套用)。用 go run 指定版本執行 fyne CLI,
# 不需要額外 go install。
#
# 打包後還要兩件事,攝影機才不會每次啟動都重新詢問:
#   1. Info.plist 要有 NSCameraUsageDescription,否則 macOS 可能直接拒絕而不跳詢問。
#   2. 用固定的憑證簽章。授權記在簽章身分上,ad-hoc 簽章每次編譯都不同,授權就作廢。
.PHONY: bundle
bundle: build
	go run fyne.io/fyne/v2/cmd/fyne@v2.8.1 package \
		-name landan-desktop-fyne \
		-appID com.landan.desktop \
		-icon asset/icon/main.png \
		-exe bin/main
	/usr/libexec/PlistBuddy -c "Add :NSCameraUsageDescription string 需要使用攝影機來顯示與處理影像" $(APP)/Contents/Info.plist
	@if security find-identity -p codesigning | grep -q '"$(SIGN_IDENTITY)"'; then \
		codesign --force --deep --sign "$(SIGN_IDENTITY)" $(APP) && echo "已用 $(SIGN_IDENTITY) 簽章"; \
	else \
		echo "警告: 找不到憑證 $(SIGN_IDENTITY),改用 ad-hoc 簽章,攝影機授權每次編譯都會失效。先執行 make cert"; \
		codesign --force --deep --sign - $(APP); \
	fi

.PHONY: protoc
protoc:
	@protoc \
	-I ./proto \
	--go_out=paths=source_relative:./pb \
	--go-grpc_out=paths=source_relative:./pb \
	$$(find ./proto/recognition -name "*.proto")

.PHONY: clean
clean:
	rm -rf bin
	rm -rf $(APP)
