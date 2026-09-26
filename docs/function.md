# locku — 功能定義

> v1.0 定案，2026-09-24；2026-09-25 全文對齊程式碼重寫（使用者：文件都是最後一起重寫）
> 本文件只談「做什麼、怎麼達成」。版面放 ui.md，按鍵與流程放 ux.md。

## 0. 定位

一句話：跑在真實終端機上的螢幕保護程式加密碼鎖。啟動後任何按鍵都不轉發，只會彈出密碼輸入，驗證通過才把終端機還回去。

### 0.1 是什麼、不是什麼

- 是 terminu family 第五個成員（kbu / filu / sshu / webu / locku），同一套 Go + Bubble Tea 技術棧與 terminu design principle（tdp）。
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
4. 執行期間不開網路。子進程只有兩種：對 tmux 立 / 清 `@locked` 旗的短命令（6.2），與 custom saver 的程式（5.5）。設定檔只讀，`pin_hash` 每一鍵重讀一次（4.5）。

### 1.3 多 client

tmux `lock-session` 對每個 attach 中的 client 各跑一份 locku，彼此獨立，A 解鎖不影響 B。這是 tmux 的行為，locku 不需要知道其他實例存在。

但 tmux 沒有「鎖著」這個狀態：`lock-server` / `lock-session` 只對那一刻 attach 著的 client 送 MSG_LOCK，之後 attach 進來的 client 什麼都不會發生（實測 2026-09-24，tmux 3.7c）。使用者要的是「只要進來、而它是鎖著的，就得看到保護程式」，所以 locku 補一個狀態：`locku lock` 啟動時把**全域** user option `@locked` 設成 1（每個 session 都看得到），Integration › tmux 的 `activate` 寫進區塊的 `client-attached` / `client-session-changed` hook 看到 `@locked` 就對那個 client `lock-client`；PIN 對了（或無 PIN 模式任意鍵）結束前把 `@locked` 拿掉；tty 消失或被砍死不拿掉，下一個進來的人還是被鎖，直到有人輸入 PIN。第一個輸入正確 PIN 的 client 把標記清掉，其他還開著 locku 的 client 各自輸 PIN 才回來（使用者 2026-09-24：維持各自輸，不做輪詢）。

鎖的範圍**預設是整台 server**，不是 session（使用者 2026-09-24 定案；2026-09-25 加 `lock-session` 一檔給使用者選，Integration › tmux › `lock`，見 6.2 與決定 34）：螢幕保護程式保護的是「坐在這台終端機前的人看得到什麼」，鎖住一個 session 的話 `tmux attach -t 另一個` 就繞過去了，等於沒鎖；一份 config、一個受管區塊、一個 profile 對整台，也不用替每個 session 各配一套。所以 `locku` 這個 alias 是 `lock-server`。閒置鎖是 tmux 的極限——`lock-after-time` 是每個 session 各自計時——維持「哪個畫面閒置就鎖哪個畫面」，不升級成整台（雙螢幕各 attach 一個 session 時，正在打字的那邊不該被另一邊的閒置鎖到）。細節見 6.2。

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
| SIGWINCH | 視窗大小改變 | 接收並重繪；custom saver 再把新尺寸轉給程式的 pty（5.5） |
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

custom saver（5.5）是同一張圖：saver 是程式自己的畫面，prompt 是疊在上面的框；程式自己結束時 saver 換成板子上的 `EXIT <code>` / `NONE`，圖不變。

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
- 修改：在 `locku` 設定 TUI 內操作，需先輸入舊 PIN。忘記走 `locku pin reset`（4.5）。

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

### 4.5 忘記 PIN：`locku pin reset`（2026-09-25，使用者定案）

前提照 0.1：locku 不是安全邊界，能用這個帳號執行指令的人本來就能 `pkill -9 -f "locku lock"`（進程結束 = 解鎖）或改 `pin_hash`。所以忘記 PIN 的路不假裝比帳號權限更安全，也不弱於它——它把「你自己從另一個 shell 把鎖處理掉」變成一個乾淨、有名字、有紀錄的動作。

- **`locku pin reset`**，從任何一個你自己的 shell 跑（另一個終端機視窗、SSH、lock-session 模式下 attach 別的 session）。問兩次：先 `Reset the PIN? … [y/N]`（只有 `y` / `Y` 算同意，其他一律 `left as it is`、exit 1），再 `Password for <user>:`——**登入密碼**，不回顯（帳號就是邊界，拿它當確認；靜態 binary 沒有 PAM，用 `su <user> -c true` 開在 locku 自己的 pty 上、等它印出密碼提示再把密碼打進去、看退出碼——輸錯 su 自己會拖時間；等提示 5 秒、等結果 15 秒，逾時或 su 沒問就當錯）。兩關都過才**產生一組新 PIN**（crypto/rand 八位數字）、把它的 bcrypt 覆蓋 config 的 `pin_hash`、在畫面上**顯示一次**（`New PIN: 12345678`，接著說寫進了哪個檔、log 在哪），之後到設定畫面用它換成自己要的——這招照 elasticsearch 的 reset password：永遠不把 `pin_hash` 清空、永遠不留一個開著的鎖。
- **紀錄**：每次 reset（成功或密碼錯被拒）在 `~/.locku/data/pin-resets.log` 追加一行 `<RFC3339> pin reset by <user>@<host>: a new PIN written` / `refused: the password is not the account's`，**不寫 PIN**；檔 600、目錄 700；目錄可用 `$LOCKU_DATA` 改。
- **鎖定中的 locku 每次按鍵先重讀一次 `pin_hash`**（只讀這個值、不重讀整份 config）：reset 之後回到被鎖的 client，新 PIN 直接能用、舊的 `wrong`；tmux 全域鎖著的每個 client 各自輸一次（維持 9/24 的各自輸）。檔案讀不到、解析失敗、hash 不是 bcrypt → 沿用記憶體裡的 hash（鎖定中檔案壞掉不能變成開鎖）；檔案裡 `pin_hash` 變空 → 視同無 PIN，任意鍵解鎖。preview 不重讀。custom saver 的 PIN 框同一套；它「沒設 PIN 就第一鍵結束」看的是啟動時讀到的 config（5.5）。
- 不跑的情況：stdin 不是 tty（`needs a terminal to ask on`，exit 2）；config 讀不到——解析失敗、`pin_hash` 不是 bcrypt、`profiles` 為空（`config.yaml: <原因> — fix that first`，exit 1）——不然會把預設值連新 PIN 一起寫回去蓋掉使用者的檔；沒有檔案可以跑（等於在預設值上設 PIN）。
- 否決：在鎖定畫面做「忘記密碼」入口（長按、特殊序列、安全問題——路人也能按，等於沒鎖）；恢復碼（比帳號權限弱的東西不值得多一套流程）；把 `pin_hash` 清空當 reset（會留一個無 PIN 的鎖）。可選的 PIN 提示（`pin_hint`）未做，使用者未定。

## 5. 螢幕保護內容

### 5.1 兩層：saver 出內容，畫布出畫法

已決（2026-09-24）：saver 只決定「顯示什麼」，畫布只有一種畫法。

- **saver** 是種類——class：clock、dino、custom。它決定怎麼產生內容，輸出不帶任何樣式：clock 是幾行 ASCII 文字；dino 是一張自己像素座標的點陣圖（2026-09-24 加入第二種）；custom 不產內容，程式自己畫在 locku 給它的 pty 上，畫布只在它結束時接手（2026-09-25 加入第三種，5.5）。
- **profile** 是具名實例——object：一種 saver 加上它的參數與顏色，有名字；config 裡 `profile` 指向的、鎖定畫面顯示的，都是 profile（2026-09-24 定案，使用者以 OOP 分：class 不用取名、object 才有名字，能新增的是 profile、新增時先選 saver）。
- **畫布**把文字用 terminu family splash 的像素風格畫出來、把點陣圖依 size 放大鋪滿，依終端機格數自動選縮放，見 5.3。saver 碰不到顏色、字形、位置。custom 不經畫布。

### 5.2 saver 與 profile

| type | 參數 | 內容 | tick |
|---|---|---|---|
| clock | `layout` row / column；`size` small / medium / large；`font` 3x7 / 3x5；`time` `HH MM` / `HH MM SS`；`date` off 或四選一；`bg` / `fg` 兩個顏色 | row：一列時間，date 不是 off 時第二列日期；column：依分隔符拆行，`HH` / `MM` / `SS`，日期再拆 `YYYY` / `MM` / `DD` | time 含秒為 1 秒，否則對齊整分每 60 秒 |
| dino（2026-09-24） | `runner` 跑者（2026-09-25 修訂，六選一）：`big`（一隻大暴龍 12 × 14）、`small`（一隻小暴龍 8 × 10）、`big-big` / `small-small` / `small-big` / `big-small`（兩隻一前一後，名字就是畫面由左到右的順序——左邊在後、右邊在前，各自跳各自的）；舊值 `trex` / `two-trex` 讀成 `big` / `big-small`，下次存檔寫新名；`scene` 場景：`grassland`（草原，障礙物是仙人掌）、`desert`（沙漠，障礙物是金字塔，沙地斑點較疏）；`bg` / `fg` 兩個顏色。沒有 size（使用者：dino 也沒有 size 的選項），畫布自己取塞得下的最大倍率 | Chrome 離線小恐龍遊戲當螢幕保護：地面與障礙物向左捲、跑者自己跳過去，無限循環沒有人玩、不會死。障礙物隨機，分小 / 中 / 大三個等級（2026-09-25 修訂，使用者：原本只有小和中）：草原是小仙人掌 1 / 2 / 3 株（5 高）、中的高仙人掌（7 高）、大仙人掌（6 × 10，粗幹兩臂）；沙漠是小金字塔（3 或 4 高）、中金字塔（5 高）、大金字塔（13 × 7）、小加小；間距隨機 44 到 100 px；兩隻跑者各自看自己前面的障礙物、各自在自己的視窗裡隨機起跳，後面那隻的步伐差半步；跳躍在「跳得過」的那段視窗裡隨機挑一幀起跳，跳多高看前面那個障礙物的等級（2026-09-25 再修訂，使用者：現在高度都一樣）：小的低跳、中的中跳、大的高跳；前面沒東西時偶爾也無故跳一下，高度隨機三選一；雲以三分之一速度飄。不記分、不畫時間，畫面上只有場景（使用者 2026-09-24：dino 上面不需要計算時間和分數）。場景像素：跑者 12 × 14、跳躍弧三條、都是 16 幀，最高 6 / 8 / 11 px 對應小 / 中 / 大，各比該等級在兩個場景裡最高的障礙物（5 / 7 / 10）高一格，沙漠的金字塔較矮、同一條弧跳過去多留幾格（2026-09-25 再修訂，原本一條 11 px 跳所有東西；再之前 8 px，加高三格才跳得過大仙人掌）、每幀走 2 px，最小場景 40 × 28（同日修訂，原本 40 × 25：跑者 14 加跳 11 加地面 2 加一列天空） | 每 70 ms 一幀（14 fps），整張換、不做 reveal |
| custom（2026-09-25） | `command`：使用者自己的指令，`sh -c` 跑，畫面由它畫；沒有 `bg` / `fg`（使用者）——結束時的板子用預設色 | 不經畫布：程式在 locku 開的 pty 上跑，輸出經 locku 的 `screen` writer 原樣到終端機；PIN 框疊在它還在動的畫面上；程式結束（它不該結束）就換成 locku 的板子照實寫 `EXIT <code>`（`EXIT` 金字，數字 0 綠、其他 peach；沒跑起來是紅色的 `NONE`），見 5.5 | 無，由程式自己 |

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
- 三種 saver：clock、dino、custom（2026-09-25）。前兩種只是多一個產內容的函式，不動畫布；custom 不經畫布——程式自己畫，見 5.5。使用者自由輸入的 text saver 已移除（2026-09-24），內容不可控。

saver 預設值（2026-09-24，使用者定案）：每種 saver 在 config 的 `savers` 有一組預設值，欄位跟它的 profile 一樣、只是沒有名字。它決定**之後**用這種 saver 新增的 profile 長什麼樣，改它不影響任何已存在的 profile；cursor 在 saver 上時 `[p]` 就用預設值跑一個臨時 profile 預覽。內建值（config 沒寫時）：

| saver | 預設值 |
|---|---|
| clock | layout row、size large、font 3x5、time `HH MM SS`、date `YYYY-MM-DD`、bg `#313244`、fg `#f2b753` |
| dino | runner big、scene grassland、bg / fg 同上 |
| custom | command 空；沒有 bg / fg（2026-09-25） |

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

dino 的畫法（2026-09-24）：場景是整塊板，k 從 3 往下取第一個讓場景（40 × 28 px；2026-09-25 修訂，跳躍弧加高前是 40 × 25）塞得下的，都塞不下就 1；沒有 size 設定；場景 w × h = 板的格數 ÷ k，地面因此貼滿整寬，右邊 / 下面除不盡的格留暗。每幀整張換掉、不做 reveal——世界在移動，不是內容在變。14 fps 不是閒置，CPU 會比時鐘高，這是遊戲 saver 的代價。

### 5.4 狀態列

已決（2026-09-24）：所有 saver 共用一行狀態列，內容 `user@hostname · 鎖定於 HH:MM`，user 是啟動 `locku lock` 的使用者。預設顯示，config `show_status: false` 可關。管多台 server 時靠它分辨機器與帳號，回來時知道離開多久。

4.3 的「未設定 PIN」提示與解析錯誤原因也在這一列，但不受 show_status 影響，永遠顯示。

### 5.5 custom：使用者自己的程式（2026-09-25，使用者定案）

使用者自己開發保護程式的動畫，locku 管其餘的——鎖、PIN、整合。profile 只有一個 `command`，`sh -c` 跑（可帶參數與 pipe），不做任何 sanitize：是使用者自己機器上自己的指令，README 寫明。

- **誰擁有終端機**：locku。真 tty 由 locku 握著（raw、ISIG 關、alternate screen、游標藏起），程式跑在 locku 開的 **pty** 上、自己一個 process group：它以為自己有終端機（尺寸、SIGWINCH、curses 正常初始化），它的 termios、崩潰、亂送訊號都碰不到 locku 的 tty。它輸出的 bytes 經 locku 的 `screen` writer 原樣轉到真 tty（每一個 byte 都到，不丟幀）；離開時 locku 自己送 reset 序列收尾。按鍵永遠到 locku（否則 `q` 就把 cmatrix 關了、PIN 也收不到）。stderr 另接一條 pipe，只留最後一行給狀態列。
- **不干涉生命週期**（使用者定案）：程式該無限迴圈；locku 不重啟、不讀它的畫面，只在鎖結束時對整個 process group 送 SIGKILL（`sh -c` 跟它底下的程式一起走）並等它被回收。視窗大小改變轉給 pty，這是任何終端機都會做的事。
- **按鍵 → PIN 框疊在動畫上**（使用者定案：底下的畫面要繼續動，跟其他 saver 的板子在 prompt 底下照常 tick 一樣）：程式不停、輸出不停。鎖定畫面的 PIN prompt 以一個沒有 renderer 的 Bubble Tea 程式跑，只透過 callback 把框的幾行交給 `screen` writer 畫在終端機正中央；框開著時程式每送出一段輸出，writer 就在後面補畫一次框——前後以 DECSC / DECRC 保存與還原游標與屬性、整段包在一次 synchronised update（`?2026`）裡——所以終端機看到的永遠是「這一幀加框」，程式察覺不到、也沒有一幀被丟掉。沒有 locku 的底色、不切 screen。同一套 prompt 狀態（wrong、cooldown、timeout）。
- **框收起**（Esc 或逾時）：writer 把框佔過的最大矩形以終端機自己的顏色清空；接著只對**閒置 ≥ 500 ms 沒畫過東西的程式**送一個 SIGWINCH 要它重畫（curses 程式收到就整頁重畫），一直在畫的程式不送——它自己會把那塊填回來，而且 curses 程式被要求重畫會從頭來過（2026-09-25 以 cmatrix 實測：閃一下、雨從頂端重來）。PIN 對了殺程式、結束。沒設 PIN 時第一個鍵就結束（以啟動時讀到的 config 為準）。tty 消失：殺程式、結束、tmux 旗不清。
- **程式結束**（使用者定案）：鎖不退。畫面換成 locku 的點陣板，字用 clock 的 3x7 字型，large → medium → small 依尺寸退階，放不下就純文字單色；狀態列紅字寫原因。板子照實寫 **`EXIT <code>`**：`EXIT` 四個字母金色（預設 fg），數字 0 綠、其他 peach——板子因此第一次有兩種亮色，畫布多一個 accent；exit 0 → `EXIT 0`（狀態列 `custom saver exited 0`）；exit 非 0 → `EXIT 3`（狀態列 `custom saver: exit 3 · boom`，接 stderr 最後一行；找不到指令是 `EXIT 127`）；被訊號殺 → 跟 shell 一樣 128 + 訊號號碼，`EXIT 139`（狀態列 `custom saver: killed: segmentation fault`）；沒填 command 或起不來 → `NONE` 整個紅字（狀態列 `custom saver: no command` / 錯誤原因）。都不重啟、PIN 照常。
- **沒有 fg / bg**（使用者）：畫面是程式的，顏色設定沒意義；`[2]` 沒有色票列、沒有草稿與 Save / Reset，檔案裡也不寫；結束時的板子底用預設色（surface0）。
- **preview**：設定畫面把終端機交給程式（Bubble Tea 的 exec），跑同一套 pty 與 writer，任意鍵殺掉回來；程式結束或沒填指令就回到畫面內、以板子上的字預覽。全域 `P` 在 profile / saver 的 `[2]` 上是那一個、在其他 `[2]` 上是啟用中的 profile、在 `[1]` 上什麼都不做（`p` 才是預覽游標那列）。
- 否決的替代方案與同日的修訂史在決定 35、38。驗收：`e2e/custom_lock.py` 與 `internal/custom`、`internal/ui` 的測試，見 §12。

## 6. CLI

| 指令 | 作用 |
|---|---|
| `locku` | 開啟設定 TUI，見 6.1。不會鎖 |
| `locku lock` | 鎖住當前 tty。tmux、screen、裸 tty 都是叫這個。可帶 `-S <socket>`（要標 `@locked` 的 tmux server）與 `-t <session>`（lock-session 時要標的 session）——這兩個是 tmux 整合寫進 lock-command 的，使用者不用打，見 6.2 |
| `locku pin reset` | 忘記 PIN 時的路（2026-09-25）：`[y/N]`、登入密碼，然後產生一組新 PIN 顯示一次並寫進 config，鎖定中的 locku 下一鍵就認得，見 4.5 |
| `locku version` | 版本 |
| `locku help`（`-h`、`--help`） | 用法。不認得的子指令印用法、exit 2 |

tmux / screen 的整合設定不是指令，在設定 TUI 的 Integration 區塊做（6.1、6.2）。

argv[0] 為 `SCREEN-LOCK` 時視同 `locku lock`（不帶 `-S` / `-t`）。原因：screen 的 LOCKPRG 由 screen 直接 execl，不經 shell、不能帶參數，argv[0] 固定為 SCREEN-LOCK（macOS /usr/bin/screen 二進位內可見此字串）。這是 screen 唯一能把「要鎖」這個意圖傳給 locku 的通道。execl 也不搜 PATH，所以 LOCKPRG 必須是絕對路徑。

### 6.1 設定 TUI 的職責

裸指令 `locku` 開一個 TUI，只做設定：

- 設定或更改 PIN：輸入兩次確認，已有 PIN 時先驗舊的。
- 清除 PIN：回到無 PIN 模式，需先驗舊的；驗過之後在 `New PIN` / `Remove PIN` 選單選 Remove，Enter 立即生效、不再 confirm（2026-09-24）。
- saver 預設值：每種 saver 的 `[2]` 列出它的預設值，可改，只影響之後新增的 profile（5.2）；`p` 用預設值預覽。
- profile 管理：new（從一種 saver）、duplicate、rename、delete、編輯參數（5.2）：clock 的 layout、size、font、time、date，dino 的 runner、scene，custom 的 command；clock 與 dino 再有 bg / fg 兩個顏色，各以 R G B 三個 slider 設定（webu slider 作法，數字清單不打字），config 存 hex。顏色走草稿：滑桿改的是草稿，`S` 才寫檔、`R` 丟掉草稿，其餘欄位立即寫檔（2026-09-24：使用者調歪過一次調不回來）。custom 沒有顏色，也就沒有草稿與 `S` / `R`。
- preference：啟用中的 profile（`profile`）、show_status、`pin_prompt_timeout`、`wrong_pin_attempts` / `wrong_pin_attempt_cooldown`。設為啟用在這裡，或側欄 profile 列按 `a`（2026-09-25，使用者：不必每次到 preference 切）。
- Integration（2026-09-25，使用者定案）：側欄第三個區塊，`tmux` 與 `screen` 各一項，`[2]` 的列：
  - **`activate`**（`on` / `off`）：區塊在不在 `config file path` 那個檔案裡，每次畫都讀檔。Enter → confirm → 執行：on 把區塊寫進檔案（tmux 有 server 在跑就整塊 `source-file` 進去；screen 連 shell rc 一起寫、跑著的 session 即時 `screen -X`），off 拿掉（tmux server 上的、跑著的 screen session 上的一併拿掉）；路徑沒填時 disabled 並說 `set the config file path first`。
  - **`config file path`**：要寫的檔案，`~/` 可用，提議 `~/.tmux.conf` / `~/.screenrc`（webu 的提議作法，ux.md §2.1）。
  - 一條分隔線：上面是 locku 的設定，下面是寫進工具設定檔的 key。
  - tmux 的 **`lock`**：鎖的**範圍**，`lock-server`（預設，整台 server，鎖著時 attach 任何 session 都被鎖）或 `lock-session`（只鎖觸發的那個 session：它的 client 與之後 attach 它的人，別的 session 照常）。值用 tmux 的指令名，因為 alias 與 bind-key 最後跑的就是它；`?` 說明只講範圍、不講觸發方式（使用者：提到 bind-key 會誤導）。screen 沒有這列：每個 screen 是自己一個 process、LOCKPRG 跟著 shell，沒有範圍可選，不硬造。
  - 閒置鎖，**用工具自己的設定名稱**：tmux 是 `lock-after-time`、screen 是 `idle`（使用者：tmux 就用 tmux 的名字），各自一份、預設 300、0 關閉，config key 同名。
  - tmux 的 **`bind-key`** / screen 的 **`bind`**：prefix / C-a 之後按哪個鍵就鎖，照工具自己的寫法（tmux `l`、`C-l`、`F12`；screen `l`、`^L`），config key `tmux.bind-key` / `screen.bind`；有值就在區塊多寫一行 `bind-key <鍵> <lock>` / `bind <鍵> lockscreen`，空就不綁（screen 內建 `C-a x` 本來就是 lockscreen，`?` 要說）；含空白或 `#` 拒收。
  - **開一次，之後隨設即得**（使用者定案）：`activate` on 時任何一列改動就直接重寫區塊、tmux 整塊套到 server、screen 送進跑著的 session；`config file path` 改路徑就把區塊從舊檔拿掉、寫進新檔，清空就拿掉；做完 toast 一行結果。off 就只寫 config.yaml，activate 仍由使用者開。兩個純方案各有硬傷：純隨設即得會在路徑打錯時生檔、也不能先填好再開；純按鈕會 stale——改了 lock-after-time 檔案裡還是舊值。
- 每個 `[2]`——profile、saver、tmux / screen、preference——第一列都是表頭 `Property` / `Value`，側欄區塊標題的 Blue，不可停（2026-09-25，使用者：所有 panel 2 都給標題列）。
- 每一列的意思在 `?`：focus 在 preference 或 tmux / screen 的 `[2]` 時，help **只有**那個面板的說明（自動換行），沒有鍵的清單；screen 多一條講 LOCKPRG 住在 shell rc；其他地方的 `?` 是鍵（2026-09-25）。
- 預覽：不驗 PIN，任意鍵回設定畫面（2026-09-25，PIN 是 `locku lock` 的事）。`[2]` 在 profile / saver 上 `P` 是那一個（帶顏色草稿）、在 preference / tmux / screen 上是啟用中的 profile；`[1]` 上 `p` 是游標那列，`P` 不作用。custom 的預覽把終端機交給程式（5.5）。
- 寫出 `~/.config/locku/config.yaml`，權限 600。

TUI 的版面與按鍵放 ui.md / ux.md。

### 6.2 Integration 怎麼寫檔

只寫受管區塊，區塊外一個字都不動；再寫一次就是替換區塊，冪等——activate 開著時每改一列就這樣重寫一次。

| 目標 | 檔案 | 區塊內容 |
|---|---|---|
| tmux | Integration › tmux 的 `config file path`（使用者輸入，`~/` 可用，不存在就建；沒設 activate 就 disabled、說先填，不猜） | `set -gF lock-command "<locku 的絕對路徑> lock -S '#{socket_path}'"`、`set -g lock-after-time <lock-after-time>`、`set -s "command-alias[90]" "locku=<lock>"`、`set-hook -g "client-attached[90]" "if -F \"#{@locked}\" lock-client"`、`set-hook -g "client-session-changed[90]" "if -F \"#{@locked}\" lock-client"`；`lock` 是 lock-session 時再一行 `set-hook -g "session-created[90]" "set -F lock-command \"<絕對路徑> lock -S '#{socket_path}' -t '#{session_id}'\""`；`bind-key` 有填就再一行 `bind-key <鍵> <lock>`；每一行尾巴都有 `# locku` 註解 |
| screen | Integration › screen 的 `config file path`（同上），加上 shell rc：`$SHELL` 是 zsh 寫 `~/.zshrc`、bash 寫 `~/.bashrc`、fish 寫 `~/.config/fish/config.fish`、其他寫 `~/.profile` | screenrc：`idle <idle> lockscreen`；`bind` 有填就再一行 `bind <鍵> lockscreen`；shell rc：`export LOCKPRG=<絕對路徑>`（fish 是 `set -gx LOCKPRG <絕對路徑>`）；每一行尾巴都有 `# locku` 註解 |

區塊標記：

```
# >>> locku >>>
...
# <<< locku <<<
```

- 路徑由使用者在 Integration 各項的 `config file path` 輸入：沒設時 `activate` 是 disabled 並說 `set the config file path first`，什麼都不寫（screen 連 shell rc 也不寫）；不猜路徑。相對路徑拒收，它會落在程式剛好執行的目錄。
- **tmux 有 server 在跑時，把整個區塊交給它**（2026-09-25，使用者定案，見決定 39）：區塊的那幾行寫進一個暫存檔、`tmux source-file` 它、刪掉——server 拿到的就是檔案拿到的同一份文字，不另外維護一份指令清單。順序：先 undo 舊區塊做過、新區塊不做的事（檔案裡原本綁的鍵跟這次不同就 `unbind-key` 舊的；`lock` 換檔就 `set -gu @locked` 並拿掉 `session-created` hook——換檔後殘留的全域旗會讓所有 session 看似被鎖），再 source 區塊，最後對每個既有 session 逐一設它自己的 lock-command（lock-session：`set -t <id> -F lock-command "… -t '#{session_id}'"`；lock-server：`set -u -t <id> lock-command`）、換檔時再清每個 session 的 `@locked`——區塊裡的 session-created hook 只管之後建立的 session。activate off 時反向一條對一條拿掉：`set -gu lock-command` / `lock-after-time`、`set -su command-alias[90]`、三個 `set-hook -gu`、綁過的鍵 `unbind-key`、每個 session 的 lock-command 與 `@locked`、再 `set -gu @locked`。綁過與否看的是檔案裡區塊的 `bind-key` 行，不另外記。unbind 之後那個鍵 tmux 內建的功能（例如 `l` 的 last-window）要 server 重啟才回來。沒有 tmux 或沒有 server 就跳過並說明。結果以 toast 一行回報（setup 印的幾行以 ` · ` 接起來）。
- tmux 那五行的道理（2026-09-24，使用者定案，全部以 pty 實測 tmux 3.7c）：
  - **預設不綁熱鍵**。用戶既然在用 tmux 就有自己一套 bind，`bind L` 會撞。改用 command alias：`prefix :` 然後打 `locku`，就是 `lock-server`（整台的 client 全鎖）；shell 裡 `tmux locku` 也一樣。要熱鍵的自己在 Integration › tmux › `bind-key` 填一個（2026-09-25，使用者：prefix shortcut），寫成 `bind-key <鍵> <lock>`——鍵是使用者選的，撞不撞他自己知道。
  - **每行尾巴 `# locku`**，加上受管區塊的頭尾標記，手動要移也認得出來；alias 與 hook 放在陣列的 90 號，不碰使用者自己的 0 號。
  - **lock-command 寫絕對路徑**：tmux client 是用它自己的 shell 環境跑 `sh -c`，`locku` 不一定在那個 PATH 上——找不到就是「畫面閃一下」（使用者 2026-09-24 實際踩到）。寫的是 `Binary()`：PATH 上的 locku（brew 的 symlink）優先，否則就是執行設定畫面的這個檔，跟 screen 的 LOCKPRG 同一套。所以從 repo 跑 `./locku` 開 activate，寫的就是 repo 那個 binary。
  - **lock-command 帶 socket**：lock-command 在 client 進程裡以 `system()` 跑，環境裡沒有 `TMUX`，被鎖的 client 也不在 `list-clients` 裡；`set -gF` 在讀檔時把 `#{socket_path}` 展開進去，`locku lock -S <socket>` 才知道要跟哪個 server 講話（`-L` 開的 server 也對）。標記是全域的，locku 不必反查自己在哪個 session。
  - **`@locked` 由 locku 設與清，全域**：解鎖沒有 hook（3.7c 的 MSG_UNLOCK 只清 flag），所以 `locku lock` 啟動時 `set -g @locked 1`、正常解鎖結束前 `set -gu @locked`，tty 消失不清；沒帶 `-S` 就什麼都不做。兩個 hook 看到 `@locked` 就 `lock-client`（attach 任何 session 與 switch-client 都驗過會觸發）。activate off 順手 `set -gu @locked`。
  - **`lock-session`（2026-09-25，使用者定案；研究 `.local/studies/lock.md` §5）**：alias 與 bind-key 指向 `lock-session`，旗立在 session 上（`set -t <session_id> @locked 1`、解鎖 `set -u -t <session_id> @locked`），hooks 一模一樣——`#{@locked}` 先查 client 當下 session 的 option、沒有才 fallback 全域（實測），所以旗立在哪就是 scope。lock 程式怎麼知道自己是哪個 session：研究 §7 的「用 tty 反查 `display-message -p -c <tty>`」實測**不行**——鎖定中的 client 不在 `list-clients` 裡，`-c` 找不到就 fallback 到最近的 session，回錯的；改成每個 session 自己的 `lock-command`（它是 session option）由 `session-created` hook 在 session 建立時 `set -F` 烘入 `-t '#{session_id}'`，activate 時對既有 session 逐一設；`$0` 一定要加單引號，否則跑 lock-command 的 `sh -c` 會把它吃成自己的名字（實測收到 `-t sh`）。session 模式的定位是「各工作區獨立的視覺遮蔽」，不是安全隔離：`capture-pane -t 別的session` 不經 client、hook 不觸發（研究 §6）。研究 §7 的「解鎖一次全亮」輪詢不做，維持各自輸（9/24 已否決）。
  - lock 程式對 tmux 的呼叫（立旗、清旗）都有 2 秒 timeout、失敗一律靜默：不可能讓鎖起不來或掉下來。裸 tty、screen、沒帶 `-S` 時什麼都不做。
- screen 的 LOCKPRG 只能走 shell 環境（實測 2026-09-24，macOS screen 4.00.03，以探針程式經 pty 驗證）。原本想走 `.screenrc` 的 `setenv LOCKPRG` 一個檔搞定，實測不通：按 `C-a x` 出現的是 screen 內建的 `Key:` 鎖，探針沒被呼叫。原因是 `lockscreen` 由 attacher（接著終端機的前端進程）呼叫 `getenv`，而 `.screenrc` 只有後端讀、`setenv` 改的是後端與視窗內 shell 的環境；attacher 的環境在 `screen` 或 `screen -r` 執行那一刻就固定了。環境變數路線則完全符合設計：LOCKPRG 被 execl、`argv[0]` 是 `SCREEN-LOCK`、stdin 是 tty。所以 activate 寫 shell rc 的受管區塊，並提示：新開 shell 才有這個變數；已在跑的 session 不必重啟，detach 後從新 shell `screen -r` 即可，因為 attacher 是新進程。
- screen 也即時套到跑著的 session（2026-09-25，使用者定案：原理照 tmux 那邊的作法；全部以 pty 實測 macOS screen 4.00.03）：`screen -ls` 列出的每個 session（tab 開頭的 `pid.name` 行；exit code 不看——沒 session 時回 1）各送 `screen -S <pid.name> -X idle <秒> lockscreen`，有 bind 就再送 `-X bind <鍵> lockscreen`，attached 或 detached 都收得到；檔案裡原本綁的鍵跟這次不同，先送 `-X bind <舊鍵>`（不帶指令就是解綁）；activate off 反向 `-X idle 0`、`-X bind <鍵>` 一條對一條。跟 tmux 一樣 best effort：沒有 screen、沒有 session 就跳過並說明，哪個 session 不收就 toast 上一句、其餘照送。能即時套的只有 idle 與鍵，**LOCKPRG 套不進去**——它是 attacher 的環境，在 `screen` / `screen -r` 那一刻就定了；所以一個從沒有 LOCKPRG 的 shell attach 的 session，idle 到了或按了鍵，跑的是 screen 內建的 `Key:` 鎖，直到 detach 後從新 shell 重新 attach；toast 與 `?` 說明都講明這點。實測事實：區塊每行尾巴的 `# locku` 註解 screen 4.00.03 讀得過（`idle 2 lockscreen   # locku: 0 never`、`bind l lockscreen   # locku: …` 都生效）；`bind l lockscreen` 蓋掉 `C-a l` 原本的 redisplay；`-X source <rc>` 也能重讀整個檔，但沒用它——一條對一條才能反向拿掉；`SCREENDIR` 有效，e2e 靠它不碰使用者自己的 session。
- 絕對路徑偏好 PATH 上找到的那個（通常是 brew 的 symlink），不用解析 symlink 後的 Cellar 路徑，升級版本後才不會失效。
- 做完以 toast 回報：改了哪個檔、有沒有即時套用（screen：套到幾個跑著的 session）、還需要做什麼（screen：新開 shell；已在跑的 session detach 後從新 shell 重新 attach，LOCKPRG 才是 locku）。
- 不備份。移除就是 `[2]` 的 `activate` 關掉（confirm 後）：把受管區塊從檔案拿掉、有 server 在跑就一併拿掉；區塊前面補的空行也一起拿掉，其餘一個字不動；沒有區塊就說沒有；檔案不存在不會生出來。screen 同時清 screenrc 與 shell rc 的區塊，跑著的 session 也 `-X idle 0`、綁過的鍵 `-X bind <鍵>` 解掉（解掉之後那個鍵 screen 內建的功能——例如 `l` 的 redisplay——要 session 重開才回來，跟 tmux 一樣）。`[2]` 的 `activate` 列每次畫都讀一次檔案，說區塊在不在。

## 7. 設定與儲存

只有一個檔：`~/.config/locku/config.yaml`

```yaml
auth: pin              # v1 只有 pin，保留給 pam 擴充
pin_hash: "$2a$10$..."   # 空或缺欄位 = 未設定 PIN，見 4.3
profile: clock         # 啟用的 profile name，必須存在於 profiles
profiles:
  - name: clock
    saver: clock          # clock / dino / custom：這個 profile 是哪一種 saver，建立後不改
    layout: row           # row / column（依分隔符拆行）
    size: medium          # small / medium / large：一個字型像素佔 1 / 2 / 3 格見方
    font: 3x7             # 3x7 / 3x5：字型高 7 列或 5 列，都是 3 格寬
    time: "HH MM"          # HH MM / HH MM SS（時分秒以空白分組，不畫冒號）
    date: off             # off / YYYY-MM-DD / YYYY-MMM-DD / MM-DD / MMM-DD
    bg: "#313244"          # 這個 profile 的點陣板暗格，預設 surface0
    fg: "#f2b753"          # 亮格，預設 splash gold
  - name: dino
    saver: dino           # dino 沒有 layout / size / font / time / date：畫布自己取最大倍率
    runner: big           # dino 才有：跑者，big / small / big-big / small-small / small-big / big-small（舊值 trex / two-trex 自動轉）
    scene: grassland      # dino 才有：場景，grassland / desert
    bg: "#313244"
    fg: "#f2b753"
  - name: matrix
    saver: custom         # custom（2026-09-25）：使用者自己的程式，見 5.5；沒有 bg / fg
    command: "cmatrix -b" # custom 才有：sh -c 跑的指令；空 = 未設，鎖定畫面的板子寫 NONE
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
    runner: big
    scene: grassland
    bg: "#313244"
    fg: "#f2b753"
  custom:
    saver: custom
    command: ""
show_status: true      # 狀態列 user@hostname · 鎖定於 HH:MM，見 5.4
pin_prompt_timeout: 30          # PIN 框連續幾秒無按鍵就收起，每次按鍵重算，0 = 永不收起
wrong_pin_attempts: 0           # 連續輸錯幾次進冷卻，0 = 關閉，見 4.4
wrong_pin_attempt_cooldown: 30  # 冷卻秒數
tmux:                           # Integration › tmux（2026-09-25：從頂層 tmux_conf / idle_lock 搬來，舊 key 自動轉）
  conf: "~/.tmux.conf"          # activate 寫的檔（畫面上叫 config file path）；空 = 未設定，activate 不能開
  lock-after-time: 300          # 閒置幾秒自動鎖，用 tmux 自己的名字（舊 key idle_lock 自動轉）；0 = 不自動鎖
  bind-key: ""                  # prefix 之後鎖的鍵，照 tmux 寫法（l、C-l、F12）；空 = 不綁（2026-09-25）
  lock: lock-server             # locku 與 bind-key 跑哪個鎖：lock-server 整台 / lock-session 只鎖這個 session（2026-09-25）
screen:                         # Integration › screen，各自一份，不跟 tmux 共用
  conf: "~/.screenrc"           # 同上；shell rc 另由 $SHELL 決定
  idle: 300                     # 同上，用 screen 自己的名字
  bind: ""                      # C-a 之後鎖的鍵，照 screen 的 bind 寫法（l、^L）；空 = 不綁，C-a x 本來就鎖（2026-09-25）
```

修訂（2026-09-24）：頂層 `style` 拿掉，顏色是每個 profile 自己的 `bg` / `fg`。同日 key 改名：`saver` → `profile`、`savers` → `profiles`、每個 profile 的 `type` → `saver`（saver 是 class、profile 是 object）；舊 key 讀進來自動轉（讀檔先解析成樹、改名再 decode），下一次寫檔就只剩新 key。`savers` 這個 key 隨後給了 saver 預設值：它是清單就是舊的 profiles，是對照表就是預設值，兩種寫法都認。

- 無 history、無 cache、無 session。
- config 是 profile 的唯一來源，命令列不提供覆蓋。
- `profile` 指向不存在的 name、或 `profiles` 為空：用內建預設 clock，狀態列顯示 config error，不算損毀。
- 讀取失敗的處理見 4.3。
- 閒置多久自動鎖由各工具自己的那一列決定——tmux 的 `lock-after-time`、screen 的 `idle`，名字就是工具自己的（2026-09-25；舊的共用 `idle_lock` 自動轉），activate on 時原樣寫進去、一改就重寫；locku 自己不計時。

## 8. 安裝與整合（README 要交付的內容）

順序不限：不設定就是純螢幕保護，任何鍵解鎖；要密碼再執行 `locku` 設 PIN。

在 `locku` 側欄 Integration › tmux / screen 的 `[2]` 填 `config file path`、把 `activate` 打開（confirm 後）就直接寫進設定檔，做法見 6.2；開著時改任何一列就直接重寫。以下是它寫的內容，手動設定也是同一份：

tmux，寫進 Integration › tmux › config file path（慣例 `~/.tmux.conf`）：

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

最後那行只在 `bind-key` 有填時才寫（這裡填的是 `l`）；`lock` 選 `lock-session` 時 alias 與 bind-key 指向 `lock-session`，並多一行：

```
set-hook -g "session-created[90]" "set -F lock-command \"/opt/homebrew/bin/locku lock -S '#{socket_path}' -t '#{session_id}'\""  # locku: a session's lock knows its session
```

鎖：`prefix :` 打 `locku`（或 shell 的 `tmux locku`，填了 bind-key 就 `prefix l`）整台的 client 全鎖（lock-session 時只鎖這個 session 的），閒置 300 秒的畫面也鎖；鎖著的時候誰 attach 進來都會看到保護程式。移除：`activate` 關掉。

screen，寫進 Integration › screen › config file path（慣例 `~/.screenrc`），同一套區塊標記與 `# locku` 註解：

```
# >>> locku >>>
idle 300 lockscreen   # locku: 0 never
bind l lockscreen     # locku: C-a l locks, as C-a x does
# <<< locku <<<
```

最後那行只在 `bind` 有填時才寫（這裡填的是 `l`）；`C-a x` 是 screen 內建的 lockscreen，不填也鎖。加上 shell rc（`~/.zshrc` 或 `~/.bashrc`；fish 用 `set -gx`），因為只有 attacher 的環境會被 lock 讀到（6.2）：

```
# >>> locku >>>
export LOCKPRG=/usr/local/bin/locku   # locku: screen's LOCKPRG
# <<< locku <<<
```

跑著的 session 即時收到 idle 與 bind（`screen -X`），但 LOCKPRG 要新開 shell 才有：已在跑的 session detach 後從新 shell `screen -r`。移除：`activate` 關掉。

裸 tty：直接執行 `locku lock`。

忘記 PIN：`locku pin reset`（4.5）。

平台：macOS / Linux（WSL 可）；不支援 Windows（第 9 節）。

分發：vulcanshen/homebrew-tap formula；GitHub release 附 static binary（linux amd64 / arm64、darwin arm64）。

## 9. 技術選型

- Go，Bubble Tea + Lipgloss，與 kbu / filu / sshu / webu 同棧；`rmhubbert/bubbletea-overlay` 疊 popup。
- `golang.org/x/crypto/bcrypt`（PIN）、`gopkg.in/yaml.v3`（config，先解析成樹再改名舊 key）。
- 2026-09-25 加入：`creack/pty`——custom saver 的程式跑在 locku 開的 pty 上（5.5）、`locku pin reset` 的 `su` 也在 pty 上（4.5）；`charmbracelet/x/term`——raw mode、尺寸、`ReadPassword`；`charmbracelet/x/ansi`——疊框時算每行的顯示寬；`muesli/cancelreader`——custom 鎖上可以取消的按鍵讀取，PIN prompt 的程式接手終端機時沒有一個 read 擋在前面。
- 無 cgo，`CGO_ENABLED=0` 靜態編譯。
- **平台：macOS / Linux（WSL 可），不支援 Windows**（2026-09-25，使用者定案）：鎖站在 tty、pty、`su` 與 tmux / screen 上，原生移植是另一個產品。
- 測試：狀態機、驗證、渲染、設定 TUI 全用 programmatic model test（不需要 tty）；`make check` = fmt-check + vet + `go test -race`（2026-09-25：custom 的 pump 與 screen writer、login 的 su、custom 鎖的 prompt 各有 goroutine，沒有 race detector 看不出來，多花十幾秒）；tmux / custom / screen 整合以 python pty harness 跑真的 binary（`make e2e`，第 12 節）。

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
21. 整合設定寫入設定檔的受管區塊，冪等；tmux 有 server 時即時套用。（2026-09-24 修訂）要寫的檔案由使用者輸入，沒設就報錯，不猜路徑；每一行尾巴 `# locku` 註解，手動也好移。（2026-09-25 修訂）原本是 CLI `locku setup [-d]`，改成 TUI 側欄 Integration 區塊的 `[S] Setup` / `[X] Remove`，指令拿掉；「不做 TUI popup」仍成立——它是 `[1]` 的一個區塊加 `[2]` 的列，不是 popup。（同日再修訂）熱鍵畫面上看不到，改成 `[2]` 第一列 `activate` 的開關（中間曾是底部一顆按鈕），Enter 後 confirm 才執行——confirm 是 message class，不是 Integration popup；見 33。
22. （2026-09-24 修訂）側欄 Enter 一律把焦點送到 `[2]`，包括 saver 與 profile；設為啟用在 preference › profile，側欄的 `●` 只顯示。Settings 只有 preference 一項，原 config 改名 preference、style 取消。
23. （2026-09-24 修訂）側欄 profile 的 item operation：`[Enter] Edit`、`[p] Preview`（預覽那一個 profile）、`[D]uplicate`、`[r]ename`、`[X] Delete`，D / X 大寫對齊 sshu；saver 的是 `[Enter] Edit`（看說明）、`[n] New`。
24. （2026-09-24 修訂）`[2]` 在 profile 上的 panel operation：`[P] Preview`（預覽正在編輯的這個 profile，帶草稿）、`[S] Save`、`[R] Reset`。全域 `P` 在 profile 的 `[2]` 上就是這個 profile，其他地方是啟用中的。
25. （2026-09-24）`tmux_conf` / `screen_conf`（2026-09-25 起是 Integration › tmux / screen 的 `conf`）是 locku 唯二的自由輸入，用 webu 的 input 作法：提議（目前值，沒有就是慣例路徑）dim 顯示，Tab 接手、Backspace 拒絕、Enter 照打的存、沒碰提議不改；只收絕對路徑或 `~/` 開頭。
26. （2026-09-24）第二種 saver `dino`：Chrome 小恐龍遊戲當螢幕保護，無限循環、隨機障礙、隨機跳躍、不會死、不記分；參數 `runner`（trex、two-trex：兩隻一前一後、前小後大、各自跳）、`scene`（grassland 仙人掌、desert 金字塔）、bg / fg，沒有 size（畫布取塞得下的最大倍率）。每 70 ms 一幀整張換，不做 reveal。（2026-09-25 修訂）使用者要 runner 六選一：`big`、`small`（一隻大或小暴龍）、`big-big`、`small-small`、`small-big`、`big-small`（兩隻一前一後，名字就是畫面由左到右的順序——左邊在後、右邊在前；`big-small` 就是原本的 `two-trex`），預設 `big`；舊 config 的 `trex` / `two-trex` 讀進來就是 `big` / `big-small`，下次存檔只寫新名。（2026-09-25 修訂）障礙物分大中小三個等級：草原加一株大仙人掌（6 × 10，粗幹兩臂），沙漠加一座大金字塔（13 × 7），原本的高仙人掌與五高金字塔就是中；跳躍弧最高 8 px 加到 11 px、幀數不變，最小場景 40 × 25 改 40 × 28（使用者：原本只有小和中）。（2026-09-25 再修訂）跳躍高度隨障礙物大中小而不同：三條弧、都是 16 幀，最高小 6 px、中 8 px、大 11 px，各比該等級在兩個場景裡最高的障礙物高一格，起跳時看前面那個障礙物的等級挑弧——看等級不看高度，因為沙漠的金字塔比仙人掌矮，照高度挑會讓中金字塔用低跳、大金字塔用中跳；幀數不縮短，因為小等級也有 11 到 13 px 寬的（三株仙人掌、小加小金字塔），跑者得在上面撐滿寬度，所以低跳是矮、不是短；無故跳隨機三選一；最高的弧沒變，最小場景 40 × 28 與最小間距 44 不動（使用者：現在高度都一樣）。
27. （2026-09-24，使用者定案）側欄分三個區塊，順序 Profiles → Savers → Settings：**Profiles** 是 object（使用者設定好的、有名字的 saver 實例），new / duplicate / rename / delete 都在這裡；**Savers** 是 class（clock、dino），沒有名字、不能增刪，`[2]` 是說明加預設值，動作 `[n] New`、`[p] Preview`；**Settings › preference**。
28. （2026-09-24，使用者定案）每種 saver 有一組預設值存在 config 的 `savers`，欄位同它的 profile；只影響之後新增的 profile，不動既有的；`[p]` 在 saver 上用預設值預覽。內建：clock 是 row / large / 3x5 / `HH MM SS` / `YYYY-MM-DD`，dino 是 trex（2026-09-25 起 big）/ grassland，顏色同 splash。名字取 profile（iTerm / VS Code 的「一組有名字的設定」），不用 config（跟檔案和 preference 撞）。config key 對應改名：`profile` / `profiles` / 每個 profile 的 `saver`，舊 key 自動轉。profile 的 saver 建立後不改。開啟時 cursor 停在啟用中的 profile。
29. （2026-09-24，使用者定案）tmux 不綁熱鍵，改 command alias `locku`（`prefix :` 打 `locku`），不跟使用者既有的 bind 撞。（2026-09-25 修訂：預設仍不綁，但使用者可在 Integration › tmux › `bind-key` 自己填一個鍵，Setup 多寫一行 `bind-key <鍵> lock-server`、有 server 就即時 bind；空就不綁，見 32。）「鎖著的時候誰進來都被鎖」用全域 user option `@locked` 加 `client-attached` / `client-session-changed` hook 做到：`locku lock` 啟動時設、正常解鎖時清、tty 消失不清；lock-command 是 locku 的絕對路徑並以 `set -gF` 帶 `#{socket_path}` 給 `locku lock -S`。以 pty 端到端測試（`make e2e`）驗收。
30. （2026-09-24，使用者定案）設定改名，三個都帶 lock 字看不出誰是誰：`prompt_timeout` → `pin_prompt_timeout`、`lockout_after` → `wrong_pin_attempts`、`lockout_seconds` → `wrong_pin_attempt_cooldown`；新增 `idle_lock`（閒置幾秒自動鎖，預設 300，0 關閉），一個值給所有拿 locku 當螢幕保護的工具：tmux 的 lock-after-time、screen 的 idle，setup 寫進去。舊 key 讀進來自動轉。（2026-09-25 再改：拆成各工具一份、用工具自己的名字，見 32。）
31. （2026-09-24，使用者定案；2026-09-25 修訂：預設仍整台，但加 `lock-session` 一檔給使用者選，見 34）鎖的範圍是**整台 tmux server**，不是 session：`locku` = `lock-server`，標記全域，attach 任何 session 都被鎖；螢幕保護程式保護的是整台，session 等級的鎖換個 session 就繞過。閒置鎖維持 tmux 的每 session 計時、不升級成整台；解鎖維持每個 client 各自輸 PIN、不輪詢。一份 config、一個區塊、一個 profile 對整台。
32. （2026-09-25，使用者定案）側欄多第三個區塊 **Integration**（順序 Profiles → Savers → Integration → Settings），`tmux` 與 `screen` 各一項：原 preference 的 `tmux_conf` / `screen_conf` 搬來當各自的 `conf`，`idle_lock` 不再共用、各工具一份；`[2]` 就是 `conf` + 閒置鎖 + 唯讀的 `status`（installed / not installed；同日修訂：原本多一列 tool 名稱、status 叫 block，使用者看不懂）；`[S] Setup` / `[X] Remove` 取代 CLI `locku setup [-d]`，指令拿掉。preference 每列下面的說明列拿掉，說明搬到 `?` help：focus 在 preference 的 `[2]` 時 `?` **只有**這些說明、自動換行，沒有鍵；tmux / screen 的 `[2]` 同理只有它們的；`[1]` 與 profile / saver 的 `[2]` 上 `?` 是鍵（同日修訂兩次：先是說明段跟著 cursor 附在鍵後面，使用者說要「只有」說明、而且要 wrap）。config 的 `tmux_conf` / `screen_conf` / `idle_lock` 自動轉成 `tmux: {conf, lock-after-time}` / `screen: {conf, idle}`——閒置鎖用工具自己的設定名稱（同日第三次修訂，使用者：tmux 就用 tmux 的 `lock-after-time`；早上寫出的 `tmux.idle_lock` / `screen.idle_lock` 也自動轉）。每個 `[2]` 第一列是表頭 `Property` / `Value`（同日，使用者：所有 panel 2 都給標題列；再同日改成單數 Property）。tmux 的 `[2]` 再多一列 `bind-key`（同日，使用者：prefix shortcut，跳輸入框，有值就幫使用者加 tmux 熱鍵、空就拿掉）：列名與 config key `tmux.bind-key` 都用 tmux 自己的指令名；Enter 開 input 框，有值就多寫 `bind-key <鍵> lock-server` 並即時 bind，空字串就那行不寫、server 上 unbind；Uninstall 一併 unbind。
33. （2026-09-25，使用者定案）整合的寫入全在 TUI，使用者不用再跑去 CLI：`[2]` 表頭下第一列 **`activate`（on / off）** 就是區塊在不在檔案裡，Enter → confirm → 執行（取代同日早上的 `[S]` / `[X]` 熱鍵——畫面上看不到、使用者無法直觀知道——與下午底部的 Install / Uninstall 按鈕——風格不對，回歸 property / value）；第二列 `config file path`（原 `conf`）；一條分隔線區隔 locku 的設定與工具自己的 key（閒置鎖、bind-key）。**開一次，之後隨設即得**——on 時任何一列改動直接重寫區塊並套到 server，路徑改了就搬區塊、清空就拿掉；off 只寫 config.yaml。兩個純方案都否決：純隨設即得（conf 打錯就生檔、Remove 只能等於清 conf、不能先填好再裝）、純按鈕（改了值檔案 stale）。`status` 列拿掉，狀態就是 `activate` 的值（下午曾放在標題膠囊尾巴，`installed` 綠 / `uninstalled` 灰）。標題改成家族的 **powerline 膠囊**（sshu `panelChip`、filu `singleChip`、webu `tabChain`；locku 是唯一還畫純文字標題的成員）：左上 `[2] <名字>` 邊框色，顏色草稿未存時接一顆 `unsaved`（focus 時黃、沒 focus 整條灰）；種類 `profile` / `saver` / `integration` / `settings` 先串在標題後（灰）、再搬到右上角一顆獨立膠囊、最後整個拿掉（同日第三、四版，使用者：第二階層拉到右上角、不跟 title 串接；再說很多餘，不需要分類資訊）；`[2]` 下框右側的 config 路徑同時拿掉（使用者問它為什麼在那；第一版就有、不是家族慣例、沒人需要）；膠囊字緊貼圓頭 cap、不留空白（使用者：圓角後多了一個空白），接縫兩側各一格；`[1] locku` 一顆。同日：**preview 不驗 PIN**，任意鍵就回設定畫面，PIN 是 `locku lock` 的事；狀態列仍照真的鎖顯示。
34. （2026-09-25，使用者定案）tmux 多一列 `lock`：鎖的範圍，`lock-server`（預設）或 `lock-session`，用 tmux 的指令名；`?` 的說明只講範圍、不提 bind-key（同日修訂，使用者：會誤導）（研究 `.local/studies/lock.md` §5 只給這兩檔、不給 client 檔：client 級鎖旁邊鏡像視窗還亮著，語意不成立）。session 模式：alias 與 bind-key 指向 lock-session、旗立在 session（`set -t <id> @locked`）、hooks 不變；lock 程式的 session 由該 session 自己的 lock-command 帶進來（`locku lock -S <socket> -t '<session_id>'`，session-created hook 烘入、activate 時對既有 session 逐一設），因為實測鎖定中的 client 用 tty 反查會拿到錯的 session（研究 §7 的作法否決）。換 lock 清所有旗與 hook。「解鎖一次全亮」不做（維持 9/24 的各自輸）。e2e 加一輪 lock-session：旗只在 session、attach 另一個 session 不受影響。
35. （2026-09-25，使用者定案）第三種 saver **custom**：使用者自己輸入指令當保護程式的動畫，locku 管鎖、PIN、整合。程式在 locku 開的 pty 上、自己一個 process group，輸出直通真 tty，按鍵留在 locku；**不干涉生命週期**——不重啟、不讀畫面，鎖結束就 SIGKILL 整個 group；按鍵時暫停轉發、換成只有 PIN 框的 lock 程式，Esc / 逾時後恢復並送 SIGWINCH 讓程式重畫。程式結束就用 locku 的板子照實寫 `EXIT <code>`（訊號是 128 + 號碼，跟 shell 一樣；沒跑起來是 `NONE`），`EXIT` 金色、數字 0 綠 / 其他橘、`NONE` 紅，large → small 退階，狀態列紅字寫原因，不重啟（同日三版：COMPLETED / ERROR → DONE / ERROR（字太多）→ 直接寫退出碼，使用者：更真實、也不用取名；顏色再修訂為字母金、數字上色）。custom 沒有 fg / bg（同日修訂，使用者）：PIN 框與板子用預設底色。否決：VT 模擬器路線、自動重啟、黑畫面加紅字。細節 5.5，驗收 `e2e/custom_lock.py`。
36. （2026-09-25，使用者定案）screen 的整合補到跟 tmux 一樣，**原理照 tmux 那邊的作法、名字用 screen 自己的**：`[2]` 是 activate、config file path、分隔線、`idle`、`bind`；`bind` 是 C-a 之後鎖的鍵，照 screen 的 `bind` 寫法（`l`、`^L`），config key `screen.bind`，有值就在區塊多寫 `bind <鍵> lockscreen`，空就不綁（`C-a x` 內建就是鎖，`?` 要說）；跟 tmux 的 bind-key 一樣不進 `Tool`，`SetTool` 不動它；同樣拒收空白與 `#`。**沒有 `lock` 列**：screen 沒有 server，每個 screen 是自己一個 process、LOCKPRG 跟著 shell，沒有範圍可選、不硬造。**即時套用照 tmux**：`screen -ls` 列出的每個 session `-X idle` / `-X bind`，off 反向一條對一條，best effort、失敗靜默、toast 回報；LOCKPRG 套不進 attacher，toast 與 `?` 說明講明（從沒有 LOCKPRG 的 shell attach 的 session，在重新 attach 前跑的是 screen 內建的 `Key:` 鎖）。曾考慮不即時套、只寫檔並提示 `C-a :source ~/.screenrc`——否決：那樣使用者每改一列就得自己 source，跟「開一次，之後隨設即得」相違；實測 4.00.03 的 `-X` 穩定，選即時套。`?` 說明多一條 LOCKPRG 住在 shell rc。驗收 `e2e/screen_lock.py`（真的 screen 4.00.03，自己的 SCREENDIR），Makefile `e2e` 第三個跑。
37. （2026-09-25，使用者定案）忘記 PIN：`locku pin reset`——`[y/N]` 後要**登入密碼**當確認（帳號是唯一的邊界，用 `su` 在 pty 上驗），過了就**產生新 PIN 覆蓋** config 並顯示一次（照 elasticsearch reset password，不把 `pin_hash` 清空），reset 紀錄寫 `~/.locku/data/pin-resets.log`（不含 PIN）；鎖定中的 locku 每次按鍵重讀 `pin_hash`，新 PIN 下一鍵就能用、檔案壞掉不變、清空視同無 PIN。否決：鎖定畫面上的忘記密碼入口、恢復碼、清空 hash。見 4.5。
38. （2026-09-25，使用者定案）custom saver 的 PIN 框**疊在動畫上、動畫不停**：程式的輸出照樣經 `screen` writer 直通，框開著時每段輸出後面補畫一次框（DECSC / DECRC 包住、一次 `?2026` synchronised update），沒有一幀被丟；框收起時清空它佔過的矩形，只對閒置 ≥ 500 ms 的程式送 SIGWINCH 要它重畫（cmatrix 實測：正在畫的程式被要求重畫會閃一下從頭來）。同日三版：先是「按鍵時暫停轉發、離開 alt screen、在 locku 自己的底上開框、回來送 SIGWINCH」（使用者：框下面沒有動畫）；再是「框畫在凍住的畫面上、回來時真的縮一欄再放回去逼它整頁重畫」（cmatrix 實測：只送訊號的 curses 程式只重畫它以為有變的部分，舊幀透出來）；最後是現在這版——不凍畫面就沒有補畫的問題。PIN prompt 因此是一個沒有 renderer 的 lock 程式，透過 callback 畫框。否決：內建 VT 終端機模擬器把輸出 parse 成格子再畫（多一個依賴、忠實度與效能都要驗）；重啟三次再退階（不干涉生命週期）；黑畫面加紅字（用板子）；框下面 locku 的底色；雙重 resize；對正在畫的程式送 SIGWINCH。同日：全域 `P` 在 `[1]` 上什麼都不做（`p` 預覽游標那列）；在 preference / tmux / screen 的 `[2]` 上預覽啟用中的 profile，custom 也走交出終端機那條路（原本走畫面內的鎖，custom 沒有顏色就成了一片白格）。`locku help` 說 `-S` / `-t` 是 tmux 整合傳的、使用者不用打，screen 以 SCREEN-LOCK 不帶參數呼叫。
39. （2026-09-25，使用者定案）tmux 的即時套用改成**整個區塊 `source-file`**：跑著的 server 拿到的就是檔案拿到的那幾行（寫進暫存檔、`tmux source-file`、刪掉），不再維護一份跟區塊平行的指令清單，server 與檔案不會走散；新區塊不做而舊區塊做過的先 undo（舊的 bind-key 解綁、`lock` 換檔時清全域與每個 session 的 `@locked`、拿掉 session-created hook），再對每個既有 session 逐一設或清它自己的 lock-command（hook 只管之後建立的 session）。e2e 因此在跑著 lock-server 區塊、且留了一個 stale 全域旗的 server 上從畫面切到 lock-session，驗 server 上的 alias、鍵、hook、每個 session 的 lock-command 都換了、旗清了；e2e 的 tmux 用預設 socket 名 `default`、跑在自己的 `TMUX_TMPDIR` 下，數鎖只數自己 socket 上的（使用者自己的 server 上有鎖也不會誤判）。
40. （2026-09-25，使用者定案）**平台：macOS / Linux（WSL 可），不支援 Windows**——鎖站在 tty、pty、`su` 與 tmux / screen 上，原生移植是另一個產品。`make check` = fmt-check + vet + `go test -race`：custom 的 pump 與 screen writer、login 的 su、custom 鎖的 prompt 三處各自有 goroutine，沒有 race detector 看不出資料競爭；多花十幾秒。
41. （2026-09-25，使用者定案）側欄 profile 列多一個熱鍵 **`a` Activate**：鎖定畫面改用這個 profile，`●` 移過去、立刻寫檔；已啟用的 disabled。preference › profile 那列照舊，是同一個設定的另一個入口；Enter 維持進 `[2]`（9/24 否決的是用 Enter 設啟用，不是熱鍵）。用 `a`：`p` 是預覽、`D` / `r` / `X` 已用、`u` / `d` / `g` / `G` 是導覽，`a` 跟 `●` 的語意對上。

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
- clock saver 連續執行 8 小時，CPU 平均 < 1%，記憶體不成長。（尚未做）
- 渲染器在 80×24、120×40、200×60 下對 `HH MM` 各選到預期的 k（large：2 / 3 / 3）；152×39 medium 配 `HH MM SS` + `YYYY-MM-DD` 是時間 2 倍、日期 1 倍帶年；130×24 只畫時間；column 配日期是兩欄、日期左時間右；30×8 退化為一般文字。
- 內容變更只動有變的像素，以 shuffle 揭露，沒變的像素輸出不變。
- profile name 重複被擋。
- 顏色任一 channel 改動進草稿、`S` 寫檔，Preview 反映；手改 config 的非法 hex 視同預設。
- config 不存在時進入無 PIN 模式：saver 照常顯示、標明未設定 PIN、任何按鍵結束。
- 設定 TUI 寫出的 config 能被 `locku lock` 讀取。
- LOCKPRG 指向 locku 本體時，screen `lockscreen` 進入鎖定而不是設定 TUI。
- Integration › tmux 的 activate 開著寫兩次，設定檔內容相同；activate 關掉後區塊消失、server 上的一條對一條拿掉（setup 套件測試；TUI 的 activate 在 app 測試與 `make e2e` 裡真的按；on 時改 lock-after-time、清 bind-key、搬 config file path 都在 app 測試裡驗檔案）。
- tmux 一輪（`e2e/tmux_attach.py`，真的 tmux 3.7c，26 項；2026-09-25 版）：預設 socket 名 `default`、自己的 `TMUX_TMPDIR` / HOME / config，數鎖只數自己 socket 上的。設定畫面填 `bind-key l`、activate 打開：檔案有區塊（每行 `# locku`、lock-command 絕對路徑帶 `#{socket_path}`、alias `locku=lock-server`、`bind-key l lock-server`），toast `wrote`；用這個檔開的 server 有 alias、hooks、鍵。`locku` 後 A 看到點陣板、全域 `@locked`；B attach 同一個 session、E attach 另一個 session 都看到點陣板；A 解鎖後旗消失、B 與 E 仍鎖著直到各自解鎖；之後 attach 的 C 不被鎖。`prefix l` 鎖住 A 並立旗，A 的終端機死掉旗留著，接著 attach 的 D 被鎖、D 解鎖才清掉。lock-session：在跑著 lock-server 區塊、且手動留了一個全域旗的 server 上，畫面上選 lock-session——檔案立刻重寫（alias、`bind-key l lock-session`、session-created hook 帶 `-t '#{session_id}'`、沒有 lock-server 殘留），server 上 alias / 鍵 / hook 都換了、每個 session 的 lock-command 帶自己的 id、stale 的全域旗清掉；`locku -t t` 後旗只在那個 session、全域沒有，B attach 同 session 被鎖、E attach 另一個 session 不受影響，A 解鎖清掉 session 的旗、B 各自解鎖。最後畫面上 activate 關掉：區塊消失，toast `removed`。
- custom 一輪（`e2e/custom_lock.py`，pty 上跑真的 binary，24 項，本機有 cmatrix 再加 4 項；2026-09-25）：無 PIN——程式的輸出出現在終端機、程式在跑、任意鍵結束並殺掉程式。設定畫面設 PIN。有 PIN——按鍵出 PIN 框，框直接疊在畫面上（沒有點陣板的 glyph、沒有切 screen），框開著時程式的輸出還在增加、框畫了不只三次，Esc 後框拿掉（有 `?2026l`）、輸出繼續、鎖與程式都還在，PIN 結束並殺程式。`echo boom >&2; exit 3` / `exit 0` / 沒填指令三種：都出現點陣板、狀態列各是 `custom saver: exit 3`（含 `boom`）/ `custom saver exited 0` / `custom saver: no command`、鎖不退、PIN 結束。cmatrix：`exec cmatrix -b -u 5` 有畫、按鍵出框、PIN 結束並殺乾淨。單元：`internal/custom`（框跟著每一幀補畫、結束碼與訊號的字、沒指令、尺寸到得了程式、框只碰自己的矩形）、`internal/ui/custom_test.go`（custom 的 `[2]` 只有 command、檔案只給 custom 存 command 且沒有顏色、板子上 `EXIT` 金 / 0 綠 / 其他 peach / `NONE` 紅與退階、prompt-only 的鎖每一步都經 callback 畫框、全域 `P` 對 custom 交出終端機且 `[1]` 上不作用）。
- screen 一輪（`e2e/screen_lock.py`，真的 macOS screen 4.00.03，15 項；自己的 SCREENDIR、HOME、SHELL=/bin/sh）：先開一個 session 在還沒有區塊的 rc 上（環境裡有 LOCKPRG，等於新 shell）；設定畫面填 `bind l`、activate 打開：rc 有區塊（`idle 300 lockscreen`、`bind l lockscreen`，每行 `# locku`，使用者自己的 `startup_message off` 留著）、`~/.profile` 有 `export LOCKPRG=<絕對路徑>`、toast `wrote`；`C-a x` 出點陣板、任意鍵解；`C-a l` 也出點陣板——這個 session 是在沒有 bind 的檔上開的，證明 `-X bind` 即時到了；再開一個 session 在有區塊的檔上，`C-a l` 一樣鎖；畫面上把 idle 改 2，檔案立刻是 `idle 2 lockscreen`、toast `wrote`、跑著的 session 兩秒後自己鎖；activate 關掉：兩個檔的區塊都消失（使用者的行留著）、toast `removed`、跑著的 session 不再自己鎖、`C-a l` 不再鎖。toast 只驗前幾個字（80 欄一行放不下整句）。
- 忘記 PIN（2026-09-25）：`cmd/locku/pin_test.go` 在 pty 上跑 `pinReset`——`y` + 正確密碼：畫面先有 `[y/N]` 與 `Password for`，再顯示一次八位數 `New PIN:`，檔案認新 PIN、不認舊的，log 有 `pin reset by …: a new PIN written`、不含 PIN；密碼錯：拒絕、log 記 `refused`、檔案不變；答 `n`：不問密碼、不變；stdin 不是 tty：`needs a terminal`、exit 2。`internal/config`：`NewPIN` 八位數字；`LoadPINHash` 沒檔案 / 非 bcrypt 回 ok=false，沒有 `pin_hash` 回空字串。`internal/ui/lockscreen_test.go` `TestLockReadsTheFilesPINAtEveryKey`：鎖定中檔案換了新 hash，舊 PIN `wrong`、新 PIN 開；檔案壞掉沿用原 hash；檔案清空任意鍵開。
- tmux lock-command 期間 prefix 到不了 tmux（2026-09-24 以探針實測：鎖定中送 prefix+d、prefix+c 都被鎖定程式吞掉，client 仍 attached；解鎖後 prefix+d 才 detach）。
- shell 環境有 LOCKPRG 時 `C-a x` 進的是 locku（2026-09-24 以探針實測通過；`.screenrc` 的 `setenv` 路線實測不通，6.2 已改為 shell rc）。
