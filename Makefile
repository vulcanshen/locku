# locku — build / run / test / package
#
# 跑 `make`（或 `make help`）列出所有指令。
# unix-first（macOS + Linux），CGO_ENABLED=0 靜態編譯；跨平台 release 走 goreleaser。

BINARY   := locku
PKG      := ./cmd/locku
DIST_DIR := dist

GOOS   := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w

.DEFAULT_GOAL := help

##@ 編譯（build）

.PHONY: build
build: ## 編譯本地執行檔 → ./locku（CGO_ENABLED=0 靜態、-trimpath、strip）
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)
	@echo "built ./$(BINARY)  ($(VERSION) $(GOOS)/$(GOARCH))"

.PHONY: install
install: ## go install → $$GOBIN/PATH
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)

.PHONY: uninstall
uninstall: ## 移除已安裝的 locku（從 $$GOBIN，否則 $$GOPATH/bin）
	@dir="$$(go env GOBIN)"; [ -n "$$dir" ] || dir="$$(go env GOPATH)/bin"; \
	if [ -f "$$dir/$(BINARY)" ]; then rm -f "$$dir/$(BINARY)" && echo "removed $$dir/$(BINARY)"; else echo "not installed ($$dir/$(BINARY))"; fi

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

##@ 發布（release）

.PHONY: release-check
release-check: ## goreleaser check：驗證 .goreleaser.yaml
	goreleaser check

.PHONY: snapshot
snapshot: ## goreleaser 本機 snapshot 打包（不發布、不推 tap）→ dist/
	goreleaser release --snapshot --clean

# vhs 0.12.0 在這台機器上有時兩秒就結束、不出檔也不報錯（webu 踩過）；遇到就指定 0.11.0：
# make gif VHS=/opt/homebrew/Cellar/vhs/0.11.0/bin/vhs
VHS ?= vhs

.PHONY: gif
gif: build ## 錄 docs/demo.gif（需 vhs、JetBrainsMono Nerd Font、cmatrix；tape 與展示用 config 在 .local/demos/，不碰你的 config）
	$(VHS) .local/demos/demo.tape

##@ 執行（run）

.PHONY: run
run: ## 本地跑設定 TUI（讀真正的 config）
	go run $(PKG)

.PHONY: lock
lock: build ## 鎖住這個終端機（./locku lock）
	./$(BINARY) lock

##@ 測試 / 檢查（test）

.PHONY: test
test: ## 跑所有測試
	go test ./...

.PHONY: test-race
test-race: ## 帶 race detector 跑測試（check 也跑它）
	go test -race ./...

.PHONY: vet
vet: ## go vet ./...
	go vet ./...

.PHONY: fmt
fmt: ## gofmt -w（就地格式化 cmd/ internal/）
	gofmt -w cmd internal

.PHONY: fmt-check
fmt-check: ## gofmt -l（列出未格式化的檔；有輸出即失敗，CI 用）
	@out=$$(gofmt -l cmd internal); if [ -n "$$out" ]; then echo "not gofmt'ed:"; echo "$$out"; exit 1; fi

.PHONY: check
check: fmt-check vet test-race ## fmt-check + vet + test -race 一次跑（commit 前；race 涵蓋 test，多花十幾秒）

.PHONY: e2e
e2e: build ## 在真的 tmux 上跑端到端（e2e/tmux_attach.py：鎖定、鎖定中 attach 也被鎖、解鎖清除、lock-session、deactivate），再在 pty 上跑 custom saver（e2e/custom_lock.py），再在真的 screen 上跑（e2e/screen_lock.py：activate 寫 screenrc 與 shell rc、C-a x 與 bind 的鍵進 locku、idle 自動鎖、跑著的 screen 即時套用、deactivate）；需要 tmux、screen 與 python3
	python3 e2e/tmux_attach.py ./$(BINARY)
	python3 e2e/custom_lock.py ./$(BINARY)
	python3 e2e/screen_lock.py ./$(BINARY)

##@ 打包（package）

.PHONY: package
package: build ## 打包 → dist/locku_<ver>_<os>_<arch>.tar.gz
	@mkdir -p $(DIST_DIR)
	tar -czf $(DIST_DIR)/$(BINARY)_$(VERSION)_$(GOOS)_$(GOARCH).tar.gz $(BINARY)
	@echo "packaged $(DIST_DIR)/$(BINARY)_$(VERSION)_$(GOOS)_$(GOARCH).tar.gz"

##@ 其他

.PHONY: clean
clean: ## 移除 ./locku 與 dist/
	rm -f $(BINARY)
	rm -rf $(DIST_DIR)

.PHONY: help
help: ## 顯示這份說明
	@awk 'BEGIN {FS = ":.*?## "} \
		/^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5); next} \
		/^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
