# locku — 開發者備忘錄

> 設計權威是 `docs/function.md`（功能邊界）、`docs/ui.md`（版面）、`docs/ux.md`（互動）與
> VTP（`thoughts/tui-design`）。本文件是給接手開發的人看的備忘：現況、決定摘要、已否決的做法、
> 程式碼目錄、建置與測試、demo gif、設計文件導讀、發版、下一步。2026-09-26 從 README 搬出來：README 只介紹工具，這些留在這裡。
> 決定的完整版與日期在三份設計文件，這裡是摘要。

---

## §0 現況

設計 2026-09-24 定案，同日實作完成第一版；2026-09-25 加 Integration 的 `activate`、dino 與 custom saver、`locku pin reset`，文件同日對齊程式碼重寫。單元測試（`make check`，含 race detector）與 tmux / custom / screen 三套 pty 端到端測試皆通過。v0.1.0 於 2026-09-25 發版：GitHub Release 附四個平台的 tarball 與 checksums，brew formula 已進 vulcanshen/homebrew-tap。

三份設計文件每條決定標日期，`function.md` 的決定清單 41 條，待決清單全部為空。

平台：macOS 與 Linux（WSL 可）。不支援 Windows：鎖站在 tty、pty、`su` 與 tmux / screen 上，原生移植是另一個產品（2026-09-25，使用者定案）。

## §1 決定摘要

- **鎖在 tmux / screen 之外，不在 pane 內。** pane 內的程式永遠看不到 prefix；tmux 的 lock-command 由 client 進程以 `system()` 同步執行、期間不讀任何鍵，screen 的 LOCKPRG 同理。這是唯一正確的 hook。
- **tmux：預設整台 server 一起鎖（`lock` 可改成只鎖這個 session，2026-09-25），預設不綁熱鍵，鎖著的時候誰進來都被鎖（2026-09-24）。** `prefix :` 打 `locku` 就是 `lock-server`（command alias，不跟你的 bind 撞；要熱鍵就在 `bind-key` 自己填一個，2026-09-25），所有 session 的所有 client 一起變保護程式；tmux 本身沒有「鎖著」的狀態，locku 在 `locku lock` 啟動時把全域 `@locked` 設起來，`client-attached` / `client-session-changed` hook 看到就 `lock-client`——attach 哪個 session 都一樣，PIN 對了才清掉，tty 消失不清。閒置鎖是 tmux 每個 session 各自計時，哪個畫面閒置就鎖哪個畫面。lock-command 是 locku 的絕對路徑，由 tmux 展開 `#{socket_path}` 帶給 `locku lock -S`，非預設 socket 也對。`lock-session` 時旗立在 session 上、hooks 不變，別的 session 照常用；lock 程式從自己 session 的 lock-command 得知 session（`-t`），因為鎖定中用 tty 反查會拿到錯的 session（實測）。跑著的 server 拿到的是整個區塊（`source-file`，2026-09-25）：跟檔案同一份文字，先 undo 舊區塊做過、新區塊不做的事，再對每個既有 session 設它自己的 lock-command。
- **screen：原理照 tmux、名字用 screen 的（2026-09-25）。** `idle N lockscreen` 與 `bind <鍵> lockscreen` 寫進 screenrc、跑著的每個 session 即時 `screen -X`；LOCKPRG 只能走 shell 環境（2026-09-24 實測 `.screenrc` 的 `setenv` 對 lock 無效：lock 是 attacher 呼叫 `getenv`），所以 activate 連 shell rc 一起寫，新開 shell 生效，已在跑的 session detach 後從新 shell `screen -r`；在那之前那些 session 鎖到的是 screen 內建的 `Key:` 鎖，toast 與 `?` 都說明。沒有 `lock`：screen 沒有 server，沒有範圍可選。
- **custom saver：你自己的程式當保護程式的動畫（2026-09-25）。** profile 填一個 `command`（`sh -c` 跑，例如 `cmatrix -b`），locku 管鎖、PIN、整合。程式跑在 locku 開的 pty 上、自己一個 process group，輸出直通終端機，按鍵永遠在 locku 手上；locku 不重啟它、不讀它的畫面，解鎖就 SIGKILL 整個 group。PIN 框疊在還在動的畫面上：每一幀後面補畫一次框、DECSC / DECRC 包住、一次 synchronised update，沒有一幀被丟；框只插在 escape sequence / UTF-8 完整的切點（`safeCut`）；收起時清掉框的位置，只對閒置半秒以上的程式送 SIGWINCH（正在畫的被要求重畫會閃一下從頭來，cmatrix 實測）。程式結束（它不該結束）鎖不退：locku 的點陣板照實寫 `EXIT <code>`（`EXIT` 金字，數字 0 綠、其他 peach；被訊號殺是 128 + 號碼，沒填指令是紅色的 `NONE`），狀態列紅字寫原因。custom 沒有 bg / fg：畫面是程式的。指令不做任何 sanitize：是使用者自己機器上自己的指令。設定畫面的預覽把終端機整個交給程式，任意鍵回來。
- **忘記 PIN：`locku pin reset`（2026-09-25）。** `[y/N]` 後要登入密碼（靜態 binary 沒有 PAM，用 `su` 在 pty 上驗），過了就產生新的八位數 PIN、bcrypt 覆蓋 config 的 `pin_hash` 並顯示一次（照 elasticsearch 的 reset password，不把 `pin_hash` 清空、不留一個開著的鎖），紀錄寫 `~/.locku/data/pin-resets.log`（`$LOCKU_DATA` 可改；成功或密碼錯被拒都記，不含 PIN）；鎖定中的 locku 每一鍵重讀 `pin_hash`，下一鍵就認得新 PIN、舊的不能用；檔案壞掉沿用原本的 hash，`pin_hash` 被清空視同無 PIN。沒有終端機不跑；config 讀不到不跑。能這樣做的人本來就能殺掉鎖——locku 不是安全邊界，帳號才是。
- **進程活著 = 鎖著，結束 = 解鎖。** 任何錯誤都不得讓進程結束；只有 PIN 正確、無 PIN 模式任意鍵、tty 消失三種情況會結束。
- **非安全邊界。** 另開一條 SSH 就能 kill。定位是螢幕保護與防誤觸，config 缺失或損毀一律 fail open。攔不到的東西（function.md §0.1、§2.3）：ssh 的 `~.`、Linux VT 切換、另一條 SSH 的 `tmux attach -d` / `kill -9`、終端機模擬器自身的快捷鍵。
- **驗證只有自家 PIN**，bcrypt 存 config；PAM 留 `auth: pam` 擴充位，shadow 不做。錯誤 PIN 固定 1 秒 debounce，連續錯誤鎖定可設定、預設關。
- **saver 是 class、profile 是 object（2026-09-24 定案）。** 三種 saver：clock、dino，與 custom——你自己的程式（2026-09-25）；profile 是設定好、有名字的一份，config 裡 `profile` 指向的就是它，鎖定畫面顯示的也是它。新增 profile 從一種 saver 按 `n`，profile 的 saver 建立後不改。dino 是 Chrome 離線小恐龍遊戲當螢幕保護：地面與仙人掌向左捲、暴龍自己跳過去，無限循環、隨機障礙、隨機跳躍、不會死，不記分也不畫時間；參數只有 runner（big / small 一隻大或小暴龍，big-big / small-small / small-big / big-small 兩隻一前一後、名字就是畫面由左到右的順序、各自跳各自的；2026-09-25 定案，舊的 trex / two-trex 自動轉）、scene（grassland 仙人掌，或 desert 金字塔）、bg / fg，沒有 size，畫布自己取塞得下的最大倍率；每 70 ms 一幀。clock 的 layout row / column（直排把 `HH` / `MM` / `SS` 拆行，字大好幾倍）、size small / medium / large（一個字型像素 1 / 2 / 3 格見方）、font 3x7 / 3x5、time `HH MM` / `HH MM SS`（24 時制，不畫冒號、以空白分組）、date off 或四選一、bg / fg 兩色；沒有任何自由輸入；可 duplicate / rename / delete。
- **畫布只有一種畫法：整面 LED 點陣板。** 每格 nf-fa-square 加空格，暗格 saver 的 bg、亮格它的 fg。字形像七段顯示器：全部直角、沒有斜線、0 沒有中間斜線，數字 3 × 7，依 size 放大；字距、行距與時間裡的空白是獨立的間隔單元（small / medium 1 格、large 2 格），不跟著像素等比放大。時間與日期是兩個獨立區塊：時間先排、日期拿剩下的空間（row 在下、column 在左），各自先降 size 再去單位（時間去秒、日期去年），日期塞不下就不畫，時間塞不下才一般文字。第一幀不動畫，之後只對有變的像素做 splash 式 shuffle。字元集 39 個。各 size 需要的終端機尺寸表在 README「你會看到什麼」。
- **狀態列**：`user@host · locked since HH:MM`（`show_status` 可關）→ config error → `no PIN · any key unlocks` → custom 的結束原因，這個順序，超過終端機寬度從尾端截掉。已知的邊角（2026-09-25）：hostname 很長加窄終端時被截的是尾端的 `no PIN` / custom note——GitHub 的 macOS runner hostname 62 字元沒有點，狀態列在 80 欄放不下，`internal/ui` 的測試在 `TestMain` 把 `whoami` 固定成 `user@host`。是設計決定，未改。
- **顏色是每個 saver 自己的**，bg / fg 各三個 RGB slider，webu 的數字清單作法，不打字；滑桿用該通道自己的顏色畫（R 列 `#RR0000`），改的是草稿，`S` 才寫檔、`R` 丟掉，`q` 遇到未存草稿先問。custom 沒有顏色。
- **設定畫面的細節**：`[1]` 的 `P` 無效（`p` 才是游標那列），profile 列 `a` 直接設為啟用（決定 41，2026-09-25）；每個 `[2]` 第一列是 `Property` / `Value` 表頭；標題是家族的 powerline 膠囊——`[2]` 名字（草稿未存接黃色 `unsaved`，只有 focus 時亮）、`[1] locku`，沒有種類 tag、沒有底部 hint；`config file path` 是唯二的自由輸入，webu 的提議作法：框裡 dim 顯示目前值（沒有就是 `~/.tmux.conf` / `~/.screenrc`），`Tab` 接手編輯、`Backspace` 拒絕、沒碰就 Enter 不改；`activate` 是「區塊在不在檔案裡」，Enter 後 confirm 才寫，路徑沒填就 disabled 並說明；區塊只碰 `# >>> locku >>>` … `# <<< locku <<<`，每行尾巴 `# locku`，冪等；preview 不驗 PIN。
- **Enter = 進 `[2]` / 編輯 / 送出，Esc 只做取消，`X` 刪除，`d` 是半頁。** 畫布上任何鍵只開 prompt，第一個鍵不算輸入。

## §2 已否決，不要重提

pane 內攔截 prefix、attach 使用者現有 session、config 缺失時鎖死、PAM / shadow 進 v1、自由文字 saver、strftime 自由格式、
跑馬燈、拿掉像素間空格、`[2]` 內的 preview 框、底板 sheet、Integration popup、`--saver` 命令列覆蓋、
`locku init`、只印不寫的 setup、`locku setup` 指令、後來的 `S` / `X` 熱鍵（畫面上看不到）與底部的 Install / Uninstall 按鈕（風格不對；改成 `[2]` 第一列 `activate` on / off）、preference 每列下面的說明列（搬進 `?` help）、`.screenrc setenv LOCKPRG`（實測不通）、全域的 style 設定（顏色改為每個 saver 自己的）、
側欄 Enter 設為啟用（改在 preference › profile 選，後來加 `a` 熱鍵）、12 時制 AM/PM、時間的冒號、有斜線的字形、
tmux 預設的 `bind L`（跟使用者既有熱鍵撞，改 command alias；要綁的自己填 `bind-key`）、session 等級的 tmux 鎖（換個 session 就繞過，改整台）、
閒置鎖升級成整台（雙螢幕會被另一邊鎖到）、一個 PIN 解全部 client（要輪詢，維持各自輸）、
純隨設即得的整合寫入（conf 打錯就生檔）、純按鈕不同步（改了值檔案 stale）、`[2]` 裡的 status 列（狀態就是 `activate` 列）、` · ` 分隔的純文字標題（改膠囊鏈）、preview 也驗 PIN、標題膠囊裡的 installed / uninstalled 狀態與串在標題後面的種類（狀態是 `activate` 列；種類搬到右上角一顆獨立膠囊後也拿掉，分類資訊多餘）、`[2]` 下框右側的 config 路徑（第一版就有，不是家族慣例）、lock 程式用 tty 反查 session（鎖定中 list-clients 是空的，display-message -c 回錯的 session）、
custom 的 VT 終端機模擬器路線（多一個依賴、忠實度與效能都要驗；改直通）、custom 程式的自動重啟（不干涉生命週期）、黑畫面加紅字（用板子）、結束的字叫 COMPLETED / ERROR 再叫 DONE / ERROR（改直接寫退出碼）、PIN 框下面凍住畫面加 locku 的底色（框下面沒有動畫）、框收起時真的縮一欄再放回去逼程式重畫（改成不凍畫面就不用補畫）、對正在畫的程式送 SIGWINCH（cmatrix 會閃一下從頭來）、tmux 即時套用維護一份跟區塊平行的指令清單（改整塊 `source-file`）、鎖定畫面上的忘記密碼入口、恢復碼、清空 `pin_hash` 當 reset、`[1]` 上的全域 `P`（`p` 才是游標那列）、原生 Windows。

## §3 目錄

```
locku/
├── cmd/locku/          進入點：lock / pin reset / version / 設定 TUI；argv[0] SCREEN-LOCK
├── internal/
│   ├── config/         config.yaml 的讀寫：fail open、原子寫、0600、bcrypt PIN、pin_hash 重讀、NewPIN、data 目錄
│   ├── custom/         custom saver：程式在 pty 上、輸出經 screen writer 直通、PIN 框疊在畫面上、按鍵留給 locku、結束的字（2026-09-25）
│   ├── login/          `locku pin reset` 的登入密碼驗證：su 在 pty 上（2026-09-25）
│   ├── saver/          內容：clock 的兩種 time × 五種 date × row / column、tick；dino 的跑者、場景、障礙與自動跳躍；custom 結束的 Word
│   ├── setup/          受管區塊寫入：tmux.conf（跑著的 server 整塊 source-file）、screenrc、shell rc
│   ├── tmux/           鎖定中對 tmux 立 / 清 @locked 旗
│   ├── ui/             渲染器（font / canvas / reveal）、鎖定畫面、PIN prompt、設定 TUI 與浮層
│   └── version/        版本字串（goreleaser 以 ldflags 注入，本機 build 是 dev）
├── e2e/                pty 端到端：tmux_attach.py、custom_lock.py、screen_lock.py
└── docs/               function.md、ui.md、ux.md、dev-remarks.md、icon.svg、demo.gif
```

## §4 建置與測試

從原始碼建置（Go 1.26+，`CGO_ENABLED=0` 靜態）：

```bash
git clone https://github.com/vulcanshen/locku.git
cd locku
make build      # → ./locku（-trimpath、strip）
make install    # → $GOBIN；make uninstall 移除
```

`go install …@latest` 也能裝，但版本字串只由 goreleaser 的 ldflags 注入，這樣裝出來的 `locku version` 是 `dev`。`make` 列出所有 target。

```bash
make check                                       # fmt-check + vet + go test -race
LOCKU_DUMP=1 go test ./internal/ui -run TestDump -v   # 印出各種尺寸的畫面
make lock                                        # 編譯並鎖住這個終端機
make e2e                                         # 端到端：真的 tmux（e2e/tmux_attach.py，自己的 TMUX_TMPDIR）、custom saver（e2e/custom_lock.py）、真的 screen（e2e/screen_lock.py，自己的 SCREENDIR）；不碰你的 server 與 session
make gif                                         # 重錄 docs/demo.gif，見 §4.1
```

TUI 行為全部用 programmatic model test 驗證（不需要 tty）；`make check` 帶 race detector（custom 的 pump 與 screen writer、login 的 su、custom 鎖的 prompt 各有 goroutine，沒有 race detector 看不出來，多花十幾秒）。tmux / custom / screen 的驗收（`docs/function.md` §12）以 python pty harness 跑真的 binary 完成：鎖定中 prefix+d、prefix+c、Ctrl+C 被吞、對 PIN 後 client 回來、鎖著的時候 attach 也被鎖、畫面上切 lock-session 時跑著的 server 整塊換掉、custom 的框疊在動畫上且程式被殺乾淨、screen 的 `C-a x` 與 `bind` 的鍵進 locku、`idle` 自動鎖、跑著的 session 即時收到設定、tty 關閉進程結束。

任何看狀態列文字的 ui 測試都要固定 `whoami`（`lockscreen_test.go` 的 `TestMain`），原因見 §1 狀態列那條。

`V` 的 splash 彩蛋就是 `docs/icon.svg`，一格對一格（2026-09-26）：第一版在 icon 出現前一天畫，裡面是一個自創的掛鎖，icon 進來後沒跟上。`splash_test.go` 的 `TestSplashIsTheIcon` 直接讀 icon.svg 比對 `logoPixels`，icon 改了 splash 沒跟就失敗。揭露順序：底色 → L、O、C、K 由外往內 → 深藍 U 由下往上。

### §4.1 demo gif（2026-09-26）

README 只放一個 gif，`docs/demo.gif`，用 VHS 錄。tape 與展示用 config 照家族慣例放在 `.local/demos/`（gitignore，不進版控）：`demo.tape` 與 `config.yaml`（clock / dino / 以 cmatrix 當 custom 的三個 profile，PIN 1234，`htpasswd -nbBC 10 x 1234` 產生）。每次錄都把 config 複製到 `.local/demos/config`、以 `LOCKU_CONFIG` / `LOCKU_DATA` 指過去，不碰真正的設定；tape 從不按 `activate`，所以不會寫到真的 `~/.tmux.conf`。需要 VHS、JetBrainsMono Nerd Font、cmatrix。

VHS 0.12.0 在這台機器上會印 `Creating docs/demo.gif...` 卻不出檔（webu 也踩過），用 0.11.0：`make gif VHS=/opt/homebrew/Cellar/vhs/0.11.0/bin/vhs`。VHS 的 `Type "…"` 不吃反斜線跳脫，字串裡要引號就用單引號。展示 config 設 `show_status: false`：狀態列會照實顯示錄影機器的 `user@host`，不公開進 README（2026-09-26）。

## §5 設計文件導讀與用什麼做的

| 檔案 | 回答什麼 | 順序 |
|---|---|---|
| [`function.md`](function.md) | 為什麼鎖站在 tmux / screen 之外、三種進入點同一契約、訊號表、狀態機、PIN 與無 PIN 模式、三種 saver、畫布渲染器、CLI、config、Integration 怎麼寫檔、決定清單、驗收 | 1 |
| [`ui.md`](ui.md) | 設定畫面兩個面板的 grid、鎖定畫布、每個欄位怎麼呈現、popup、PIN prompt 四個狀態、色帶、存檔 | 2 |
| [`ux.md`](ux.md) | core-key 語意、Space menu 內容、`?` 全域、每種欄位怎麼填、PIN 三連問、hotkey 分層、浮層、時間軸 | 3 |
| [`icon.svg`](icon.svg) | 圖示：黑底方塊上家族的方塊字 mark，深藍 U 包住金色的 L、O、C、K；splash 照它畫 | — |

Go、[Bubble Tea](https://github.com/charmbracelet/bubbletea) 與 [Lip Gloss](https://github.com/charmbracelet/lipgloss)、浮層用 [bubbletea-overlay](https://github.com/rmhubbert/bubbletea-overlay)、custom saver 的程式與登入密碼驗證用 [creack/pty](https://github.com/creack/pty)、`charmbracelet/x/term` 與 `charmbracelet/x/ansi`、`muesli/cancelreader`、PIN 用 `golang.org/x/crypto/bcrypt`、config 用 `gopkg.in/yaml.v3`。色系 catppuccin-mocha。

## §6 發版

發版走家族的流程：push 一個 `v*` tag，GitHub Actions（`.github/workflows/release.yml`）在 ubuntu 與 macOS 跑 `go test -race`，過了 goreleaser 打包四個平台的 tarball 與 checksums、更新 vulcanshen/homebrew-tap 的 `locku.rb`，release notes 是 `CHANGELOG.md` 對應的那一節。CHANGELOG 只記 binary 行為的變動；純文件、打包、CI 的改動不記。

## §7 下一步

1. 8 小時 CPU / 記憶體觀察（`function.md` §12 最後一項，尚未做）。
