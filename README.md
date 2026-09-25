# locku

終端機的螢幕保護程式加 PIN 鎖。u-family 第五個成員（kbu / filu / sshu / webu / locku），依 VTP 設計；圖示是家族的方塊字 mark，[`docs/icon.svg`](docs/icon.svg)。

作為 tmux 的 `lock-command`、screen 的 `LOCKPRG`，或在裸 tty 直接執行：tmux / screen 把真實 tty 交給它，
它把整個畫面鋪成 LED 點陣板畫時鐘、跑小恐龍，或跑你自己的程式；任何鍵只會叫出 PIN 輸入框，驗證通過才把 tty 還回去。
不設 PIN 就是純螢幕保護，任何鍵解鎖。

> 設計 2026-09-24 定案，同日實作完成第一版；2026-09-25 加 Integration 的 `activate`、dino 與 custom saver、`locku pin reset`，文件同日對齊程式碼重寫。
> 單元測試（`make check`，含 race detector）與 tmux / custom / screen 三套 pty 端到端測試皆通過。尚未發版（v0.1.0 待英文 README、brew formula、tag）。

**平台**：macOS 與 Linux（WSL 可）。不支援 Windows：鎖站在 tty、pty、`su` 與 tmux / screen 上，原生移植是另一個產品（2026-09-25，使用者定案）。

## 四個指令

| 指令 | 作用 |
|---|---|
| `locku` | 設定 TUI：三種 saver（clock、dino、custom）各生幾個有名字的 profile，每個 profile 的參數與 bg / fg 顏色（custom 只有 `command`，沒有顏色）；preference 的 PIN、啟用中的 profile、show_status、pin_prompt_timeout、wrong_pin_attempts / wrong_pin_attempt_cooldown；Integration 的 tmux / screen（見下）。除了顏色走草稿（`S` 存、`R` 丟），每次改動立即寫檔。`[2]` 上 `P` 就地預覽（任意鍵回來，不驗 PIN） |
| `locku lock` | 鎖住當前 tty。tmux、screen、裸 tty 都是叫這個。`-S <socket>` / `-t <session>` 是 tmux 整合寫進 lock-command 的，你不用打 |
| `locku pin reset` | 忘記 PIN 的路，見下 |
| `locku version` | 版本（`locku help` 印用法） |

argv[0] 是 `SCREEN-LOCK` 時視同 `locku lock`，因為 screen 的 LOCKPRG 是 execl、不能帶參數。

## 忘記 PIN

從你自己的任何一個 shell 跑 `locku pin reset`：先問 `[y/N]`，再要你的**登入密碼**（帳號就是唯一的邊界；靜態 binary 沒有 PAM，用 `su` 在 pty 上驗），兩關都過才產生一組新的八位數 PIN、把它的 bcrypt 覆蓋 config 的 `pin_hash`、顯示一次——永遠不把 PIN 清空、不留一個開著的鎖，之後到設定畫面換成自己要的。鎖定中的畫面每一鍵都重讀 `pin_hash`，下一鍵就認得新 PIN、舊的不能用；檔案壞掉沿用原本的 hash，`pin_hash` 被清空視同無 PIN。每次 reset（成功或密碼錯被拒）記在 `~/.locku/data/pin-resets.log`（`$LOCKU_DATA` 可改），不含 PIN。沒有終端機不跑；config 讀不到不跑。能這樣做的人本來就能殺掉鎖——locku 不是安全邊界，帳號才是。

## 安裝

```bash
git clone https://github.com/vulcanshen/locku.git
cd locku
make build      # → ./locku（CGO_ENABLED=0 靜態）
./locku         # Integration › tmux / screen：填 config file path，activate 打開寫整合設定；設 PIN（可省略：不設就是純螢幕保護）
./locku lock    # 現在就鎖
```

brew formula 與 install.sh 在 v0.1.0 發版後可用。**Nerd Font 必裝**：點陣板的每個像素就是 nf-fa-square。

## 鎖定畫面

任何鍵開 PIN prompt，那個鍵不算輸入；`Enter` 送出、`Esc` 回 saver、`Backspace` 刪一字。
錯誤 PIN 邊框變紅 1 秒並吞掉所有輸入；連錯 `wrong_pin_attempts` 次進入 `wrong_pin_attempt_cooldown` 秒倒數；
`pin_prompt_timeout` 秒沒按鍵 prompt 自動收起。閒置幾秒自動鎖由 tmux 的 `lock-after-time`、screen 的 `idle` 決定（各自一個，activate 開著時原樣寫進去、一改就重寫，預設 300，0 關）。狀態列 `user@host · locked since HH:MM`，沒設 PIN 時標明 `no PIN · any key unlocks`。

Ctrl+C、Ctrl+Z、Ctrl+\ 只是按鍵；SIGINT / SIGTERM / SIGHUP 一律忽略；panic 後鎖定畫面重新升起。
進程只在三種情況結束：PIN 正確、無 PIN 模式任意鍵、tty 消失。

攔不到的（非安全邊界，見 `docs/function.md` §0.1、§2.3）：ssh 的 `~.`、Linux VT 切換、另一條 SSH 的 `tmux attach -d` / `kill -9`、終端機模擬器自身的快捷鍵。

## 自己的程式當保護程式（custom saver）

profile 選 custom、填一個 `command`（`sh -c` 跑，例如 `cmatrix -b`），locku 管鎖、PIN、整合。程式跑在 locku 開的 pty 上、自己一個 process group，輸出原樣直通終端機，按鍵永遠在 locku 手上；locku 不重啟它、不讀它的畫面，解鎖就 SIGKILL 整個 group。按鍵時 PIN 框直接疊在還在動的畫面上——程式每送一段輸出，框就在後面補畫一次，包在一次 synchronised update 裡，沒有一幀被丟；Esc 或逾時就把框的位置清掉、畫面繼續，只有閒置半秒以上沒在畫的程式才會被要求重畫一次。程式結束（它不該結束）鎖不退：locku 的點陣板照實寫 `EXIT <code>`（`EXIT` 金字，數字 0 綠、其他 peach；被訊號殺是 128 + 號碼；沒填指令或起不來是紅色的 `NONE`），狀態列紅字寫原因。custom 沒有 bg / fg：畫面是程式的。指令不做任何 sanitize：是你自己機器上自己的指令。設定畫面的預覽把終端機整個交給程式，任意鍵回來。

## 設定畫面

```
╔[1] locku═════════════╗╭[2] clock  unsaved─────────────────────────────╮
║ Profiles               ║│ Property          Value                          │
║ ● clock                ║│ name              clock                          │
║   clock2               ║│ saver             clock                          │
║   dino                 ║│ layout            row                            │
║ Savers                 ║│ size              medium                         │
║   clock                ║│ time              HH MM                          │
║   dino                 ║│ date              off                            │
║ Integration            ║│ bg                ■ #313244  →  ■ #ff3244        │
║   tmux                 ║│   R               ───────────● 255               │
║   screen               ║│   G               ──●───────── 50                │
║ Settings               ║│   B               ───●──────── 68                │
║   preference           ║│ fg                ■ #f2b753                      │
╚════════════════════════╝╰──────────────────────────────────────────────────╯
 space menu   ? help   tab/1-2 panels   q quit
```

**Profiles** 是你設定好的、有名字的 saver（object）：new / duplicate / rename / delete 都在這裡，`●` 是啟用中的那個。
**Savers** 是種類（class）：clock、dino、custom，沒有名字、不能增刪，`[2]` 是說明加**預設值**——之後用這種 saver 新增的 profile 就長這樣，
改它不動既有的 profile；在上面按 `n` 生一個它的 profile、`p` 用預設值預覽。內建預設：clock 是 row / large / 3x5 / `HH MM SS` / `YYYY-MM-DD`，dino 是 big / grassland，custom 的 command 是空的。

`Tab` / `1` / `2` 切面板、`Enter` 進 `[2]` 或編輯、`Esc` 關浮層、`Space` 列出當前能做的事、`?` 全域動作。
`[1]` 的 saver：`n` new profile、`p` 預覽預設值；`[1]` 的 profile：`p` 預覽這個 profile、`D` duplicate、`r` rename、`X` delete；`[2]` profile / saver 上：`P` 預覽、`S` 存顏色草稿、`R` 丟掉（custom 只有 `P`）；
`[2]` preference 的 PIN 列：已設時 Enter 先驗目前的 PIN，再選 `New PIN` 或 `Remove PIN`（Remove 立即生效）。啟用哪個 profile 在 preference › profile 選，側欄的 `●` 只顯示。`P` 只在 `[2]` 上預覽（profile / saver 的 `[2]` 是那一個、其他是啟用中的；`[1]` 上不作用，`p` 才是游標那列）、`q` 離開。
**Integration** 的 tmux / screen 的 `[2]` 是 `activate`（on / off）、`config file path`（要寫的設定檔）、一條分隔線、然後工具自己的 key：tmux 的 `lock`（`lock-server` 整台，或 `lock-session` 只鎖這個 session）、閒置鎖——用工具自己的設定名稱，tmux 是 `lock-after-time`、screen 是 `idle`——tmux 的 `bind-key`（prefix 之後按哪個鍵就鎖，照 tmux 的寫法 `l`、`C-l`，寫成 `bind-key <鍵> <lock>`）、screen 的 `bind`（C-a 之後按哪個鍵就鎖，照 screen 的寫法 `l`、`^L`，寫成 `bind <鍵> lockscreen`；`C-a x` 內建就鎖），空就不綁；screen 沒有 `lock`，它沒有 server 可選範圍。`activate` 就是區塊在不在檔案裡，Enter 後 confirm 才寫；開著時改任何一列就直接重寫區塊，tmux 有 server 在跑就整塊 `source-file` 上去、screen 即時 `screen -X` 進跑著的 session，關掉就拿掉。只碰 `# >>> locku >>>` … `# <<< locku <<<` 區塊，每行尾巴 `# locku`，冪等；路徑沒填就不能按、不猜。每個 `[2]` 第一列是 `Property` / `Value` 表頭；標題是家族的 powerline 膠囊：`[2]` 名字（草稿未存接 `unsaved`）、`[1] locku`。`config file path` 是唯二的自由輸入，webu 的提議作法：框裡 dim 顯示目前值（沒有就是 `~/.tmux.conf` / `~/.screenrc`），`Tab` 接手編輯、`Backspace` 拒絕、沒碰就 Enter 不改。每個設定是什麼：focus 在 preference 或 tmux / screen 的 `[2]` 時按 `?`，help 只列那個面板的設定說明（自動換行），其他地方的 `?` 是鍵。

## 文件

| 檔案 | 回答什麼 | 順序 |
|---|---|---|
| [`docs/function.md`](docs/function.md) | 為什麼不能在 pane 內攔 prefix、三種進入點同一契約、訊號表、狀態機、PIN 驗證、無 PIN 模式、`locku pin reset`、三種 saver（含 custom 的 pty 與疊框）、畫布渲染器與退階、CLI、config、Integration 怎麼寫檔（含 tmux / screen 的實測）、技術選型與平台、決定清單 40 條、MVP 驗收 | 1 |
| [`docs/ui.md`](docs/ui.md) | 設定畫面兩個面板的 grid、鎖定畫布的點陣板與 `EXIT` 板、每個欄位怎麼呈現、popup 清單、PIN prompt 四個狀態、色帶、chrome、存檔 | 2 |
| [`docs/ux.md`](docs/ux.md) | core-key 語意、Space menu 內容、`?` 全域、每種欄位怎麼填、PIN 三連問、畫布 prompt 事件表、hotkey 分層與撞字檢查、浮層、時間軸 | 3 |
| [`docs/icon.svg`](docs/icon.svg) | 圖示：黑底方塊上家族的方塊字 mark，locku 版（2026-09-25） | — |

三份文件都是繁體中文，每條決定標日期，待決清單全部為空。

## 決定摘要

- **鎖在 tmux / screen 之外，不在 pane 內。** pane 內的程式永遠看不到 prefix；tmux 的 lock-command 由 client 進程以 `system()` 同步執行、期間不讀任何鍵，screen 的 LOCKPRG 同理。這是唯一正確的 hook。
- **tmux：預設整台 server 一起鎖（`lock` 可改成只鎖這個 session，2026-09-25），預設不綁熱鍵，鎖著的時候誰進來都被鎖（2026-09-24）。** `prefix :` 打 `locku` 就是 `lock-server`（command alias，不跟你的 bind 撞；要熱鍵就在 `bind-key` 自己填一個，2026-09-25），所有 session 的所有 client 一起變保護程式；tmux 本身沒有「鎖著」的狀態，locku 在 `locku lock` 啟動時把全域 `@locked` 設起來，`client-attached` / `client-session-changed` hook 看到就 `lock-client`——attach 哪個 session 都一樣，PIN 對了才清掉，tty 消失不清。閒置鎖是 tmux 每個 session 各自計時，哪個畫面閒置就鎖哪個畫面。lock-command 是 locku 的絕對路徑，由 tmux 展開 `#{socket_path}` 帶給 `locku lock -S`，非預設 socket 也對。`lock-session` 時旗立在 session 上、hooks 不變，別的 session 照常用；lock 程式從自己 session 的 lock-command 得知 session（`-t`），因為鎖定中用 tty 反查會拿到錯的 session（實測）。跑著的 server 拿到的是整個區塊（`source-file`，2026-09-25）：跟檔案同一份文字，先 undo 舊區塊做過、新區塊不做的事，再對每個既有 session 設它自己的 lock-command。
- **screen：原理照 tmux、名字用 screen 的（2026-09-25）。** `idle N lockscreen` 與 `bind <鍵> lockscreen` 寫進 screenrc、跑著的每個 session 即時 `screen -X`；LOCKPRG 只能走 shell 環境（2026-09-24 實測 `.screenrc` 的 `setenv` 對 lock 無效：lock 是 attacher 呼叫 `getenv`），所以 activate 連 shell rc 一起寫，新開 shell 生效，已在跑的 session detach 後從新 shell `screen -r`；在那之前那些 session 鎖到的是 screen 內建的 `Key:` 鎖，toast 與 `?` 都說明。沒有 `lock`：screen 沒有 server，沒有範圍可選。
- **custom saver：你自己的程式當保護程式的動畫（2026-09-25）。** profile 填一個 `command`（`sh -c` 跑，例如 `cmatrix -b`），locku 管鎖、PIN、整合。程式跑在 locku 開的 pty 上、自己一個 process group，輸出直通終端機，按鍵永遠在 locku 手上；locku 不重啟它、不讀它的畫面，解鎖就 SIGKILL 整個 group。PIN 框疊在還在動的畫面上：每一幀後面補畫一次框、DECSC / DECRC 包住、一次 synchronised update，沒有一幀被丟；收起時清掉框的位置，只對閒置半秒以上的程式送 SIGWINCH（正在畫的被要求重畫會閃一下從頭來，cmatrix 實測）。程式結束（它不該結束）鎖不退：locku 的點陣板照實寫 `EXIT <code>`（`EXIT` 金字，數字 0 綠、其他 peach；被訊號殺是 128 + 號碼，沒填指令是紅色的 `NONE`），狀態列紅字寫原因。custom 沒有 bg / fg：畫面是程式的。指令不做任何 sanitize。
- **忘記 PIN：`locku pin reset`（2026-09-25）。** `[y/N]` 後要登入密碼（`su` 在 pty 上驗），過了就產生新 PIN 覆蓋 config 並顯示一次（照 elasticsearch 的 reset password，不把 `pin_hash` 清空），紀錄寫 `~/.locku/data/pin-resets.log`（不含 PIN）；鎖定中的 locku 每一鍵重讀 `pin_hash`。
- **進程活著 = 鎖著，結束 = 解鎖。** 任何錯誤都不得讓進程結束；只有 PIN 正確、無 PIN 模式任意鍵、tty 消失三種情況會結束。
- **非安全邊界。** 另開一條 SSH 就能 kill。定位是螢幕保護與防誤觸，config 缺失或損毀一律 fail open。
- **驗證只有自家 PIN**，bcrypt 存 config；PAM 留 `auth: pam` 擴充位，shadow 不做。錯誤 PIN 固定 1 秒 debounce，連續錯誤鎖定可設定、預設關。
- **saver 是 class、profile 是 object（2026-09-24 定案）。** 三種 saver：clock、dino，與 custom——你自己的程式（2026-09-25）；profile 是設定好、有名字的一份，config 裡 `profile` 指向的就是它，鎖定畫面顯示的也是它。新增 profile 從一種 saver 按 `n`，profile 的 saver 建立後不改。dino 是 Chrome 離線小恐龍遊戲當螢幕保護：地面與仙人掌向左捲、暴龍自己跳過去，無限循環、隨機障礙、隨機跳躍、不會死，不記分也不畫時間；參數只有 runner（big / small 一隻大或小暴龍，big-big / small-small / small-big / big-small 兩隻一前一後、名字就是畫面由左到右的順序、各自跳各自的；2026-09-25 定案，舊的 trex / two-trex 自動轉）、scene（grassland 仙人掌，或 desert 金字塔）、bg / fg，沒有 size，畫布自己取塞得下的最大倍率；每 70 ms 一幀。clock 的 layout row / column（直排把 `HH` / `MM` / `SS` 拆行，字大好幾倍）、size small / medium / large（一個字型像素 1 / 2 / 3 格見方）、font 3x7 / 3x5、time `HH MM` / `HH MM SS`（24 時制，不畫冒號、以空白分組）、date off 或四選一、bg / fg 兩色；沒有任何自由輸入；可 duplicate / rename / delete。
- **畫布只有一種畫法：整面 LED 點陣板。** 每格 nf-fa-square 加空格，暗格 saver 的 bg、亮格它的 fg。字形像七段顯示器：全部直角、沒有斜線、0 沒有中間斜線，數字 3 × 7，依 size 放大；字距、行距與時間裡的空白是獨立的間隔單元（small / medium 1 格、large 2 格），不跟著像素等比放大。時間與日期是兩個獨立區塊：時間先排、日期拿剩下的空間（row 在下、column 在左），各自先降 size 再去單位（時間去秒、日期去年），日期塞不下就不畫，時間塞不下才一般文字。第一幀不動畫，之後只對有變的像素做 splash 式 shuffle。字元集 39 個。

  各 size 需要的終端機（欄 × 列，3x7 / 3x5）：

  | 內容 | small | medium | large |
  |---|---|---|---|
  | `HH MM` 一行 | 38 × 10 / 8 | 62 × 17 / 13 | 96 × 24 / 18 |
  | `HH MM SS` 一行 | 58 × 10 / 8 | 94 × 17 / 13 | 148 × 24 / 18 |
  | `HH` / `MM` 直排 | 18 × 18 / 14 | 30 × 32 / 24 | 44 × 47 / 35 |
  | `HH` / `MM` / `SS` 直排 | 18 × 26 / 20 | 30 × 47 / 35 | 44 × 70 / 52 |
- **顏色是每個 saver 自己的**，bg / fg 各三個 RGB slider，webu 的數字清單作法，不打字；滑桿用該通道自己的顏色畫（R 列 `#RR0000`），改的是草稿，`S` 才寫檔、`R` 丟掉，`q` 遇到未存草稿先問。custom 沒有顏色。
- **平台：macOS / Linux（WSL 可），不支援 Windows（2026-09-25）。** 鎖站在 tty、pty、`su` 與 tmux / screen 上。
- **Enter = 進 `[2]` / 編輯 / 送出，Esc 只做取消，`X` 刪除，`d` 是半頁。** 畫布上任何鍵只開 prompt，第一個鍵不算輸入。

## 已否決，不要重提

pane 內攔截 prefix、attach 使用者現有 session、config 缺失時鎖死、PAM / shadow 進 v1、自由文字 saver、strftime 自由格式、
跑馬燈、拿掉像素間空格、`[2]` 內的 preview 框、底板 sheet、Integration popup、`--saver` 命令列覆蓋、
`locku init`、只印不寫的 setup、`locku setup` 指令、後來的 `S` / `X` 熱鍵（畫面上看不到）與底部的 Install / Uninstall 按鈕（風格不對；改成 `[2]` 第一列 `activate` on / off）、preference 每列下面的說明列（搬進 `?` help）、`.screenrc setenv LOCKPRG`（實測不通）、全域的 style 設定（顏色改為每個 saver 自己的）、
側欄 Enter 設為啟用（改在 preference › saver 選）、12 時制 AM/PM、時間的冒號、有斜線的字形、
tmux 預設的 `bind L`（跟使用者既有熱鍵撞，改 command alias；要綁的自己填 `bind-key`）、session 等級的 tmux 鎖（換個 session 就繞過，改整台）、
閒置鎖升級成整台（雙螢幕會被另一邊鎖到）、一個 PIN 解全部 client（要輪詢，維持各自輸）、
純隨設即得的整合寫入（conf 打錯就生檔）、純按鈕不同步（改了值檔案 stale）、`[2]` 裡的 status 列（狀態就是 `activate` 列）、` · ` 分隔的純文字標題（改膠囊鏈）、preview 也驗 PIN、標題膠囊裡的 installed / uninstalled 狀態與串在標題後面的種類（狀態是 `activate` 列；種類搬到右上角一顆獨立膠囊後也拿掉，分類資訊多餘）、`[2]` 下框右側的 config 路徑（第一版就有，不是家族慣例）、lock 程式用 tty 反查 session（鎖定中 list-clients 是空的，display-message -c 回錯的 session）、
custom 的 VT 終端機模擬器路線（多一個依賴、忠實度與效能都要驗；改直通）、custom 程式的自動重啟（不干涉生命週期）、黑畫面加紅字（用板子）、結束的字叫 COMPLETED / ERROR 再叫 DONE / ERROR（改直接寫退出碼）、PIN 框下面凍住畫面加 locku 的底色（框下面沒有動畫）、框收起時真的縮一欄再放回去逼程式重畫（改成不凍畫面就不用補畫）、對正在畫的程式送 SIGWINCH（cmatrix 會閃一下從頭來）、tmux 即時套用維護一份跟區塊平行的指令清單（改整塊 `source-file`）、鎖定畫面上的忘記密碼入口、恢復碼、清空 `pin_hash` 當 reset、`[1]` 上的全域 `P`（`p` 才是游標那列）、原生 Windows。

## 目錄

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
│   └── version/        版本字串
├── e2e/                pty 端到端：tmux_attach.py、custom_lock.py、screen_lock.py
└── docs/               function.md、ui.md、ux.md、icon.svg
```

## 開發

```bash
make check                                       # fmt-check + vet + go test -race
LOCKU_DUMP=1 go test ./internal/ui -run TestDump -v   # 印出各種尺寸的畫面
make lock                                        # 編譯並鎖住這個終端機
make e2e                                         # 端到端：真的 tmux（e2e/tmux_attach.py，自己的 TMUX_TMPDIR）、custom saver（e2e/custom_lock.py）、真的 screen（e2e/screen_lock.py，自己的 SCREENDIR）；不碰你的 server 與 session
```

TUI 行為全部用 programmatic model test 驗證（不需要 tty）；`make check` 帶 race detector（custom 的 pump 與 screen writer、login 的 su、custom 鎖的 prompt 各有 goroutine）。tmux / custom / screen 的驗收（`docs/function.md` §12）
以 python pty harness 跑真的 binary 完成：鎖定中 prefix+d、prefix+c、Ctrl+C 被吞、對 PIN 後 client 回來、鎖著的時候 attach 也被鎖、畫面上切 lock-session 時跑著的 server 整塊換掉、custom 的框疊在動畫上且程式被殺乾淨、screen 的 `C-a x` 與 `bind` 的鍵進 locku、`idle` 自動鎖、跑著的 session 即時收到設定、tty 關閉進程結束。

## 下一步

1. 8 小時 CPU / 記憶體觀察（§12 最後一項）。
2. README 英文版。
3. brew formula（vulcanshen/homebrew-tap）。
4. CHANGELOG 收 0.1.0、tag。
