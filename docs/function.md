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

但 tmux 沒有「鎖著」這個狀態：`lock-server` / `lock-session` 只對那一刻 attach 著的 client 送 MSG_LOCK，之後 attach 進來的 client 什麼都不會發生（實測 2026-09-24，tmux 3.7c）。使用者要的是「只要進來、而它是鎖著的，就得看到保護程式」，所以 locku 補一個狀態：`locku lock` 啟動時把**全域** user option `@locked` 設成 1（每個 session 都看得到），`locku setup tmux` 寫的 `client-attached` / `client-session-changed` hook 看到 `@locked` 就對那個 client `lock-client`；PIN 對了（或無 PIN 模式任意鍵）結束前把 `@locked` 拿掉；tty 消失或被砍死不拿掉，下一個進來的人還是被鎖，直到有人輸入 PIN。第一個輸入正確 PIN 的 client 把標記清掉，其他還開著 locku 的 client 各自輸 PIN 才回來（使用者 2026-09-24：維持各自輸，不做輪詢）。

鎖的範圍是**整台 server**，不是 session（使用者 2026-09-24 定案）：螢幕保護程式保護的是「坐在這台終端機前的人看得到什麼」，鎖住一個 session 的話 `tmux attach -t 另一個` 就繞過去了，等於沒鎖；一份 config、一個受管區塊、一個 profile 對整台，也不用替每個 session 各配一套。所以 `locku` 這個 alias 是 `lock-server`。閒置鎖是 tmux 的極限——`lock-after-time` 是每個 session 各自計時——維持「哪個畫面閒置就鎖哪個畫面」，不升級成整台（雙螢幕各 attach 一個 session 時，正在打字的那邊不該被另一邊的閒置鎖到）。細節見 6.2。

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
- 閒置回 saver：prompt 內連續 `pin_prompt_timeout` 秒沒有任何按鍵就收起回 saver，每次按鍵重算，所以輸入到一半不會消失。收起時清空已輸入內容。預設 30，0 表示永不收起。（2026-09-24 改名，原 `prompt_timeout`；同日改名的還有 `lockout_after` → `wrong_pin_attempts`、`lockout_seconds` → `wrong_pin_attempt_cooldown`，三個都帶 lock 字看不出誰是誰；舊 key 讀進來自動轉。）

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
- 連續錯誤冷卻，config 設定，預設關閉：連續錯 `wrong_pin_attempts` 次後進入冷卻 `wrong_pin_attempt_cooldown` 秒，期間 prompt 顯示剩餘秒數並吞掉所有輸入。`wrong_pin_attempts: 0` 即關閉。計數只在進程內存活，Esc 回 saver 不重置，冷卻結束才歸零，成功解鎖進程即結束。

## 5. 螢幕保護內容

### 5.1 兩層：saver 出內容，畫布出畫法

已決（2026-09-24）：saver 只決定「顯示什麼」，畫布只有一種畫法。

- **saver** 是種類——class：clock、dino。它決定怎麼產生內容，輸出不帶任何樣式：clock 是幾行 ASCII 文字；dino 是一張自己像素座標的點陣圖（2026-09-24 加入第二種）。
- **profile** 是具名實例——object：一種 saver 加上它的參數與顏色，有名字；config 裡 `profile` 指向的、鎖定畫面顯示的，都是 profile（2026-09-24 定案，使用者以 OOP 分：class 不用取名、object 才有名字，能新增的是 profile、新增時先選 saver）。
- **畫布**把文字用 u-family splash 的像素風格畫出來、把點陣圖依 size 放大鋪滿，依終端機格數自動選縮放，見 5.3。saver 碰不到顏色、字形、位置。

### 5.2 saver 與 profile

| type | 參數 | 內容 | tick |
|---|---|---|---|
| clock | `layout` row / column；`size` small / medium / large；`font` 3x7 / 3x5；`time` `HH MM` / `HH MM SS`；`date` off 或四選一；`bg` / `fg` 兩個顏色 | row：一列時間，date 不是 off 時第二列日期；column：依分隔符拆行，`HH` / `MM` / `SS`，日期再拆 `YYYY` / `MM` / `DD` | time 含秒為 1 秒，否則對齊整分每 60 秒 |
| dino（2026-09-24） | `runner` 跑者：`trex`（暴龍）、`two-trex`（兩隻暴龍一前一後，前面小的 8 × 10、後面大的 12 × 14，各自跳各自的）；`scene` 場景：`grassland`（草原，障礙物是仙人掌）、`desert`（沙漠，障礙物是金字塔，沙地斑點較疏）；`bg` / `fg` 兩個顏色。沒有 size（使用者：dino 也沒有 size 的選項），畫布自己取塞得下的最大倍率 | Chrome 離線小恐龍遊戲當螢幕保護：地面與障礙物向左捲、跑者自己跳過去，無限循環沒有人玩、不會死。障礙物隨機（草原：仙人掌 1 / 2 / 3 株、高仙人掌；沙漠：金字塔小 / 中 / 大、小加中），間距隨機 44 到 100 px；兩隻跑者各自看自己前面的障礙物、各自在自己的視窗裡隨機起跳，後面那隻的步伐差半步；跳躍在「跳得過」的那段視窗裡隨機挑一幀起跳，前面沒東西時偶爾也無故跳一下；雲以三分之一速度飄。不記分、不畫時間，畫面上只有場景（使用者 2026-09-24：dino 上面不需要計算時間和分數）。場景像素：跑者 12 × 14、跳躍弧 16 幀最高 8 px、每幀走 2 px，最小場景 40 × 25 | 每 70 ms 一幀（14 fps），整張換、不做 reveal |

修訂（2026-09-24，第四輪）：`font` 新增，3x7 之外多一套 3x5（同樣直角、同樣 3 格寬，只有 5 列高），使用者要試；原本「第二套 3 × 5 字型」是在 5 × 7 時代否決的，那時它會是第二種畫法，現在字形已經是七段式，5 列只是把直線縮短，兩套並列讓使用者比，決定後留一套或都留。

修訂（2026-09-24，第三輪）：12 時制 AM/PM 拿掉，time 只剩兩種；時間不畫冒號，時、分、秒之間用一個 2 px 的空白隔開（組內間隔 1 px、組間 4 px，分組看得出來），日期的減號保留。選項名稱改為 `HH MM` / `HH MM SS`，舊寫法 `HH:MM`、`HH:MM:SS` 與 AM/PM 兩種讀到時自動對應。

修訂（2026-09-24，使用者實機試用後）：`layout` 新增，column 讓每行只有 2 到 4 個字，字因此大好幾倍；
`size` 新增，一個字型像素佔 1 × 1 / 2 × 2 / 3 × 3 格，預設 medium，塞不下怎麼退見 5.3；
點陣板的 `bg` / `fg` 從全域 style 搬進每個 saver，每個實例自己一組顏色，沒有全域顏色設定。

time 兩種：`HH MM`、`HH MM SS`（24 時制）。date 四種：`YYYY-MM-DD`、`YYYY-MMM-DD`、`MM-DD`、`MMM-DD`，MMM 是英文月份縮寫大寫（JAN 到 DEC）。時間與日期各自設定。

沒有自由輸入：所有內容由這兩個選項產生，字元集只有 0 到 9、冒號、減號、空白、大寫 A 到 Z，點陣字只畫這 39 個。

profile 規則：

- name 唯一，是 config 裡 `profile` 指向的鍵。
- 預設一個 profile `clock`，就是 clock saver 的預設值生的。config 缺 `profiles` 時用它。
- 新增（`[1]` 的 Savers 區塊在一種 saver 上按 `n`：要名字，提議 saver 自己的名字、用了就加號碼；以那種 saver 的預設值生出來）、duplicate（複製參數、要求新 name）、rename（連動 `profile` 指向）、delete。啟用中的不可刪，最後一個不可刪。
- profile 的 saver 建立後不可改：class 就是 class，要換就新增一個 profile（2026-09-24 定案；同一天曾短暫讓 type 可在 `[2]` 改，那是 dino 剛加進來、還沒有 New 時的權宜）。
- 兩種 saver：clock、dino。新 saver 只是多一個產內容的函式，不動畫布。使用者自由輸入的 text saver 已移除（2026-09-24），內容不可控。

saver 預設值（2026-09-24，使用者定案）：每種 saver 在 config 的 `savers` 有一組預設值，欄位跟它的 profile 一樣、只是沒有名字。它決定**之後**用這種 saver 新增的 profile 長什麼樣，改它不影響任何已存在的 profile；cursor 在 saver 上時 `[p]` 就用預設值跑一個臨時 profile 預覽。內建值（config 沒寫時）：

| saver | 預設值 |
|---|---|
| clock | layout row、size large、font 3x5、time `HH MM SS`、date `YYYY-MM-DD`、bg `#313244`、fg `#f2b753` |
| dino | runner trex、scene grassland、bg / fg 同上 |

`[2]` 在 saver 上除了說明還把預設值列出來，跟 profile 同一套列與操作（options popup、RGB slider 草稿、`S` / `R`）。

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

dino 的畫法（2026-09-24）：場景是整塊板，k 從 3 往下取第一個讓場景（40 × 25 px）塞得下的，都塞不下就 1；沒有 size 設定；場景 w × h = 板的格數 ÷ k，地面因此貼滿整寬，右邊 / 下面除不盡的格留暗。每幀整張換掉、不做 reveal——世界在移動，不是內容在變。14 fps 不是閒置，CPU 會比時鐘高，這是遊戲 saver 的代價。

### 5.4 狀態列

已決（2026-09-24）：所有 saver 共用一行狀態列，內容 `user@hostname · 鎖定於 HH:MM`，user 是啟動 `locku lock` 的使用者。預設顯示，config `show_status: false` 可關。管多台 server 時靠它分辨機器與帳號，回來時知道離開多久。

4.3 的「未設定 PIN」提示與解析錯誤原因也在這一列，但不受 show_status 影響，永遠顯示。

## 6. CLI

| 指令 | 作用 |
|---|---|
| `locku` | 開啟設定 TUI，見 6.1。不會鎖 |
| `locku lock` | 鎖住當前 tty。tmux、screen、裸 tty 都是叫這個 |
| （`locku` › Integration） | tmux / screen 的整合設定在設定 TUI 裡做：`[2]` 底部的 Install 寫進設定檔、Uninstall 拿掉，見 6.2（2026-09-25 取代 `locku setup [-d]` 指令） |
| `locku version` | 版本 |

argv[0] 為 `SCREEN-LOCK` 時視同 `locku lock`。原因：screen 的 LOCKPRG 由 screen 直接 execl，不經 shell、不能帶參數，argv[0] 固定為 SCREEN-LOCK（macOS /usr/bin/screen 二進位內可見此字串）。這是 screen 唯一能把「要鎖」這個意圖傳給 locku 的通道。execl 也不搜 PATH，所以 LOCKPRG 必須是絕對路徑。

### 6.1 設定 TUI 的職責

裸指令 `locku` 開一個 TUI，只做設定：

- 設定或更改 PIN：輸入兩次確認，已有 PIN 時先驗舊的。
- 清除 PIN：回到無 PIN 模式，需先驗舊的；驗過之後在 `New PIN` / `Remove PIN` 選單選 Remove，Enter 立即生效、不再 confirm（2026-09-24，原本是另一個 `x` 熱鍵加 confirm）。
- saver 預設值：每種 saver 的 `[2]` 列出它的預設值，可改，只影響之後新增的 profile（5.2）；`p` 用預設值預覽。
- profile 管理：new（從一種 saver）、duplicate、rename、delete、編輯參數（5.2）：layout、time、date……，以及 bg / fg 兩個顏色，各以 R G B 三個 slider 設定（webu slider 作法，數字清單不打字），config 存 hex。顏色走草稿：滑桿改的是草稿，`S` 才寫檔、`R` 丟掉草稿，其餘欄位立即寫檔（修訂 2026-09-24：使用者調歪過一次調不回來）。
- preference：啟用中的 profile（`profile`）、show_status、`pin_prompt_timeout`、`wrong_pin_attempts` / `wrong_pin_attempt_cooldown`。設為啟用在這裡，側欄的 `●` 只顯示。每一列的意思：focus 在 preference 的 `[2]` 時按 `?`，help 就**只有**這幾列的說明（自動換行），沒有鍵的清單；其他地方的 `?` 是鍵（2026-09-25：原本列在每列下面，使用者要搬到 help，而且 `[2]` 上只要 preference 的說明）。tmux / screen 的 `[2]` 同理，只有 conf、lock-after-time（screen 是 idle）、tmux 的 bind-key、Install 按鈕的說明。
- Integration（2026-09-25，取代 `locku setup` 指令）：側欄第三個區塊，`tmux` 與 `screen` 各一項，各自有 `conf`（要寫的檔案）與閒置幾秒自動鎖，後者**用工具自己的設定名稱**：tmux 是 `lock-after-time`、screen 是 `idle`（2026-09-25 修訂，使用者：tmux 就用 tmux 的設定名稱；同日早上兩個都叫 `idle_lock`，config key 一併改、舊 key 自動轉），各自獨立、不再共用一個值；預設 300，0 關閉；tmux 再多一列 `bind-key`（2026-09-25，使用者：prefix shortcut）：prefix 之後按哪個鍵就鎖整台，照 tmux 的寫法（`l`、`C-l`、`F12`），Enter 開 input 框；有值就在區塊多寫一行 `bind-key <鍵> lock-server`、有 server 在跑就即時 bind；清空就是不綁，那一行不寫、server 上原本綁的鍵 unbind；含空白或 `#` 拒收；screen 沒有這列。區塊在不在檔案裡是**狀態不是屬性**，不在列表裡，在 `[2]` 標題膠囊鏈的尾巴：`installed` / `uninstalled`，每次畫都讀檔——`[2]` 沒 focus 時三顆一起是 unfocus 的灰，focus 時 `installed` 綠、`uninstalled` 留灰（同日修訂，使用者）（2026-09-25 第三次修訂：早上是 dim 的 `tool` 名稱加 `block` `in the file` 兩列，使用者說看不懂、`[2]` 應該就是 property / value 兩欄；中午是一列 `status`）。`[2]` 底部一條分隔線下方、置中一顆**按鈕** `Install` / `Uninstall`，平常不亮、cursor 移到才亮（同日修訂，使用者：按鈕不是表格的列，要看得出 focus 在它上面）——Enter 跳 confirm 才執行：Install 把區塊寫進 conf（tmux 有 server 在跑就即時套用）、Uninstall 拿掉；`conf` 沒填時 disabled 並說明（2026-09-25 定案，取代同日早上的 `[S]` / `[X]` 熱鍵——畫面上看不到，使用者無法直觀知道）。**裝一次，之後隨設即得**：裝著時上面任何一列改動就直接重寫區塊、tmux 即時套到 server，conf 改路徑就把區塊從舊檔搬到新檔、清空就拿掉；沒裝就只寫 config.yaml，Install 仍由使用者按。兩個純方案各有硬傷才折衷：純隨設即得會在 conf 打錯時生檔、Remove 只能等於清 conf、不能先填好再裝；純按鈕會 stale——改了 lock-after-time 檔案裡還是舊值。`conf` 是 locku 唯二的自由輸入，用 webu 的 input 作法：框裡先 dim 顯示一個**提議**——目前值，沒有就是慣例的 `~/.tmux.conf` / `~/.screenrc`——Tab 接手編輯、Backspace 拒絕、打字就從頭打；Enter 照打的存，沒碰提議就 Enter 不改。
- 每個 `[2]`——profile、saver、tmux / screen、preference——第一列都是表頭 `Property` / `Value`，側欄區塊標題的 Blue，不可停（2026-09-25，使用者：所有 panel 2 都給標題列；同日改成單數 Property）。
- 試鎖：從 TUI 直接進入 `locku lock` 的流程，解鎖後回到 TUI；全域 `P` 看啟用中的 saver，側欄 saver 上的 `p` 看那一個，兩者都帶著顏色草稿。
- 寫出 `~/.config/locku/config.yaml`，權限 600。

TUI 的版面與按鍵放 ui.md / ux.md。

### 6.2 Integration 的 Setup / Remove 怎麼寫

（2026-09-25：從 CLI `locku setup [-d]` 搬進 TUI 的 Integration 區塊，同日從 `[S]` / `[X]` 熱鍵改成 `[2]` 底部的 Install / Uninstall 按鈕，Enter 後 confirm；下文的「setup」指寫入這個動作。）只寫受管區塊，區塊外一個字都不動；再寫一次就是替換區塊，冪等——裝著的時候每改一列就這樣重寫一次。

| 目標 | 檔案 | 區塊內容 |
|---|---|---|
| tmux | Integration › tmux 的 `conf`（使用者輸入，`~/` 可用，不存在就建；沒設按鈕就 disabled、說先填 conf，不猜） | `set -gF lock-command "<locku 的絕對路徑> lock -S '#{socket_path}'"`、`set -g lock-after-time <tmux 的 lock-after-time>`、`set -s "command-alias[90]" "locku=lock-server"`、`set-hook -g "client-attached[90]" "if -F \"#{@locked}\" lock-client"`、`set-hook -g "client-session-changed[90]" "if -F \"#{@locked}\" lock-client"`，Integration › tmux › bind-key 有填就再一行 `bind-key <鍵> lock-server`；每一行尾巴都有 `# locku` 註解 |
| screen | Integration › screen 的 `conf`（同上），加上 shell rc：`$SHELL` 是 zsh 寫 `~/.zshrc`、bash 寫 `~/.bashrc`、fish 寫 `~/.config/fish/config.fish` | `.screenrc`：`idle 300 lockscreen`；shell rc：`export LOCKPRG=<絕對路徑>`（fish 是 `set -gx LOCKPRG <絕對路徑>`） |

區塊標記：

```
# >>> locku >>>
...
# <<< locku <<<
```

- 路徑由使用者在 Integration 各項的 `conf` 輸入（2026-09-24 起在 preference，2026-09-25 搬來）：沒設時 Install / Uninstall 按鈕是 disabled 並說 `set conf first`，什麼都不寫（screen 連 shell rc 也不寫）；原本「`~/.tmux.conf` 不在就找 `~/.config/tmux/tmux.conf`」的猜法拿掉。相對路徑拒收，它會落在程式剛好執行的目錄。
- tmux 有 server 在跑時同時即時套用同樣五條（`tmux set -gF lock-command …` 等），有 bind-key 就多 `bind-key <鍵> lock-server`；Uninstall 時反向 `set -gu` / `set -su` / `set-hook -gu` 一條對一條拿掉、綁過的鍵 `unbind-key`、再清 `@locked`。Setup 時檔案裡原本綁的鍵跟這次不同（改了或清空），先 `unbind-key` 舊的再套用（2026-09-25）；綁過與否看的是檔案裡區塊的 `bind-key` 行，不另外記。unbind 之後那個鍵 tmux 內建的功能（例如 `l` 的 last-window）要 server 重啟才回來。沒有 tmux 或沒有 server 就跳過並說明。結果以 toast 一行回報（原本印在終端機的幾行，以 ` · ` 接起來）。
- tmux 那五行的道理（2026-09-24，使用者定案，全部以 pty 實測 tmux 3.7c）：
  - **預設不綁熱鍵**。用戶既然在用 tmux 就有自己一套 bind，`bind L` 會撞。改用 command alias：`prefix :` 然後打 `locku`，就是 `lock-server`（整台的 client 全鎖）；shell 裡 `tmux locku` 也一樣。要熱鍵的自己在 Integration › tmux › `bind-key` 填一個（2026-09-25，使用者：prefix shortcut），寫成 `bind-key <鍵> lock-server`——鍵是使用者選的，撞不撞他自己知道。
  - **每行尾巴 `# locku`**，加上受管區塊的頭尾標記，手動要移也認得出來；alias 與 hook 放在陣列的 90 號，不碰使用者自己的 0 號。
  - **lock-command 寫絕對路徑**：tmux client 是用它自己的 shell 環境跑 `sh -c`，`locku` 不一定在那個 PATH 上——找不到就是「畫面閃一下」（使用者 2026-09-24 實際踩到）。setup 寫的是 `Binary()`：PATH 上的 locku（brew 的 symlink）優先，否則就是執行 setup 的這個檔，跟 screen 的 LOCKPRG 同一套。所以從 repo 跑 `./locku setup tmux` 寫的就是 repo 那個 binary。
  - **lock-command 帶 socket**：lock-command 在 client 進程裡以 `system()` 跑，環境裡沒有 `TMUX`，被鎖的 client 也不在 `list-clients` 裡；`set -gF` 在讀檔時把 `#{socket_path}` 展開進去，`locku lock -S <socket>` 才知道要跟哪個 server 講話（`-L` 開的 server 也對）。標記是全域的，locku 不必反查自己在哪個 session（改成 server 等級前曾用 `tty` 反查，已不需要）。
  - **`@locked` 由 locku 設與清，全域**：解鎖沒有 hook（3.7c 的 MSG_UNLOCK 只清 flag），所以 `locku lock` 啟動時 `set -g @locked 1`、正常解鎖結束前 `set -gu @locked`，tty 消失不清；沒帶 `-S` 就什麼都不做。兩個 hook 看到 `@locked` 就 `lock-client`（attach 任何 session 與 switch-client 都驗過會觸發）。`setup -d` 順手 `set -gu @locked`。
  - 這些 tmux 呼叫都有 2 秒 timeout、失敗一律靜默：不可能讓鎖起不來或掉下來。裸 tty、screen、沒有 tmux 時 `Session()` 回空字串，什麼都不做。
- screen 的 LOCKPRG 只能走 shell 環境（實測 2026-09-24，macOS screen 4.00.03，以探針程式經 pty 驗證）。原本想走 `.screenrc` 的 `setenv LOCKPRG` 一個檔搞定，實測不通：按 `C-a x` 出現的是 screen 內建的 `Key:` 鎖，探針沒被呼叫。原因是 `lockscreen` 由 attacher（接著終端機的前端進程）呼叫 `getenv`，而 `.screenrc` 只有後端讀、`setenv` 改的是後端與視窗內 shell 的環境；attacher 的環境在 `screen` 或 `screen -r` 執行那一刻就固定了。環境變數路線則完全符合設計：LOCKPRG 被 execl、`argv[0]` 是 `SCREEN-LOCK`、stdin 是 tty。所以 setup 寫 shell rc 的受管區塊，並提示：新開 shell 才有這個變數；已在跑的 session 不必重啟，detach 後從新 shell `screen -r` 即可，因為 attacher 是新進程。
- 絕對路徑偏好 PATH 上找到的那個（通常是 brew 的 symlink），不用解析 symlink 後的 Cellar 路徑，升級版本後才不會失效。
- 執行後印出改了哪個檔、有沒有即時套用、還需要做什麼（screen：新開 shell；已在跑的 session detach 後從新 shell 重新 attach）。
- 不備份。移除用 `[2]` 的 Uninstall 按鈕（confirm 後；2026-09-25 早上是 `[X]` 熱鍵，前一天是 `locku setup -d`）：把受管區塊從檔案拿掉、有 server 在跑就一併拿掉；區塊前面補的空行也一起拿掉，其餘一個字不動；沒有區塊就說沒有；檔案不存在不會生出來。screen 的 Remove 同時清 `.screenrc` 與 shell rc 的區塊。`[2]` 標題尾巴的膠囊每次畫都讀一次檔案，說區塊在不在。
## 7. 設定與儲存

只有一個檔：`~/.config/locku/config.yaml`

```yaml
auth: pin              # v1 只有 pin，保留給 pam 擴充
pin_hash: "$2a$10$..."   # 空或缺欄位 = 未設定 PIN，見 4.3
profile: clock         # 啟用的 profile name，必須存在於 profiles
profiles:
  - name: clock
    saver: clock          # clock / dino：這個 profile 是哪一種 saver，建立後不改
    layout: row           # row / column（依分隔符拆行）
    size: medium          # small / medium / large：一個字型像素佔 1 / 2 / 3 格見方
    font: 3x7             # 3x7 / 3x5：字型高 7 列或 5 列，都是 3 格寬
    time: "HH MM"          # HH MM / HH MM SS（時分秒以空白分組，不畫冒號）
    date: off             # off / YYYY-MM-DD / YYYY-MMM-DD / MM-DD / MMM-DD
    bg: "#313244"          # 這個 profile 的點陣板暗格，預設 surface0
    fg: "#f2b753"          # 亮格，預設 splash gold
  - name: dino
    saver: dino           # dino 沒有 layout / size / font / time / date：畫布自己取最大倍率
    runner: trex          # dino 才有：跑者，trex / two-trex
    scene: grassland      # dino 才有：場景，grassland / desert
    bg: "#313244"
    fg: "#f2b753"
savers:                # 每種 saver 的預設值：之後新增的 profile 長這樣，改它不動既有的 profile（2026-09-24）
  clock:
    saver: clock
    layout: row
    size: large
    font: 3x5
    time: "HH MM SS"
    date: YYYY-MM-DD
    bg: "#313244"
    fg: "#f2b753"
  dino:
    saver: dino
    runner: trex
    scene: grassland
    bg: "#313244"
    fg: "#f2b753"
show_status: true      # 狀態列 user@hostname · 鎖定於 HH:MM，見 5.4
pin_prompt_timeout: 30          # PIN 框連續幾秒無按鍵就收起，每次按鍵重算，0 = 永不收起
wrong_pin_attempts: 0           # 連續輸錯幾次進冷卻，0 = 關閉，見 4.4
wrong_pin_attempt_cooldown: 30  # 冷卻秒數
tmux:                           # Integration › tmux（2026-09-25：從頂層 tmux_conf / idle_lock 搬來，舊 key 自動轉）
  conf: "~/.tmux.conf"          # Install 寫的檔；空 = 未設定，按鈕不能按
  lock-after-time: 300          # 閒置幾秒自動鎖，用 tmux 自己的名字（同日改名，早上的 idle_lock 自動轉）；0 = 不自動鎖
  bind-key: ""                  # prefix 之後鎖整台的鍵，照 tmux 寫法（l、C-l、F12）；空 = 不綁（2026-09-25）
screen:                         # Integration › screen，各自一份，不跟 tmux 共用
  conf: "~/.screenrc"           # Install 寫的檔；shell rc 另由 $SHELL 決定
  idle: 300                     # 同上，用 screen 自己的名字
```

修訂（2026-09-24）：頂層 `style` 拿掉，顏色是每個 profile 自己的 `bg` / `fg`。同日 key 改名：`saver` → `profile`、`savers` → `profiles`、每個 profile 的 `type` → `saver`（saver 是 class、profile 是 object）；舊 key 讀進來自動轉（讀檔先解析成樹、改名再 decode），下一次寫檔就只剩新 key。`savers` 這個 key 隨後給了 saver 預設值：它是清單就是舊的 profiles，是對照表就是預設值，兩種寫法都認。

- 無 history、無 cache、無 session。
- config 是 profile 的唯一來源，命令列不提供覆蓋。
- `profile` 指向不存在的 name、或 `profiles` 為空：用內建預設 clock，狀態列顯示 config error，不算損毀。
- 讀取失敗的處理見 4.3。
- 閒置多久自動鎖由各工具自己的那一列決定——tmux 的 `lock-after-time`、screen 的 `idle`，名字就是工具自己的（2026-09-25 拆開並改名，原本一個共用的 `idle_lock`；更早是寫死 300 在區塊裡），Install 原樣填進去，裝著時一改就重寫；locku 自己不計時。

## 8. 安裝與整合（README 要交付的內容）

順序不限：不設定就是純螢幕保護，任何鍵解鎖；要密碼再執行 `locku` 設 PIN。

在 `locku` 側欄 Integration › tmux 的 `[2]` 底部按 Install（confirm 後）直接寫進設定檔，做法見 6.2；裝了以後改任何一列就直接重寫。以下是它寫的內容，手動設定也是同一份：

tmux，寫進 Integration › tmux › conf（慣例 `~/.tmux.conf`）：

```
# >>> locku >>>
set -gF lock-command "/opt/homebrew/bin/locku lock -S '#{socket_path}'"  # locku
set -g lock-after-time 300                                                  # locku: 0 never
set -s "command-alias[90]" "locku=lock-server"                              # locku: prefix : locku locks every client
set-hook -g "client-attached[90]" "if -F \"#{@locked}\" lock-client"        # locku: attaching while locked locks the client
set-hook -g "client-session-changed[90]" "if -F \"#{@locked}\" lock-client" # locku: so does switching sessions
bind-key l lock-server                                                      # locku: prefix l locks every client
# <<< locku <<<
```

最後那行只在 Integration › tmux › bind-key 有填時才寫（這裡填的是 `l`）。鎖：`prefix :` 打 `locku`（或 shell 的 `tmux locku`，填了 bind-key 就 `prefix l`）整台的 client 全鎖，閒置 300 秒的畫面也鎖；鎖著的時候誰 attach 哪個 session 都會看到保護程式。移除：同一列變成 Uninstall。

screen，寫進 Integration › screen › conf（慣例 `~/.screenrc`）：

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
11. 錯誤 PIN 節流兩層：固定 1 秒 debounce；連續錯誤冷卻由 config 的 `wrong_pin_attempts` / `wrong_pin_attempt_cooldown` 控制（2026-09-24 改名，原 lockout_after / lockout_seconds），預設 0 關閉。
12. saver 是 class（clock、dino），profile 是有名字的 object（2026-09-24 定案，見 27）：clock 的參數 layout row / column、size small / medium / large、font 3x7 / 3x5、time `HH MM` / `HH MM SS`（24 時制，2026-09-24 拿掉 AM/PM）、date off 或四選一、bg / fg 兩色，沒有自由輸入；預設 profile clock；可 new / duplicate / rename / delete，啟用中與最後一個不可刪；profile 的 saver 建立後不改。
13. 狀態列 user@hostname 與鎖定時間預設顯示，show_status 可關；未設定 PIN 提示不可關。
14. `pin_prompt_timeout`（原 prompt_timeout）預設 30 秒，以最後一次按鍵起算，0 為永不收起。
15. 畫布只有一種樣式：整面 LED 點陣板，暗格 saver 的 bg、亮格它的 fg，點陣字依 saver 的 size 放大 1 / 2 / 3 倍；間隔是獨立的單元（size 1、2 是 1 格，3 是 2 格），隨顯示單元變大但不等比放大（2026-09-24 修訂，原本間隔跟著字型像素放大，large 大半是間隔）；退階見 20，1 倍也塞不下退化為一般文字疊在板上。saver 決定內容、大小與顏色。
16. 內容全由固定選項產生；字元集 39 個（數字、冒號、減號、空白、大寫字母）。字形一律直角、沒有斜線，像七段顯示器：0 沒有中間斜線、7 沒有勾、S / O / I 與 5 / 0 / 1 同形；也因此數字壓成 3 格寬，字母也是（M、W 5 格），高 7 列或 5 列兩套字型（`font`），標點比例寬（減號 3、冒號 1，空白是 1 個間隔單元，時間不再用冒號）；沒有直角寫法的字母取方塊字型的畫法（N 是 Π、V 是底部收尖的 U；3x5 的 B 與 8 同形）。2026-09-24 修訂。
17. Nerd Font 必裝，與家族相同；字型在使用者本機終端機，SSH 不影響。
18. 第一幀不動畫；之後內容變更只對有變的像素做 splash 式 shuffle 揭露。
19. 顏色是每個 saver 自己的 bg / fg（修訂 2026-09-24，原為全域 Settings › style），bg 預設 surface0、fg 預設 gold；以 RGB slider 設定、config 存 hex；滑桿改草稿，`S` 存、`R` 丟，`q` 遇到未存草稿先問。
20. 時間與日期是兩個獨立區塊，各自排版、各自退階：時間先拿整個畫布，日期拿剩下的（row 在下、column 在左）；每個區塊先降 size 再去單位（時間去秒、日期去年）；日期塞不下就不畫，時間塞不下才一般文字；config 不改。（2026-09-24 修訂三次，最後由使用者定案。）
21. 整合設定寫入設定檔的受管區塊，冪等；tmux 有 server 時即時套用。（2026-09-24 修訂）要寫的檔案由使用者輸入，沒設就報錯，不猜路徑；每一行尾巴 `# locku` 註解，手動也好移。（2026-09-25 修訂）原本是 CLI `locku setup [-d]`，改成 TUI 側欄 Integration 區塊的 `[S] Setup` / `[X] Remove`，指令拿掉；「不做 TUI popup」仍成立——它是 `[1]` 的一個區塊加 `[2]` 的列，不是 popup。（同日再修訂）熱鍵畫面上看不到，改成 `[2]` 底部的 Install / Uninstall 按鈕，Enter 後 confirm 才執行——confirm 是 message class，不是 Integration popup；見 33。
30. （2026-09-24，使用者定案）設定改名，三個都帶 lock 字看不出誰是誰：`prompt_timeout` → `pin_prompt_timeout`、`lockout_after` → `wrong_pin_attempts`、`lockout_seconds` → `wrong_pin_attempt_cooldown`；新增 `idle_lock`（閒置幾秒自動鎖，預設 300，0 關閉），一個值給所有拿 locku 當螢幕保護的工具：tmux 的 lock-after-time、screen 的 idle，setup 寫進去。舊 key 讀進來自動轉。（2026-09-25 再改：拆成各工具一份、用工具自己的名字，見 32。）
29. （2026-09-24，使用者定案）tmux 不綁熱鍵，改 command alias `locku`（`prefix :` 打 `locku`），不跟使用者既有的 bind 撞。（2026-09-25 修訂：預設仍不綁，但使用者可在 Integration › tmux › `bind-key` 自己填一個鍵，Setup 多寫一行 `bind-key <鍵> lock-server`、有 server 就即時 bind；空就不綁，見 32。）「鎖著的時候誰進來都被鎖」用全域 user option `@locked` 加 `client-attached` / `client-session-changed` hook 做到：`locku lock` 啟動時設、正常解鎖時清、tty 消失不清；lock-command 是 locku 的絕對路徑並以 `set -gF` 帶 `#{socket_path}` 給 `locku lock -S`。以 pty 端到端測試（`make e2e`）驗收。
31. （2026-09-24，使用者定案）鎖的範圍是**整台 tmux server**，不是 session：`locku` = `lock-server`，標記全域，attach 任何 session 都被鎖；螢幕保護程式保護的是整台，session 等級的鎖換個 session 就繞過。閒置鎖維持 tmux 的每 session 計時、不升級成整台；解鎖維持每個 client 各自輸 PIN、不輪詢。一份 config、一個區塊、一個 profile 對整台。
22. （2026-09-24 修訂）側欄 Enter 一律把焦點送到 `[2]`，包括 saver 與 profile；設為啟用在 preference › profile，側欄的 `●` 只顯示。Settings 只有 preference 一項，原 config 改名 preference、style 取消。
23. （2026-09-24 修訂）側欄 profile 的 item operation：`[Enter] Edit`、`[p] Preview`（預覽那一個 profile）、`[D]uplicate`、`[r]ename`、`[X] Delete`，D / X 大寫對齊 sshu；saver 的是 `[Enter] Edit`（看說明）、`[n] New`。
24. （2026-09-24 修訂）`[2]` 在 profile 上的 panel operation：`[P] Preview`（預覽正在編輯的這個 profile，帶草稿）、`[S] Save`、`[R] Reset`。全域 `P` 在 profile 的 `[2]` 上就是這個 profile，其他地方是啟用中的。
25. （2026-09-24）`tmux_conf` / `screen_conf`（2026-09-25 起是 Integration › tmux / screen 的 `conf`）是 locku 唯二的自由輸入，用 webu 的 input 作法：提議（目前值，沒有就是慣例路徑）dim 顯示，Tab 接手、Backspace 拒絕、Enter 照打的存、沒碰提議不改；只收絕對路徑或 `~/` 開頭。
26. （2026-09-24）第二種 saver `dino`：Chrome 小恐龍遊戲當螢幕保護，無限循環、隨機障礙、隨機跳躍、不會死、不記分；參數 `runner`（trex、two-trex：兩隻一前一後、前小後大、各自跳）、`scene`（grassland 仙人掌、desert 金字塔）、bg / fg，沒有 size（畫布取塞得下的最大倍率）。每 70 ms 一幀整張換，不做 reveal。
27. （2026-09-24，使用者定案）側欄分三個區塊，順序 Profiles → Savers → Settings：**Profiles** 是 object（使用者設定好的、有名字的 saver 實例），new / duplicate / rename / delete 都在這裡；**Savers** 是 class（clock、dino），沒有名字、不能增刪，`[2]` 是說明加預設值，動作 `[n] New`、`[p] Preview`；**Settings › preference**。
28. （2026-09-24，使用者定案）每種 saver 有一組預設值存在 config 的 `savers`，欄位同它的 profile；只影響之後新增的 profile，不動既有的；`[p]` 在 saver 上用預設值預覽。內建：clock 是 row / large / 3x5 / `HH MM SS` / `YYYY-MM-DD`，dino 是 trex / grassland，顏色同 splash。名字取 profile（iTerm / VS Code 的「一組有名字的設定」），不用 config（跟檔案和 preference 撞）。config key 對應改名：`profile` / `profiles` / 每個 profile 的 `saver`，舊 key 自動轉。profile 的 saver 建立後不改。開啟時 cursor 停在啟用中的 profile。
32. （2026-09-25，使用者定案）側欄多第三個區塊 **Integration**（順序 Profiles → Savers → Integration → Settings），`tmux` 與 `screen` 各一項：原 preference 的 `tmux_conf` / `screen_conf` 搬來當各自的 `conf`，`idle_lock` 不再共用、各工具一份；`[2]` 就是 `conf` + 閒置鎖 + 唯讀的 `status`（installed / not installed；同日修訂：原本多一列 tool 名稱、status 叫 block，使用者看不懂）；`[S] Setup` / `[X] Remove` 取代 CLI `locku setup [-d]`，指令拿掉。preference 每列下面的說明列拿掉，說明搬到 `?` help：focus 在 preference 的 `[2]` 時 `?` **只有**這些說明、自動換行，沒有鍵；tmux / screen 的 `[2]` 同理只有它們的；`[1]` 與 profile / saver 的 `[2]` 上 `?` 是鍵（同日修訂兩次：先是說明段跟著 cursor 附在鍵後面，使用者說要「只有」說明、而且要 wrap）。config 的 `tmux_conf` / `screen_conf` / `idle_lock` 自動轉成 `tmux: {conf, lock-after-time}` / `screen: {conf, idle}`——閒置鎖用工具自己的設定名稱（同日第三次修訂，使用者：tmux 就用 tmux 的 `lock-after-time`；早上寫出的 `tmux.idle_lock` / `screen.idle_lock` 也自動轉）。每個 `[2]` 第一列是表頭 `Property` / `Value`（同日，使用者：所有 panel 2 都給標題列；再同日改成單數 Property）。tmux 的 `[2]` 再多一列 `bind-key`（同日，使用者：prefix shortcut，跳輸入框，有值就幫使用者加 tmux 熱鍵、空就拿掉）：列名與 config key `tmux.bind-key` 都用 tmux 自己的指令名；Enter 開 input 框，有值就多寫 `bind-key <鍵> lock-server` 並即時 bind，空字串就那行不寫、server 上 unbind；Uninstall 一併 unbind。
33. （2026-09-25，使用者定案）整合的寫入全在 TUI，使用者不用再跑去 CLI：`[2]` 底部一條分隔線下方、置中一顆 **Install / Uninstall 按鈕**，平常不亮、cursor 到才亮（Enter → confirm → 執行；取代同日早上的 `[S]` / `[X]` 熱鍵，畫面上看不到、使用者無法直觀知道），**裝一次，之後隨設即得**——裝著時任何一列改動直接重寫區塊並套到 server，conf 改路徑就搬區塊、清空就拿掉；沒裝只寫 config.yaml。兩個純方案都否決：純隨設即得（conf 打錯就生檔、Remove 只能等於清 conf、不能先填好再裝）、純按鈕（改了值檔案 stale）。`status` 從列表拿掉——那是狀態不是屬性——放在 `[2]` 標題的膠囊鏈尾巴，`installed` 綠 / `uninstalled` 灰。標題改成家族的 **powerline 膠囊鏈**（sshu `panelChip`、filu `singleChip`、webu `tabChain`；locku 是唯一還畫純文字標題的成員）：`[2] <名字>` 邊框色、種類 `profile` / `saver` / `integration` / `settings` 灰、狀態各自上色——但只在面板 focus 時：沒 focus 整條一起是 unfocus 的灰，`uninstalled` 永遠灰（同日修訂，使用者：原版 uninstalled 用 peach、按鈕是表格最後一列且常亮）；` · ` 的三格換成一格接縫；profile 也補上 `profile` 這段；`[1] locku` 一顆。同日：**preview 不驗 PIN**，任意鍵就回設定畫面，PIN 是 `locku lock` 的事；狀態列仍照真的鎖顯示。

## 11. 待決清單

無。2026-09-24 全部定案。

## 12. MVP 驗收

- tmux 內 `lock-session` 後：prefix+d、prefix+&、prefix+c、prefix+x 皆無反應。
- Ctrl+C、Ctrl+Z、Ctrl+\ 無反應。
- 錯誤 PIN 留在 prompt；正確 PIN 後 tmux 畫面完整恢復，pane 內程式狀態未變。
- 錯誤 PIN 後 1 秒內的輸入被吞掉。wrong_pin_attempts 設 3 時，第 3 次錯誤後顯示倒數，倒數期間輸入無效，結束後可再試。
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
- Integration › tmux 的 Install 寫兩次，設定檔內容相同；有 server 時 `tmux show -g lock-command` 立即是 `<絕對路徑> lock -S '<socket>'`；Uninstall 後區塊消失、server 上五條（有 bind-key 是六條）都拿掉（setup 套件測試；TUI 的 Install / Uninstall 在 app 測試與 `make e2e` 裡真的按；裝著時改 lock-after-time、清 bind-key、搬 conf 都在 app 測試裡驗檔案）。
- 鎖著的時候誰進來都被鎖（2026-09-24，`make e2e` 在真的 tmux 3.7c 上跑）：`locku` 後 client A 看到點陣板、全域標 `@locked`；B 此時 attach 同一個 session、E attach 另一個 session，都看到點陣板；A 解鎖後 `@locked` 消失、B 與 E 仍鎖著直到各自解鎖；之後 attach 的 C 不被鎖；設定畫面填的 `bind-key l` 寫進區塊、server 上有這個 bind（2026-09-25）；A 這次由 `prefix l` 鎖住，鎖定中 A 的終端機死掉，`@locked` 留著，接著 attach 的 D 被鎖、D 解鎖才清掉；Uninstall 後區塊消失。
- shell 環境有 LOCKPRG 時 `C-a x` 進的是 locku（2026-09-24 以探針實測通過；`.screenrc` 的 `setenv` 路線實測不通，6.2 已改為 shell rc）。
- tmux lock-command 期間 prefix 到不了 tmux（2026-09-24 以探針實測：鎖定中送 prefix+d、prefix+c 都被鎖定程式吞掉，client 仍 attached；解鎖後 prefix+d 才 detach）。
