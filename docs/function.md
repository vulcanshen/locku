# locku — 功能定義

> v1.0 定案，2026-09-24
> 本文件只談「做什麼、怎麼達成」。版面放 ui.md，按鍵與流程放 ux.md。

## 0. 定位

一句話：跑在真實終端機上的螢幕保護程式加密碼鎖。啟動後任何按鍵都不轉發，只會彈出密碼輸入，驗證通過才把終端機還回去。

### 0.1 是什麼、不是什麼

- 是 u-family 第五個成員（kbu / filu / sshu / webu / locku），同一套 Go + Bubble Tea 技術棧與 VTP 操作原則。
- 是 tmux 的 lock-command、screen 的 LOCKPRG，也可以在裸 tty（teletype，這裡泛指任何終端機裝置）直接執行。
- 不是安全邊界。同一使用者另開一條 SSH 就能 `tmux attach -d`、`kill -9`。定位是防路人、防誤觸、好看。
- 不是 pane 內的攔截器。原因見 0.2。

### 0.2 為什麼不能在 pane 內做

按鍵流向：實體終端機 → tmux client 進程 → tmux server 進程 → 該 pane 的 PTY（pseudo terminal，虛擬終端）→ pane 內的程式。prefix 在 tmux server 就被消化，那個 byte 永遠不會進到 pane。screen 的 Ctrl+a 同理。所以「在 pane 內攔 prefix」在架構上不可能，locku 必須站在 tmux/screen 之外，由它們把 tty 交出來。

### 0.3 與既有工具比較

| | vlock | lock -np（BSD） | screen 內建 | locku |
|---|---|---|---|---|
| 驗證 | PAM 系統密碼 | 系統密碼 | 系統密碼 | 自家 PIN |
| 畫面 | 純文字提示 | 純文字提示 | 純文字提示 | 螢幕保護動畫 + popup |
| VT 切換鎖 | 有，需 root | 無 | 無 | 不做 |
| tmux 整合 | 手動設 lock-command | tmux 預設值 | 不適用 | 文件提供設定 |
| macOS | 無 | 無 | 有 | 有 |

PAM 是 Pluggable Authentication Modules，Linux/macOS 的系統驗證框架。VT 是 virtual terminal，Linux 的實體主控台 Alt+F1 到 F7。

## 1. 執行模型

### 1.1 三種進入點

| 進入點 | 誰啟動 | tty 從哪來 | 結束後 |
|---|---|---|---|
| tmux lock-command | tmux client 進程用 `sh -c` 執行 | client 自己的真實 tty | client 送 MSG_UNLOCK，tmux 重繪 |
| screen LOCKPRG | screen 以使用者 uid/gid 直接 execl，不經 shell、不帶參數，argv[0] 為 SCREEN-LOCK | display 的 tty | screen 恢復接受命令鍵 |
| 裸 tty | 使用者在 shell 打 `locku lock` | shell 的 tty | 回到 shell |

三者對 locku 完全相同：拿到的 stdin/stdout 是一個 tty，全權擁有，直到進程結束。

tmux 的細節：server 呼叫前已自行切到 alternate screen 並清屏，client 進程停止讀 tty、以 `system()` 同步等待 locku 結束，期間不處理任何按鍵；結束後 tmux 自己重繪整個畫面。locku 不必替 tmux 保存或還原任何狀態。

### 1.2 契約

1. 進程活著等於鎖著，進程結束等於解鎖。tmux/screen 不看 exit code。
2. 因此任何錯誤都不得讓進程結束。panic 一律 recover 後回到保護畫面。
3. 只有三種情況可以結束：密碼驗證通過；無 PIN 模式下收到任何按鍵（4.3）；tty 已經消失（read 得到 EOF 或 EIO），此時已無東西可保護。
4. 執行期間不建立子進程、不開網路，只碰 tty 與唯讀的設定檔。

### 1.3 多 client

tmux `lock-session` 對每個 attach 中的 client 各跑一份 locku，彼此獨立，A 解鎖不影響 B。這是 tmux 的行為，locku 不需要知道其他實例存在。

## 2. 輸入與訊號

### 2.1 終端機模式

- raw mode：關 ICANON、ECHO、ISIG。ISIG 關掉後 Ctrl+C、Ctrl+Z、Ctrl+\ 變成普通 byte 進到程式，不再產生 SIGINT、SIGTSTP、SIGQUIT。
- alternate screen：避免捲動緩衝區露出鎖定前的內容。
- 不開 mouse tracking：滑鼠留給終端機本身做文字選取，無害。

### 2.2 訊號表

| 訊號 | 來源 | 處理 |
|---|---|---|
| SIGINT / SIGTERM / SIGQUIT | kill 預設、ISIG | 忽略 |
| SIGHUP | tty 掛斷 | 忽略，交由 read EOF 決定是否結束 |
| SIGTSTP / SIGTTIN / SIGTTOU | Ctrl+Z、背景讀寫 | 忽略 |
| SIGWINCH | 視窗大小改變 | 接收並重繪 |
| SIGKILL / SIGSTOP | kill -9 | 無法攔截，範圍外 |

Bubble Tea 實作注意：預設會安裝 SIGINT/SIGTERM handler 讓程式結束，必須用 `tea.WithoutSignalHandler()` 並自行 `signal.Ignore`。Ctrl+C 會以 KeyMsg 進來，當成一般按鍵處理即可。

### 2.3 攔不到的東西（範圍外，README 要說清楚）

- ssh client 端的 `~.`：在本機端處理，byte 不會過線。
- Linux VT 的 Alt+F1 到 F7 切換、SysRq：需要 VT_LOCKSWITCH ioctl 與 root，vlock 的領域，不做。
- 另一條 SSH：`tmux attach -d`、`tmux kill-server`、`kill -9`。
- 終端機模擬器自身的快捷鍵：iTerm2 分頁切換、Cmd+W 等。

## 3. 狀態機

```
        任何按鍵
saver ───────────▶ prompt ── Enter 且正確 ──▶ exit 0
  ▲                  │
  │  Esc             │  Enter 且錯誤：顯示錯誤、清空輸入、留在 prompt
  │  閒置 N 秒       │
  └──────────────────┘
```

無 PIN 模式（4.3）：saver ── 任何按鍵 ──▶ exit 0，沒有 prompt。

- saver：畫保護內容，吞掉所有按鍵。第一個按鍵只負責切到 prompt，不當作密碼輸入。
- prompt：密碼輸入 popup，輸入不回顯。
- 錯誤處理：每次錯誤固定 1 秒 debounce，連續錯誤鎖定可設定，見 4.4。
- 閒置回 saver：prompt 內連續 `prompt_timeout` 秒沒有任何按鍵就收起回 saver，每次按鍵重算，所以輸入到一半不會消失。收起時清空已輸入內容。預設 30，0 表示永不收起。

## 4. 驗證

### 4.1 選項

| 方式 | 做法 | 優點 | 代價 |
|---|---|---|---|
| A. 自家 PIN | 在 `locku` 設定 TUI 設定，bcrypt 存 config | 純 Go、零依賴、單檔 static binary | 與系統帳號無關 |
| B. PAM | cgo + libpam，pam_authenticate | 用系統密碼 | cgo、需 libpam-dev、macOS 走不同 PAM、無法 static build |
| C. 讀 /etc/shadow | crypt 比對 | 純 Go 可做 | 需 root 或 shadow 群組，macOS 沒有 shadow |

已決（2026-09-24）：v1 只做 A。理由：0.1 已定位為非安全邊界；B 會讓整個 binary 變 cgo，不能 static、跨平台要 C 工具鏈、brew 要拉 libpam，所有使用者一起付代價；C 需 root 或 shadow 群組，macOS 沒有 shadow。config 保留 `auth: pin` 欄位，將來要 B 再以 `auth: pam` 加入，格式不變。C 不做。

### 4.2 PIN 規格

- 長度 4 到 64 字元，任意可列印字元。
- bcrypt cost 10，存於 config，檔案權限 600。
- 修改：在 `locku` 設定 TUI 內操作，需先輸入舊 PIN。忘記則手動刪 config 重設，這在非安全邊界的定位下可接受。

### 4.3 未設定 PIN、config 缺失或損毀

已決（2026-09-24）：一律進入「無 PIN 模式」，畫面照常顯示 saver，任何按鍵直接結束（等同解鎖），沒有 prompt。locku 因此不設定也能當純螢幕保護程式用，PIN 是加購。

- config 不存在：用預設值（saver clock），無 PIN。
- config 存在但 `pin_hash` 為空或缺欄位：依 config 的 saver，無 PIN。
- config YAML 解析失敗或 `pin_hash` 不是合法 bcrypt：視同不存在，fail open。理由同 0.1，非安全邊界，鎖死的代價比誤放大。
- 無 PIN 模式的 saver 畫面必須有一行狀態文字標明「未設定 PIN」，避免使用者誤以為有鎖。解析失敗時該行改顯示錯誤原因。

### 4.4 錯誤 PIN 的節流

已決（2026-09-24）：兩層。

- debounce，固定不可設定：每次錯誤後 1 秒內顯示錯誤訊息並吞掉所有輸入，之後清空輸入回到可輸入。目的是明確的「錯了」回饋，且不能用連打 Enter 閃過訊息。
- 連續錯誤鎖定，config 設定，預設關閉：連續錯 `lockout_after` 次後進入冷卻 `lockout_seconds` 秒，期間 prompt 顯示剩餘秒數並吞掉所有輸入。`lockout_after: 0` 即關閉。計數只在進程內存活，Esc 回 saver 不重置，冷卻結束才歸零，成功解鎖進程即結束。

## 5. 螢幕保護內容

### 5.1 兩層：saver 出內容，畫布出畫法

已決（2026-09-24）：saver 只決定「顯示什麼」，畫布只有一種畫法。

- **saver** 是具名實例：`type` 決定它怎麼產生內容，參數決定內容細節。輸出永遠是幾行 ASCII 文字，不帶任何樣式。
- **畫布**把這幾行文字用 u-family splash 的像素風格畫出來，依終端機格數自動選縮放，見 5.3。saver 碰不到顏色、字形、位置。

### 5.2 saver 型別與實例

| type | 參數 | 內容 | tick |
|---|---|---|---|
| clock | `layout` row / column；`size` small / medium / large；`font` 3x7 / 3x5；`time` `HH MM` / `HH MM SS`；`date` off 或四選一；`bg` / `fg` 兩個顏色 | row：一列時間，date 不是 off 時第二列日期；column：依分隔符拆行，`HH` / `MM` / `SS`，日期再拆 `YYYY` / `MM` / `DD` | time 含秒為 1 秒，否則對齊整分每 60 秒 |

修訂（2026-09-24，第四輪）：`font` 新增，3x7 之外多一套 3x5（同樣直角、同樣 3 格寬，只有 5 列高），使用者要試；原本「第二套 3 × 5 字型」是在 5 × 7 時代否決的，那時它會是第二種畫法，現在字形已經是七段式，5 列只是把直線縮短，兩套並列讓使用者比，決定後留一套或都留。

修訂（2026-09-24，第三輪）：12 時制 AM/PM 拿掉，time 只剩兩種；時間不畫冒號，時、分、秒之間用一個 2 px 的空白隔開（組內間隔 1 px、組間 4 px，分組看得出來），日期的減號保留。選項名稱改為 `HH MM` / `HH MM SS`，舊寫法 `HH:MM`、`HH:MM:SS` 與 AM/PM 兩種讀到時自動對應。

修訂（2026-09-24，使用者實機試用後）：`layout` 新增，column 讓每行只有 2 到 4 個字，字因此大好幾倍；
`size` 新增，一個字型像素佔 1 × 1 / 2 × 2 / 3 × 3 格，預設 medium，塞不下怎麼退見 5.3；
點陣板的 `bg` / `fg` 從全域 style 搬進每個 saver，每個實例自己一組顏色，沒有全域顏色設定。

time 兩種：`HH MM`、`HH MM SS`（24 時制）。date 四種：`YYYY-MM-DD`、`YYYY-MMM-DD`、`MM-DD`、`MMM-DD`，MMM 是英文月份縮寫大寫（JAN 到 DEC）。時間與日期各自設定。

沒有自由輸入：所有內容由這兩個選項產生，字元集只有 0 到 9、冒號、減號、空白、大寫 A 到 Z，點陣字只畫這 39 個。

實例規則：

- name 唯一，是 config 裡 `saver` 指向的鍵。
- 預設一個實例 `clock`（type clock，layout row，size medium，time `HH MM`，date off，bg surface0 `#313244`，fg gold `#f2b753`）。config 缺 `savers` 時用它。
- 可 duplicate（複製參數、要求新 name）、rename（連動 `saver` 指向）、delete。啟用中的不可刪，最後一個不可刪。type 建立後不可改，要換 type 就 duplicate 另一個。
- v1 只有 clock 一個 type。type 欄位保留：新 type 只是多一個產內容的函式，不動畫布。使用者自由輸入的 text type 已移除（2026-09-24），內容不可控。

### 5.3 畫布渲染器

只有一種樣式：splash 的像素風格。一個像素 = Nerd Font 的方塊 glyph（nf-fa-square，``）加一個空格，佔 2 欄 1 列，在螢幕上接近正方形。字型是 3 × 7 或 3 × 5 點陣字（5.2 的 `font`）。

兩種單位（2026-09-24，使用者定案）：**顯示單元**是一個字型像素，size k 時佔 k × k 格；**間隔單元**是字與字、行與行之間的暗格，size 1 與 2 時 1 格、size 3 時 2 格——隨顯示單元變大，但不跟著等比放大，否則 large 大半是間隔。字與字之間 1 個間隔單元，行與行之間也是；時間裡的空白本身就是 1 個間隔單元（加兩側的字距，組與組之間隔 3 個）。

Nerd Font 必裝，與家族相同。字型在使用者本機的終端機模擬器，不在 server，經 SSH 不受影響。

縮放：

1. 一行的寬（格）= 各字寬相加、字與字之間 1 個間隔單元 g：數字與字母 3k 格（M、W 5k）、減號 3k、冒號 k、空白 g；k 是 size 的倍率（1 / 2 / 3），g 是間隔單元（1 / 1 / 2）。所以 `HH MM` = 12k + 5g、`HH MM SS` = 18k + 9g、`YYYY-MM-DD` = 30k + 9g：small 17 / 27 / 39，medium 29 / 45 / 69，large 46 / 72 / 108 格。行高 = h·k·m + (m − 1)·g，h 是字型高（3x7 是 7、3x5 是 5）。取最寬的一行。（修訂 2026-09-24：原本每個字一律 5 px、行距 2 px，`HH:MM` 29 px，large 要 174 欄，32 吋螢幕都吃不到；先把標點改成比例寬，再因字形全是直角而把數字壓成 3 px 並拿掉時間的冒號；最後把間隔從「跟著放大的字型像素」改成獨立的間隔單元，large 的 `HH MM SS` 從 178 欄降到 148 欄，152 欄的終端機終於吃得到。）

   各 size 需要的終端機（含 4 欄 / 3 列邊距與狀態列），欄 × 列，3x7 / 3x5：

   | 內容 | small | medium | large |
   |---|---|---|---|
   | `HH MM` 一行 | 38 × 10 / 8 | 62 × 17 / 13 | 96 × 24 / 18 |
   | `HH MM SS` 一行 | 58 × 10 / 8 | 94 × 17 / 13 | 148 × 24 / 18 |
   | `HH` / `MM` 直排 | 18 × 18 / 14 | 30 × 32 / 24 | 44 × 47 / 35 |
   | `HH` / `MM` / `SS` 直排 | 18 × 26 / 20 | 30 × 47 / 35 | 44 × 70 / 52 |
2. 可用區 = 終端機寬減 4 欄邊距，高減 1 列狀態列再減 2 列邊距。
3. 倍數 k 由 saver 的 `size` 決定：small 1、medium 2、large 3，每個字型像素放大成 k × k 格（修訂 2026-09-24：原本 k 是「塞得下的最大整數」，使用者要的是明確的大小選項，不是計算結果）。塞不下的順序：先照退階梯砍內容（下述），內容砍到底還塞不下才把 k 降一級再從完整內容試起；k = 1 也塞不下才把內容改用一般文字以 fg 色置中疊在板上，點陣板照鋪。所以 large 在寬終端機配 column 排版正好，在窄終端機會自己退成 medium 或 small，不會爆框。
4. 整個畫布（狀態列以外的所有列）都是像素格，像 LED 點陣板：每格一個 glyph 加空格，沒亮的用該 saver 的 bg（預設 surface0 #313244），亮的用它的 fg（預設 gold #f2b753）。內容置中。終端機寬為奇數時最右一欄留白。splash 的名字、版本、開發者不出現，只取 glyph 畫法。

排版與退階（2026-09-24 第五版，使用者定案）：**時間與日期是兩個獨立的區塊，各自排、各自退，互不侵犯。**

- 時間先排，拿整個畫布；日期排在剩下的空間。row 是日期在時間**下面**（剩下的高度）；column 是日期在時間**左邊**、時間在右邊（剩下的寬度），兩欄各自垂直置中。區塊之間的距離以時間的間隔單元計：上下 2 個、左右 6 個（兩欄的溝要比組間的空白寬，日期才不會讀成時間的一部分）。
- 每個區塊的順序是**先降 size、再去單位**：完整內容從設定的 size 一路試到 small，都塞不下才去掉一個單位再從設定的 size 試起。時間的單位是秒，日期的單位是年。
- 日期怎麼都塞不下就整個不畫；時間怎麼都塞不下才退成一般文字。
- 所以 120 欄（58 格）medium 的 `HH MM SS` + `YYYY-MM-DD`：時間 medium（45 格）、日期 medium 要 69 格塞不下 → 日期降成 small 帶年（39 格），時間不動；152 欄（74 格）兩個都是 medium。
- config 不改，視窗變大就回來。

修訂史：原本是全部內容一條單向鏈「去年 → 去秒 → 去日期」、size 最後降；高度不夠時會白白丟掉秒，改成候選清單；再改成現在的兩區塊獨立、size 先於單位。

resize 重算 k 整張重畫。動畫：第一幀直接出現不動畫；之後內容變更（clock tick）只對有變的像素做 splash 式 shuffle 揭露，沒變的不動，一次變更 ≤ 400 ms。CPU 預算不變：閒置 < 1%。

### 5.4 狀態列

已決（2026-09-24）：所有 saver 共用一行狀態列，內容 `user@hostname · 鎖定於 HH:MM`，user 是啟動 `locku lock` 的使用者。預設顯示，config `show_status: false` 可關。管多台 server 時靠它分辨機器與帳號，回來時知道離開多久。

4.3 的「未設定 PIN」提示與解析錯誤原因也在這一列，但不受 show_status 影響，永遠顯示。

## 6. CLI

| 指令 | 作用 |
|---|---|
| `locku` | 開啟設定 TUI，見 6.1。不會鎖 |
| `locku lock` | 鎖住當前 tty。tmux、screen、裸 tty 都是叫這個 |
| `locku setup [tmux\|screen]` | 直接把整合設定寫進 tmux / screen 的設定檔，見 6.2；不帶參數兩個都做 |
| `locku version` | 版本 |

argv[0] 為 `SCREEN-LOCK` 時視同 `locku lock`。原因：screen 的 LOCKPRG 由 screen 直接 execl，不經 shell、不能帶參數，argv[0] 固定為 SCREEN-LOCK（macOS /usr/bin/screen 二進位內可見此字串）。這是 screen 唯一能把「要鎖」這個意圖傳給 locku 的通道。execl 也不搜 PATH，所以 LOCKPRG 必須是絕對路徑。

### 6.1 設定 TUI 的職責

裸指令 `locku` 開一個 TUI，只做設定：

- 設定或更改 PIN：輸入兩次確認，已有 PIN 時先驗舊的。
- 清除 PIN：回到無 PIN 模式，需先驗舊的。
- saver 實例管理：duplicate、rename、delete、編輯參數（5.2）：layout、time、date，以及 bg / fg 兩個顏色，各以 R G B 三個 slider 設定（webu slider 作法，數字清單不打字），config 存 hex。顏色走草稿：滑桿改的是草稿，`S` 才寫檔、`R` 丟掉草稿，其餘欄位立即寫檔（修訂 2026-09-24：使用者調歪過一次調不回來）。
- preference：啟用中的 saver（`saver`）、show_status、prompt_timeout、lockout 兩個值。設為啟用在這裡，側欄的 `●` 只顯示。
- 試鎖：從 TUI 直接進入 `locku lock` 的流程，解鎖後回到 TUI；全域 `P` 看啟用中的 saver，側欄 saver 上的 `p` 看那一個，兩者都帶著顏色草稿。
- 寫出 `~/.config/locku/config.yaml`，權限 600。

TUI 的版面與按鍵放 ui.md / ux.md。

### 6.2 `locku setup` 怎麼寫

只寫受管區塊，區塊外一個字都不動；重跑就是替換區塊，冪等。不帶參數兩個都做。

| 目標 | 檔案 | 區塊內容 |
|---|---|---|
| tmux | `~/.tmux.conf`；它不存在而 `~/.config/tmux/tmux.conf` 存在就用後者；都沒有就建 `~/.tmux.conf` | `set -g lock-command "locku lock"`、`set -g lock-after-time 300`、`bind L lock-session` |
| screen | `~/.screenrc`，加上 shell rc：`$SHELL` 是 zsh 寫 `~/.zshrc`、bash 寫 `~/.bashrc`、fish 寫 `~/.config/fish/config.fish` | `.screenrc`：`idle 300 lockscreen`；shell rc：`export LOCKPRG=<絕對路徑>`（fish 是 `set -gx LOCKPRG <絕對路徑>`） |

區塊標記：

```
# >>> locku >>>
...
# <<< locku <<<
```

- tmux 有 server 在跑時同時即時套用：`tmux set -g lock-command "locku lock"`、`tmux set -g lock-after-time 300`、`tmux bind L lock-session`。沒有 tmux 或沒有 server 就跳過並說明。
- screen 的 LOCKPRG 只能走 shell 環境（實測 2026-09-24，macOS screen 4.00.03，以探針程式經 pty 驗證）。原本想走 `.screenrc` 的 `setenv LOCKPRG` 一個檔搞定，實測不通：按 `C-a x` 出現的是 screen 內建的 `Key:` 鎖，探針沒被呼叫。原因是 `lockscreen` 由 attacher（接著終端機的前端進程）呼叫 `getenv`，而 `.screenrc` 只有後端讀、`setenv` 改的是後端與視窗內 shell 的環境；attacher 的環境在 `screen` 或 `screen -r` 執行那一刻就固定了。環境變數路線則完全符合設計：LOCKPRG 被 execl、`argv[0]` 是 `SCREEN-LOCK`、stdin 是 tty。所以 setup 寫 shell rc 的受管區塊，並提示：新開 shell 才有這個變數；已在跑的 session 不必重啟，detach 後從新 shell `screen -r` 即可，因為 attacher 是新進程。
- 絕對路徑偏好 PATH 上找到的那個（通常是 brew 的 symlink），不用解析 symlink 後的 Cellar 路徑，升級版本後才不會失效。
- 執行後印出改了哪個檔、有沒有即時套用、還需要做什麼（screen：新開 shell；已在跑的 session detach 後從新 shell 重新 attach）。
- 不備份、不刪除、沒有 unsetup：要移除就手動刪區塊。
## 7. 設定與儲存

只有一個檔：`~/.config/locku/config.yaml`

```yaml
auth: pin              # v1 只有 pin，保留給 pam 擴充
pin_hash: "$2a$10$..."   # 空或缺欄位 = 未設定 PIN，見 4.3
saver: clock           # 啟用的 saver name，必須存在於 savers
savers:
  - name: clock
    type: clock
    layout: row           # row / column（依分隔符拆行）
    size: medium          # small / medium / large：一個字型像素佔 1 / 2 / 3 格見方
    font: 3x7             # 3x7 / 3x5：字型高 7 列或 5 列，都是 3 格寬
    time: "HH MM"          # HH MM / HH MM SS（時分秒以空白分組，不畫冒號）
    date: off             # off / YYYY-MM-DD / YYYY-MMM-DD / MM-DD / MMM-DD
    bg: "#313244"          # 這個 saver 的點陣板暗格，預設 surface0
    fg: "#f2b753"          # 亮格，預設 splash gold
show_status: true      # 狀態列 user@hostname · 鎖定於 HH:MM，見 5.4
prompt_timeout: 30     # prompt 連續幾秒無按鍵就收起，每次按鍵重算，0 = 永不收起
lockout_after: 0       # 連續錯幾次進冷卻，0 = 關閉，見 4.4
lockout_seconds: 30    # 冷卻秒數
```

修訂（2026-09-24）：頂層 `style` 拿掉，顏色是每個 saver 自己的 `bg` / `fg`。

- 無 history、無 cache、無 session。
- config 是 saver 的唯一來源，命令列不提供覆蓋。
- `saver` 指向不存在的 name、或 `savers` 為空：用內建預設 clock，狀態列顯示 config error，不算損毀。
- 讀取失敗的處理見 4.3。
- 閒置多久自動鎖是 tmux 的 lock-after-time、screen 的 idle，不是 locku 的設定。

## 8. 安裝與整合（README 要交付的內容）

順序不限：不設定就是純螢幕保護，任何鍵解鎖；要密碼再執行 `locku` 設 PIN。

`locku setup tmux` / `locku setup screen` / `locku setup` 直接寫進設定檔，做法見 6.2。以下是它寫的內容，手動設定也是同一份：

tmux，寫進 `~/.tmux.conf`：

```
set -g lock-command "locku lock"
set -g lock-after-time 300
bind L lock-session
```

screen，`~/.screenrc`：

```
idle 300 lockscreen
```

加上 shell rc（`~/.zshrc` 或 `~/.bashrc`；fish 用 `set -gx`），因為只有 attacher 的環境會被 lock 讀到（6.2）：

```
export LOCKPRG=/usr/local/bin/locku   # 絕對路徑，不能帶參數
```

裸 tty：直接執行 `locku lock`。

分發：vulcanshen/homebrew-tap formula；GitHub release 附 static binary（linux amd64 / arm64、darwin arm64）。

## 9. 技術選型

- Go，Bubble Tea + Lipgloss，與 kbu / filu / sshu / webu 同棧。
- `golang.org/x/crypto/bcrypt`。
- 無 cgo。
- 測試：狀態機與驗證邏輯單元測試；tmux / screen 整合走手動 checklist（第 12 節）。

## 10. 決定清單（2026-09-24）

1. 名稱 locku。
2. 進入點是 tmux lock-command、screen LOCKPRG、裸 tty。不在 pane 內攔截。
3. 非安全邊界，定位為螢幕保護與防誤觸。
4. 進程結束等於解鎖，任何錯誤不得導致進程結束。
5. 不做 VT 切換鎖。
6. 不開 mouse tracking。
7. 自動鎖定的閒置計時交給 tmux / screen，locku 不自己計時。
8. 裸指令 `locku` 開設定 TUI，`locku lock` 才鎖定，與 kbu / filu / sshu 裸指令即 TUI 的慣例一致。screen 經 argv[0] SCREEN-LOCK 辨識。
9. 驗證 v1 只做自家 PIN，PAM 留 `auth: pam` 擴充位，shadow 不做。
10. 未設定 PIN 或 config 缺失、損毀時進入無 PIN 模式：照常顯示 saver，任何按鍵解鎖，畫面標明未設定 PIN。fail open。
11. 錯誤 PIN 節流兩層：固定 1 秒 debounce；連續錯誤鎖定由 config 的 lockout_after / lockout_seconds 控制，預設 0 關閉。
12. saver 分 type 與具名實例：v1 type 只有 clock，參數 layout row / column、size small / medium / large、font 3x7 / 3x5、time `HH MM` / `HH MM SS`（24 時制，2026-09-24 拿掉 AM/PM）、date off 或四選一、bg / fg 兩色，沒有自由輸入；預設實例 clock；可 duplicate / rename / delete，啟用中與最後一個不可刪。
13. 狀態列 user@hostname 與鎖定時間預設顯示，show_status 可關；未設定 PIN 提示不可關。
14. prompt_timeout 預設 30 秒，以最後一次按鍵起算，0 為永不收起。
15. 畫布只有一種樣式：整面 LED 點陣板，暗格 saver 的 bg、亮格它的 fg，點陣字依 saver 的 size 放大 1 / 2 / 3 倍；間隔是獨立的單元（size 1、2 是 1 格，3 是 2 格），隨顯示單元變大但不等比放大（2026-09-24 修訂，原本間隔跟著字型像素放大，large 大半是間隔）；退階見 20，1 倍也塞不下退化為一般文字疊在板上。saver 決定內容、大小與顏色。
16. 內容全由固定選項產生；字元集 39 個（數字、冒號、減號、空白、大寫字母）。字形一律直角、沒有斜線，像七段顯示器：0 沒有中間斜線、7 沒有勾、S / O / I 與 5 / 0 / 1 同形；也因此數字壓成 3 格寬，字母也是（M、W 5 格），高 7 列或 5 列兩套字型（`font`），標點比例寬（減號 3、冒號 1，空白是 1 個間隔單元，時間不再用冒號）；沒有直角寫法的字母取方塊字型的畫法（N 是 Π、V 是底部收尖的 U；3x5 的 B 與 8 同形）。2026-09-24 修訂。
17. Nerd Font 必裝，與家族相同；字型在使用者本機終端機，SSH 不影響。
18. 第一幀不動畫；之後內容變更只對有變的像素做 splash 式 shuffle 揭露。
19. 顏色是每個 saver 自己的 bg / fg（修訂 2026-09-24，原為全域 Settings › style），bg 預設 surface0、fg 預設 gold；以 RGB slider 設定、config 存 hex；滑桿改草稿，`S` 存、`R` 丟，`q` 遇到未存草稿先問。
20. 時間與日期是兩個獨立區塊，各自排版、各自退階：時間先拿整個畫布，日期拿剩下的（row 在下、column 在左）；每個區塊先降 size 再去單位（時間去秒、日期去年）；日期塞不下就不畫，時間塞不下才一般文字；config 不改。（2026-09-24 修訂三次，最後由使用者定案。）
21. 整合設定是 CLI：`locku setup [tmux|screen]` 直接寫入設定檔的受管區塊，冪等；tmux 有 server 時即時套用；不做 TUI popup。
22. （2026-09-24 修訂）側欄 Enter 一律把焦點送到 `[2]`，包括 saver；設為啟用在 preference › saver，側欄的 `●` 只顯示。Settings 只有 preference 一項，原 config 改名 preference、style 取消。
23. （2026-09-24 修訂）側欄 saver 的 item operation：`[Enter] Edit`、`[p] Preview`（預覽那一個 saver）、`[D]uplicate`、`[r]ename`、`[X] Delete`，D / X 大寫對齊 sshu。
24. （2026-09-24 修訂）`[2]` 在 saver 上的 panel operation：`[P] Preview`（預覽正在編輯的這個 saver，帶草稿）、`[S] Save`、`[R] Reset`。全域 `P` 在 saver 的 `[2]` 上就是這個 saver，其他地方是啟用中的。

## 11. 待決清單

無。2026-09-24 全部定案。

## 12. MVP 驗收

- tmux 內 `lock-session` 後：prefix+d、prefix+&、prefix+c、prefix+x 皆無反應。
- Ctrl+C、Ctrl+Z、Ctrl+\ 無反應。
- 錯誤 PIN 留在 prompt；正確 PIN 後 tmux 畫面完整恢復，pane 內程式狀態未變。
- 錯誤 PIN 後 1 秒內的輸入被吞掉。lockout_after 設 3 時，第 3 次錯誤後顯示倒數，倒數期間輸入無效，結束後可再試。
- 鎖定中 resize 視窗，畫面重繪不破。
- screen 內 `lockscreen` 同上。
- 裸 ssh 內執行同上。
- clock saver 連續執行 8 小時，CPU 平均 < 1%，記憶體不成長。
- 渲染器在 80×24、120×40、200×60 下對 `HH MM` 各選到預期的 k（large：2 / 3 / 3）；152×39 medium 配 `HH MM SS` + `YYYY-MM-DD` 是時間 2 倍、日期 1 倍帶年；130×24 只畫時間；column 配日期是兩欄、日期左時間右；30×8 退化為一般文字。
- 內容變更只動有變的像素，以 shuffle 揭露，沒變的像素輸出不變。
- saver name 重複被擋。
- style 任一 channel 改動後立即寫檔，Preview 反映；手改 config 的非法 hex 視同預設。
- config 不存在時進入無 PIN 模式：saver 照常顯示、標明未設定 PIN、任何按鍵結束。
- 設定 TUI 寫出的 config 能被 `locku lock` 讀取。
- LOCKPRG 指向 locku 本體時，screen `lockscreen` 進入鎖定而不是設定 TUI。
- `locku setup tmux` 跑兩次，設定檔內容相同；有 server 時 `tmux show -g lock-command` 立即是 `locku lock`。
- shell 環境有 LOCKPRG 時 `C-a x` 進的是 locku（2026-09-24 以探針實測通過；`.screenrc` 的 `setenv` 路線實測不通，6.2 已改為 shell rc）。
- tmux lock-command 期間 prefix 到不了 tmux（2026-09-24 以探針實測：鎖定中送 prefix+d、prefix+c 都被鎖定程式吞掉，client 仍 attached；解鎖後 prefix+d 才 detach）。
