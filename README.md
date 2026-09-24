# locku

終端機的螢幕保護程式加 PIN 鎖。u-family 第五個成員（kbu / filu / sshu / webu / locku），依 VTP 設計。

作為 tmux 的 `lock-command`、screen 的 `LOCKPRG`，或在裸 tty 直接執行：tmux / screen 把真實 tty 交給它，
它把整個畫面鋪成 LED 點陣板畫時鐘，任何鍵只會叫出 PIN 輸入框，驗證通過才把 tty 還回去。
不設 PIN 就是純螢幕保護，任何鍵解鎖。

> 設計 2026-09-24 定案，同日實作完成第一版：渲染器、`locku lock`、設定 TUI、`locku setup`，
> 單元測試與 tmux / screen 的 pty 端到端測試皆通過。尚未發版（v0.1.0 待 icon、英文 README、brew formula）。

## 四個指令

| 指令 | 作用 |
|---|---|
| `locku` | 設定 TUI：每個 saver 的 layout / time / date 與 bg / fg 顏色，preference 的 PIN、啟用中的 saver、show_status、prompt_timeout、lockout。除了顏色走草稿（`S` 存、`R` 丟），每次改動立即寫檔。`P` 就地預覽鎖定畫面 |
| `locku lock` | 鎖住當前 tty。tmux、screen、裸 tty 都是叫這個 |
| `locku setup [tmux\|screen]` | 把整合設定寫進 `~/.tmux.conf`（有 server 在跑就即時套用）與 `~/.screenrc` 加 shell rc；不帶參數兩個都做。只碰 `# >>> locku >>>` … `# <<< locku <<<` 受管區塊，冪等 |
| `locku version` | 版本 |

argv[0] 是 `SCREEN-LOCK` 時視同 `locku lock`，因為 screen 的 LOCKPRG 是 execl、不能帶參數。

## 安裝

```bash
git clone https://github.com/vulcanshen/locku.git
cd locku
make build      # → ./locku（CGO_ENABLED=0 靜態）
./locku setup   # 寫 tmux / screen 的整合設定
./locku         # 設 PIN（可省略：不設就是純螢幕保護）
./locku lock    # 現在就鎖
```

brew formula 與 install.sh 在 v0.1.0 發版後可用。**Nerd Font 必裝**：點陣板的每個像素就是 nf-fa-square。

## 鎖定畫面

任何鍵開 PIN prompt，那個鍵不算輸入；`Enter` 送出、`Esc` 回 saver、`Backspace` 刪一字。
錯誤 PIN 邊框變紅 1 秒並吞掉所有輸入；連錯 `lockout_after` 次進入 `lockout_seconds` 秒倒數；
`prompt_timeout` 秒沒按鍵 prompt 自動收起。狀態列 `user@host · locked since HH:MM`，沒設 PIN 時標明 `no PIN · any key unlocks`。

Ctrl+C、Ctrl+Z、Ctrl+\ 只是按鍵；SIGINT / SIGTERM / SIGHUP 一律忽略；panic 後鎖定畫面重新升起。
進程只在三種情況結束：PIN 正確、無 PIN 模式任意鍵、tty 消失。

攔不到的（非安全邊界，見 `docs/function.md` §0.1、§2.3）：ssh 的 `~.`、Linux VT 切換、另一條 SSH 的 `tmux attach -d` / `kill -9`、終端機模擬器自身的快捷鍵。

## 設定畫面

```
╔ [1] locku ═════════════╗╭ [2] clock · unsaved ─────────────────────────────╮
║ Savers                 ║│ name              clock                          │
║ ● clock                ║│ type              clock                          │
║   clock2               ║│ layout            row                            │
║ Settings               ║│ time              HH MM                          │
║   preference           ║│ date              off                            │
║                        ║│ bg                ■ #313244  →  ■ #ff3244        │
║                        ║│   R               ───────────● 255               │
║                        ║│   G               ──●───────── 50                │
║                        ║│   B               ───●──────── 68                │
║                        ║│ fg                ■ #f2b753                      │
╚════════════════════════╝╰──────────────── ~/.config/locku/config.yaml ─────╯
 space menu   ? help   tab/1-2 panels   q quit
```

`Tab` / `1` / `2` 切面板、`Enter` 進 `[2]` 或編輯、`Esc` 關浮層、`Space` 列出當前能做的事、`?` 全域動作。
`[1]` 的 saver：`p` 預覽這個 saver、`D` duplicate、`r` rename、`X` delete；`[2]` saver 上：`P` 預覽這個 saver、`S` 存顏色草稿、`R` 丟掉；
`[2]` preference 的 PIN 列：`x` clear。啟用哪個 saver在 preference › saver 選，側欄的 `●` 只顯示。`P` 預覽、`q` 離開。

## 文件

| 檔案 | 回答什麼 | 順序 |
|---|---|---|
| [`docs/function.md`](docs/function.md) | 為什麼不能在 pane 內攔 prefix、三種進入點同一契約、訊號表、狀態機、PIN 驗證、無 PIN 模式、saver 實例、畫布渲染器與退階、CLI、config、`locku setup` 怎麼寫檔（含 2026-09-24 的 screen 實測）、決定清單 21 條、MVP 驗收 | 1 |
| [`docs/ui.md`](docs/ui.md) | 設定畫面兩個面板的 grid、鎖定畫布的點陣板、每個欄位怎麼呈現、popup 清單、PIN prompt 四個狀態、色帶、chrome、存檔 | 2 |
| [`docs/ux.md`](docs/ux.md) | core-key 語意、Space menu 內容、`?` 全域、每種欄位怎麼填、PIN 三連問、畫布 prompt 事件表、hotkey 分層與撞字檢查、浮層、時間軸 | 3 |

三份都是繁體中文，每條決定標日期，待決清單全部為空。

## 決定摘要

- **鎖在 tmux / screen 之外，不在 pane 內。** pane 內的程式永遠看不到 prefix；tmux 的 lock-command 由 client 進程以 `system()` 同步執行、期間不讀任何鍵，screen 的 LOCKPRG 同理。這是唯一正確的 hook。
- **進程活著 = 鎖著，結束 = 解鎖。** 任何錯誤都不得讓進程結束；只有 PIN 正確、無 PIN 模式任意鍵、tty 消失三種情況會結束。
- **非安全邊界。** 另開一條 SSH 就能 kill。定位是螢幕保護與防誤觸，config 缺失或損毀一律 fail open。
- **驗證只有自家 PIN**，bcrypt 存 config；PAM 留 `auth: pam` 擴充位，shadow 不做。錯誤 PIN 固定 1 秒 debounce，連續錯誤鎖定可設定、預設關。
- **saver 是具名實例，v1 只有 clock 一個 type。** layout row / column（直排把 `HH` / `MM` / `SS` 拆行，字大好幾倍）、size small / medium / large（一個字型像素 1 / 2 / 3 格見方）、font 3x7 / 3x5、time `HH MM` / `HH MM SS`（24 時制，不畫冒號、以空白分組）、date off 或四選一、bg / fg 兩色；沒有任何自由輸入；可 duplicate / rename / delete。
- **畫布只有一種畫法：整面 LED 點陣板。** 每格 nf-fa-square 加空格，暗格 saver 的 bg、亮格它的 fg。字形像七段顯示器：全部直角、沒有斜線、0 沒有中間斜線，數字 3 × 7，依 size 放大；塞不下依偏好取第一個塞得下的內容（去年 → 去秒、日期留著 → 去日期、秒回來 → 去秒），還不行才降一級 size，最後才一般文字。第一幀不動畫，之後只對有變的像素做 splash 式 shuffle。字元集 39 個。

  各 size 需要的終端機（欄 × 列，3x7 / 3x5）：

  | 內容 | small | medium | large |
  |---|---|---|---|
  | `HH MM` 一行 | 40 × 10 / 8 | 76 × 17 / 13 | 112 × 24 / 18 |
  | `HH MM SS` 一行 | 62 × 10 / 8 | 120 × 17 / 13 | 178 × 24 / 18 |
  | `HH` / `MM` 直排 | 18 × 18 / 14 | 32 × 33 / 25 | 46 × 48 / 36 |
  | `HH` / `MM` / `SS` 直排 | 18 × 26 / 20 | 32 × 49 / 37 | 46 × 72 / 54 |
- **顏色是每個 saver 自己的**，bg / fg 各三個 RGB slider，webu 的數字清單作法，不打字；滑桿用該通道自己的顏色畫（R 列 `#RR0000`），改的是草稿，`S` 才寫檔、`R` 丟掉，`q` 遇到未存草稿先問。
- **screen 的 LOCKPRG 只能走 shell 環境**（2026-09-24 實測）：`.screenrc` 的 `setenv` 對 lock 無效，因為 lock 是 attacher 呼叫 `getenv`，`.screenrc` 只有後端讀。`locku setup screen` 因此寫 `~/.zshrc` / `~/.bashrc` / fish 的受管區塊，新開 shell 生效；已在跑的 session detach 後從新 shell `screen -r` 即可。
- **Enter = 進 `[2]` / 編輯 / 送出，Esc 只做取消，`X` 刪除，`d` 是半頁。** 畫布上任何鍵只開 prompt，第一個鍵不算輸入。

## 已否決，不要重提

pane 內攔截 prefix、attach 使用者現有 session、config 缺失時鎖死、PAM / shadow 進 v1、自由文字 saver、strftime 自由格式、
跑馬燈、拿掉像素間空格、`[2]` 內的 preview 框、底板 sheet、Integration popup、`--saver` 命令列覆蓋、
`locku init`、只印不寫的 setup、`.screenrc setenv LOCKPRG`（實測不通）、全域的 style 設定（顏色改為每個 saver 自己的）、
側欄 Enter 設為啟用（改在 preference › saver 選）、12 時制 AM/PM、時間的冒號、有斜線的字形。

## 目錄

```
locku/
├── cmd/locku/          進入點：lock / setup / version / 設定 TUI；argv[0] SCREEN-LOCK
├── internal/
│   ├── config/         config.yaml 的讀寫：fail open、原子寫、0600、bcrypt PIN
│   ├── saver/          內容：clock 的兩種 time × 五種 date × row / column、tick、退階梯
│   ├── setup/          受管區塊寫入：tmux.conf、screenrc、shell rc
│   └── ui/             渲染器（font / canvas / reveal）、鎖定畫面、PIN prompt、設定 TUI 與浮層
└── docs/               function.md、ui.md、ux.md
```

## 開發

```bash
make check                                       # fmt-check + vet + test
LOCKU_DUMP=1 go test ./internal/ui -run TestDump -v   # 印出各種尺寸的畫面
make lock                                        # 編譯並鎖住這個終端機
```

TUI 行為全部用 programmatic model test 驗證（不需要 tty）；tmux / screen 整合的驗收（`docs/function.md` §12）
以 python pty harness 跑真的 binary 完成：鎖定中 prefix+d、prefix+c、Ctrl+C 被吞、錯 PIN 顯示 wrong、
對 PIN 後 client 回來、screen 的 `C-a x` 進 locku、tty 關閉進程結束。

## 下一步

1. 在真的終端機看一次顏色與 Nerd Font 寬度（CJK 字型可能把 nf-fa-square 畫成兩格）。
2. `docs/icon.svg`、README 英文版、CHANGELOG 收 0.1.0、tag、brew formula。
3. 8 小時 CPU / 記憶體觀察（§12 最後一項）。
