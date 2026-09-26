# 桌面程式用 cgo(Fyne + OpenGL),只能在對應平台上編譯,這裡編本機版本。

.PHONY: help
help:
	@echo "make build   編譯 -> bin/desktop"
	@echo "make run     直接執行(前景,關掉視窗才結束)"
	@echo "make protoc  由 proto/ 產生 pb/"
	@echo "make clean   刪除 bin/"

.PHONY: build
build:
	go build -o bin/desktop .

.PHONY: run
run: build
	./bin/desktop desktop

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
