# locku — build / run / test / package（家族共用的 target 在 family.mk）
#
# 跑 `make`（或 `make help`）列出所有指令。

BINARY := locku
include family.mk

##@ locku 自己的

.PHONY: lock
lock: build ## 鎖住這個終端機（./locku lock）
	./$(BINARY) lock

.PHONY: e2e
e2e: build ## 在真的 tmux 上跑端到端（e2e/tmux_attach.py：鎖定、鎖定中 attach 也被鎖、解鎖清除、lock-session、deactivate），再在 pty 上跑 custom saver（e2e/custom_lock.py），再在真的 screen 上跑（e2e/screen_lock.py：activate 寫 screenrc 與 shell rc、C-a x 與 bind 的鍵進 locku、idle 自動鎖、跑著的 screen 即時套用、deactivate）；需要 tmux、screen 與 python3
	python3 e2e/tmux_attach.py ./$(BINARY)
	python3 e2e/custom_lock.py ./$(BINARY)
	python3 e2e/screen_lock.py ./$(BINARY)

.PHONY: shots
shots: build ## 拍 docs/savers/*.png，README 每種 saver 一張（需 vhs、JetBrainsMono Nerd Font、cmatrix；隨機的畫面要挑，見 dev-remarks「saver 的靜態圖」）
	$(VHS) .local/demos/savers.tape
