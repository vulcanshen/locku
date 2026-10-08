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

按鍵流向：實體終端機 → tmux client 進程 → tmux server 進程 → 該 pane 的 PTY（pseudo terminal，虛擬終端）→ pane 內的程式。prefix 在 tmux server 就被消化，那個 byte 永遠不會進到 pane。screen 的 C-a 同理。所以「在 pane 內攔 prefix」在架構上不可能，locku 必須站在 tmux/screen 之外，由它們把 tty 交出來。

### 0.3 與既有工具比較

| | vlock | lock -np（BSD） | screen 內建 | locku |
|---|---|---|---|---|
| 驗證 | PAM 系統密碼 | 系統密碼 | 系統密碼 | 自家 PIN |
| 畫面 | 純文字提示 | 純文字提示 | 純文字提示 | 螢幕保護動畫 + popup |
| VT 切換鎖 | 有，需 root | 無 | 無 | 不做 |
| tmux 整合 | 手動設 lock-command | tmux 預設值 | 不適用 | 文件提供設定 |
| macOS | 無 | 無 | 有 | 有 |

PAM 是 Pluggable Authentication Modules，Linux/macOS 的系統驗證框架。VT 是 virtual terminal，Linux 的實體主控台 Alt-F1–F7。

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

- raw mode：關 ICANON、ECHO、ISIG。ISIG 關掉後 Ctrl-C、Ctrl-Z、Ctrl-\ 變成普通 byte 進到程式，不再產生 SIGINT、SIGTSTP、SIGQUIT。
- alternate screen：避免捲動緩衝區露出鎖定前的內容。
- 不開 mouse tracking：滑鼠留給終端機本身做文字選取，無害。
- **終端機的回答不是按鍵**（2026-09-28）：終端機回答別人問它的事（背景色、游標位置、它是什麼）走的是跟按鍵同一條輸入，鎖定畫面又把任何鍵都當成按鍵——回答一進來就開 PIN 框，沒設 PIN 就直接解鎖。tmux 在 client attach 的那一刻會問終端機背景色、DA1、DA2、XTVERSION，而 hook 把 attach 進已鎖 tmux 的 client 鎖起來是緊接在後（§6.2），回答就落到 locku 手上（實測 3.7c：鎖 17 ms 就沒了，每次）。所以兩條讀按鍵的路（畫布與 PIN prompt 經 Bubble Tea、custom 的 `Key`）都先過`internal/termreply`：OSC（`ESC ]` 加數字，到 BEL 或 `ESC \`）、DCS（`ESC P` 加數字、`>` 或 `!`，到 `ESC \`）、CSI 帶 `?` `>` `=`、或結尾是 `R` `n` `t` `c`、或 `$y`，整段丟掉；跨兩次讀取的也丟得乾淨。其他都是按鍵。單獨一個 `Esc` 在一次讀取的結尾就當場算數，不等下一個 byte；`ESC ]`、`ESC P` 在結尾則等下一次讀取（Alt-] / Alt-P 沒人在鎖上按，回答漏進來卻會解鎖）。Shift-F3 在部分終端機是 `ESC [ 1 ; 2 R`，跟游標回報同形，一起丟掉，按別的鍵就好。
- **鎖從空的輸入開始**（2026-09-28）：`locku lock` 一開始就 tcflush 終端機的輸入（`termreply.DropPending`），鎖出現之前按的鍵、還沒被讀走的，不算。另一個理由是只過濾不夠：Bubble Tea 在 `main` 之前（套件 init）就問終端機背景色，代它讀回答的 termenv 把 `ESC \` 當成到 `ESC` 為止；tmux 的回答先到時，兩句回答各被讀成一句、最後剩下一個 `\`，形狀不像回答，過濾器當它是按鍵（實測：鎖讀到 `\` 加游標回報就解鎖了）。清掉之後，之後才到的回答交給過濾器。

### 2.2 訊號表

| 訊號 | 來源 | 處理 |
|---|---|---|
| SIGINT / SIGTERM / SIGQUIT | kill 預設、ISIG | 忽略 |
| SIGHUP | tty 掛斷 | 忽略，交由 read EOF 決定是否結束 |
| SIGTSTP / SIGTTIN / SIGTTOU | Ctrl-Z、背景讀寫 | 忽略 |
| SIGWINCH | 視窗大小改變 | 接收並重繪；custom saver 再把新尺寸轉給程式的 pty（5.5） |
| SIGKILL / SIGSTOP | kill -9 | 無法攔截，範圍外 |

Bubble Tea 實作注意：預設會安裝 SIGINT/SIGTERM handler 讓程式結束，必須用 `tea.WithoutSignalHandler()` 並自行 `signal.Ignore`。Ctrl-C 會以 KeyMsg 進來，當成一般按鍵處理即可。

### 2.3 攔不到的東西（範圍外，README 要說清楚）

- ssh client 端的 `~.`：在本機端處理，byte 不會過線。
- Linux VT 的 Alt-F1–F7 切換、SysRq：需要 VT_LOCKSWITCH ioctl 與 root，vlock 的領域，不做。
- 另一條 SSH：`tmux attach -d`、`tmux kill-server`、`kill -9`。
- 終端機模擬器自身的快捷鍵：iTerm2 分頁切換、Cmd-W 等。

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

- 長度 4 到 12 字元（2026-10-08，使用者定案，決定 94；之前 4 到 64），換行、Tab 與其他控制字元以外的任何字元（2026-10-06：貼上的換行與 Tab 進得了框、畫得出來，但 Enter 不收；之前設定畫面照收、鎖定畫面打不進去，設得出一個永遠解不開的 PIN）。
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
- 連續錯誤冷卻，config 設定，預設關閉：連續錯 `wrong_pin_attempts` 次後進入冷卻 `wrong_pin_attempt_cooldown` 秒，期間 prompt 顯示剩餘秒數並吞掉所有輸入。`wrong_pin_attempts: 0` 即關閉。計數只在進程內存活，Esc 回 saver 不重置，冷卻結束才歸零，成功解鎖進程即結束。custom saver 也一樣：它的每個 prompt 是各自的 lock 程式，計數與冷卻的結束時間由 locku 一個接一個傳下去（2026-10-06；之前每次開 prompt 從零開始，冷卻可以用 Esc 繞過）。冷卻在 prompt 收起的時候結束，下次打開也歸零，跟在 prompt 上看著它結束一樣（2026-10-06；之前計數留著，下一次錯就又進冷卻）。

### 4.5 忘記 PIN：`locku pin reset`（2026-09-25，使用者定案）

前提照 0.1：locku 不是安全邊界，能用這個帳號執行指令的人本來就能 `pkill -9 -f "locku lock"`（進程結束 = 解鎖）或改 `pin_hash`。所以忘記 PIN 的路不假裝比帳號權限更安全，也不弱於它——它把「你自己從另一個 shell 把鎖處理掉」變成一個乾淨、有名字、有紀錄的動作。

- **`locku pin reset`**，從任何一個你自己的 shell 跑（另一個終端機視窗、SSH、lock-session 模式下 attach 別的 session）。問兩次：先 `Reset the PIN? … [y/N]`（只有 `y` / `Y` 算同意，其他一律 `left as it is`、exit 1），再 `Password for <user>:`——**登入密碼**，不回顯（帳號就是邊界，拿它當確認；靜態 binary 沒有 PAM，用 `su <user> -c true` 開在 locku 自己的 pty 上、等它印出密碼提示再把密碼打進去、看退出碼——輸錯 su 自己會拖時間；等提示 5 秒、等結果 15 秒，逾時或 su 沒問就當錯）。兩關都過才**產生一組新 PIN**（crypto/rand 八位數字）、把它的 bcrypt 覆蓋 config 的 `pin_hash`、在畫面上**顯示一次**（`New PIN: 12345678`，接著說寫進了哪個檔、log 在哪），之後到設定畫面用它換成自己要的——這招照 elasticsearch 的 reset password：永遠不把 `pin_hash` 清空、永遠不留一個開著的鎖。
- **紀錄**：每次 reset（成功或密碼錯被拒）在 `~/.locku/data/pin-resets.log` 追加一行 `<RFC3339> pin reset by <user>@<host>: a new PIN written` / `refused: the password is not the account's`，**不寫 PIN**；檔 600、目錄 700；目錄可用 `$LOCKU__DATA` 改。
- **鎖定中的 locku 每次按鍵先重讀一次 `pin_hash`**（只讀這個值、不重讀整份 config）：reset 之後回到被鎖的 client，新 PIN 直接能用、舊的 `wrong`；tmux 全域鎖著的每個 client 各自輸一次（維持 9/24 的各自輸）。檔案讀不到、解析失敗、hash 不是 bcrypt → 沿用記憶體裡的 hash（鎖定中檔案壞掉不能變成開鎖）；檔案裡 `pin_hash` 變空 → 視同無 PIN，任意鍵解鎖。preview 不重讀。custom saver 的 PIN 框同一套；它「沒設 PIN 就第一鍵結束」看的是啟動時讀到的 config（5.5）。
- 不跑的情況：stdin 不是 tty（`needs a terminal to ask on`，exit 2）；config 讀不到——解析失敗、`pin_hash` 不是 bcrypt、`profiles` 為空（`config.yaml: <原因> — fix that first`，exit 1）——不然會把預設值連新 PIN 一起寫回去蓋掉使用者的檔；沒有檔案可以跑（等於在預設值上設 PIN）。
- 否決：在鎖定畫面做「忘記密碼」入口（長按、特殊序列、安全問題——路人也能按，等於沒鎖）；恢復碼（比帳號權限弱的東西不值得多一套流程）；把 `pin_hash` 清空當 reset（會留一個無 PIN 的鎖）。可選的 PIN 提示（`pin_hint`）未做，使用者未定。

## 5. 螢幕保護內容

### 5.1 兩層：saver 出內容，畫布出畫法

已決（2026-09-24）：saver 只決定「顯示什麼」，畫布只有一種畫法。

- **saver** 是種類——class：clock、runner、bounce、snake、tetromino、custom。它決定怎麼產生內容，輸出不帶任何樣式：clock 是幾行 ASCII 文字；runner（2026-10-07 前叫 dino）是一張自己像素座標的點陣圖（2026-09-24 加入第二種），bounce 與 snake 也是（2026-10-06），tetromino 與 pets 也是（2026-10-07）；custom 不產內容，程式自己畫在 locku 給它的 pty 上，畫布只在它結束時接手（2026-09-25 加入第三種，5.5）。
- **profile** 是具名實例——object：一種 saver 加上它的參數與顏色，有名字；config 裡 `profile` 指向的、鎖定畫面顯示的，都是 profile（2026-09-24 定案，使用者以 OOP 分：class 不用取名、object 才有名字，能新增的是 profile、新增時先選 saver）。
- **畫布**把文字用 terminu family splash 的像素風格畫出來、把點陣圖依 size 放大鋪滿，依終端機格數自動選縮放，見 5.3。saver 碰不到顏色、字形、位置——自帶配色的 saver 例外：它說出自己的顏色、沒有 bg / fg（2026-10-06，使用者定案；目前是 runner、bounce、snake、tetromino 與 pets）。custom 不經畫布。

### 5.2 saver 與 profile

| type | 參數 | 內容 | tick |
|---|---|---|---|
| clock | `layout` row / column；`size` small / medium / large；`font` 3x7 / 3x5；`time` `HH MM` / `HH MM SS`；`date` off 或四選一；`bg` / `fg` 兩個顏色 | row：一列時間，date 不是 off 時第二列日期；column：依分隔符拆行，`HH` / `MM` / `SS`，日期再拆 `YYYY` / `MM` / `DD` | time 含秒為 1 秒，否則對齊整分每 60 秒 |
| runner（2026-09-24；2026-10-07 前叫 dino） | `participants` 跑者（2026-09-25 修訂，六選一；2026-10-07 前叫 `runner`）：`big`（一隻大暴龍 12 × 14）、`small`（一隻小暴龍 8 × 10）、`big-big` / `small-small` / `small-big` / `big-small`（兩隻一前一後，名字就是畫面由左到右的順序——左邊在後、右邊在前，各自跳各自的）；舊值 `trex` / `two-trex` 讀成 `big` / `big-small`，下次存檔寫新名；`character` 角色（2026-10-06，使用者）：`t-rex`（預設，就是原本的暴龍）、`cat`（貓 12 × 9 / 8 × 7；小貓 2026-10-08 前 8 × 6、沒有眼睛，見 95）、`rabbit`（兔子 12 × 13 / 8 × 10）、`giraffe`（長頸鹿 12 × 12 / 8 × 9；2026-10-07 前叫 `horse`，使用者：其實比較像長頸鹿）、`ghost`（小精靈的鬼 12 × 12 / 8 × 8，沒有腳，跑步是裙擺兩幀擺動），任何角色配任何隊形，大的是 participants 的 big、小的是 small；每個角色的寬跟同尺寸的暴龍一樣（12 / 8）、不比它高，跳躍弧與「跳得過」的時窗照舊；`scene` 場景：`grassland`（草原，障礙物是仙人掌）、`desert`（沙漠，障礙物是金字塔，沙地斑點較疏）、`city`（城市，2026-10-08：障礙物是平房、大廈、摩天大樓，夜裡一部分窗戶亮成金色，見 90）；`background` 背景（2026-10-07，使用者）：`day` / `night` / `time-shifting`（預設），內建，沒有 `bg` / `fg`——每種都是從畫面頂端到底端的漸層、鋪滿整個板子（使用者：不要用單一顏色）：day 頂端藍 `#7ab8f5` 往下到白 `#ffffff`（2026-10-08 前頂端是淡藍 `#cfe8ff`，見 91），跑者與整個世界畫成 `#313244`，雲是白的（2026-10-08，見 91）；黃昏的雲也是白的；night 是原本的配色，`#1e1e2e` → `#313244` → `#45475a`，地面線、障礙物、雲畫成月光的灰藍 overlay1 `#7f849c`（同日修訂，使用者：既然考慮了月光的顏色，金色的雲和障礙物也要換；原本金色 `#f2b753`）；跑者白天、黃昏是 `#313244`，夜晚是更深的 base `#1e1e2e`——比它跑過的夜空深，月光下的剪影（同日修訂兩次，使用者採用我的建議：原本夜晚也是 `#313244`，跟夜空中段同色，身體看不見、只剩外框；改成 crust `#11111b` 後，跟終端機的背景同色，方塊之間的縫隙不見了，身體糊成一塊，使用者：不是一格一格的）——外圍多一圈白色的外框（catppuccin 的 text `#cdd6f4`，偏冷的白，月亮是偏暖的 `#f5e0dc`；同日修訂兩次，使用者：月亮是白色，外框也該是白色，原本金色；外框跟月亮同色時，跳到最高會跟月亮黏在一起）與金色的眼睛——外框含角落、只畫在天空上、不蓋住經過的東西，所以站在地上時腳底下是地面線、看不到外框，跳起來時腳底下也有（同日修訂，使用者）；眼睛是角色圖裡標出來的那幾格（2026-10-07 修訂，使用者：原本夜晚整隻金色，黃昏到夜晚很突兀）；time-shifting 三分鐘一天，照牆上時鐘的分鐘除以 3：餘 0 白天、餘 1 黃昏（mauve → red → peach → yellow 的夕陽，畫成 `#313244`）、餘 2 夜晚；每一段的前 10 秒從上一段換過來（同日修訂，使用者：原本 20 秒）：一格一格換，從右邊往左推，同一欄裡哪一格先換是隨機的；每一格不是上一段就是這一段——天空、跑者與世界、太陽與月亮都是——沒有中間色，換過就不會換回去；之後 50 秒不變（2026-10-07 修訂兩次，使用者：先是到點整面直接換，有點突然、尤其夜晚到白天；改成整面漸變後，使用者：很順，但原本想的是直接一點，從右到左隨機格子換色，不要整面 fade）。白天的天空有太陽（`#f5c211`，7 × 7 的圓；2026-10-07 修訂，使用者：原本的 `#df8e1d` 有點橘，要偏黃一點），夜晚有月亮（`#f5e0dc`，7 × 7，照使用者的示意圖：左邊與下緣是太陽的圓，內緣斜切，上端往右上翹、右端的尖在第 4 列；同日修訂三次，使用者：上下一樣粗的 C 看起來像橢圓；扣掉跟太陽一樣大的圓之後，兩頭不夠尖、不夠寬；扣掉右上角的小圓之後，使用者畫了示意圖、標出要拿掉與要補的格子），黃昏是夕陽（`#fe640b`，太陽的上半、4 列高，平的那邊貼著地面，在白天太陽的正下方，障礙物從它前面經過；2026-10-07，使用者：黃昏也加一個夕陽）；太陽與月亮在右上方，都不跟著捲動，太陽在月亮左邊一點、月亮在最右邊，離跑者最遠（同日修訂：原本月亮在左，場景最窄時跟跑者跳到最高的外框只隔一格），雲從它們前面飄過；換的時候一個一格一格消失、一個一格一格出現，兩個都看得到。沒有 size（使用者：dino 也沒有 size 的選項），畫布自己取塞得下的最大倍率 | Chrome 離線小恐龍遊戲當螢幕保護：地面與障礙物向左捲、跑者自己跳過去，無限循環沒有人玩、不會死。障礙物隨機，分小 / 中 / 大三個等級（2026-09-25 修訂，使用者：原本只有小和中）：草原是小仙人掌 1 / 2 / 3 株（5 高）、中的高仙人掌（7 高）、大仙人掌（6 × 10，粗幹兩臂）；沙漠是小金字塔（3 或 4 高）、中金字塔（5 高）、大金字塔（13 × 7）、小加小；間距隨機 44 到 100 px；兩隻跑者各自看自己前面的障礙物、各自在自己的視窗裡隨機起跳，後面那隻的步伐差半步；跳躍在「跳得過」的那段視窗裡隨機挑一幀起跳，跳多高看前面那個障礙物的等級（2026-09-25 再修訂，使用者：現在高度都一樣）：小的低跳、中的中跳、大的高跳；前面沒東西時偶爾也無故跳一下，高度隨機三選一；雲以三分之一速度飄。不記分、不畫時間，畫面上只有場景（使用者 2026-09-24：dino 上面不需要計算時間和分數）。場景像素：跑者 12 × 14、跳躍弧三條、都是 16 幀，最高 6 / 8 / 11 px 對應小 / 中 / 大，各比該等級在兩個場景裡最高的障礙物（5 / 7 / 10）高一格，沙漠的金字塔較矮、同一條弧跳過去多留幾格（2026-09-25 再修訂，原本一條 11 px 跳所有東西；再之前 8 px，加高三格才跳得過大仙人掌）、每幀走 2 px，最小場景 40 × 28（同日修訂，原本 40 × 25：跑者 14 加跳 11 加地面 2 加一列天空） | 每 70 ms 一幀（14 fps），整張換、不做 reveal |
| bounce（2026-10-06） | `speed`：跟 snake 同樣五段，從清單選，預設 `normal`——每秒 7 / 10 / 14 / 20 / 28 px，normal 是原本的 10；`time`：`HH:MM` / `HH:MM:SS`，預設 `HH:MM`，畫出冒號（2026-10-07，使用者：bounce 沒有 size，所以給冒號；原本沒有參數、`HH MM`）；沒有 `bg` / `fg`：顏色是它自己的（使用者：多色的 saver 自帶配色） | 舊錄影機的螢幕保護：一個框斜著飄，每幀各方向走 1 px，碰到邊就反彈、換一個跟現在不同的隨機顏色；剛好同時碰到兩個邊（撞進角落）就把所有顏色快速閃兩輪，一幀換一色。框裡是時間（3x5 字型、跟 clock 同樣的字距，冒號 1 px 寬），跟框同色，框線 2 px 粗（2026-10-07，使用者：border 厚一點；原本 1 px），框線與字之間空 2 px，`HH:MM` 的框 25 × 13 px、`HH:MM:SS` 的 35 × 13 px（使用者：框裡放時間）。暗格 surface0 `#313244`，框的顏色是 splash gold 與 catppuccin-mocha 的 red、peach、yellow、green、teal、sky、blue、mauve、pink 十色。場景要三個框寬、三個框高（`HH:MM` 是 75 × 39 px），k 從 4 往下取，都不夠就 1、框照樣在裡面飄；比框還小的那個方向不動 | 每 1 ÷（每秒幾 px）秒一幀（normal 10 fps），整張換、不做 reveal |
| snake（2026-10-06） | `speed`：`slow` / `normal` / `fast` / `very-fast` / `super-fast`，從清單選，預設 `normal`（2026-10-07，使用者：輸入數字不太直觀；原本是每秒幾格自己輸入，1–30、預設 12）——每秒 5 / 7 / 10 / 14 / 20 格：normal 是使用者定的 7，其他每級約乘 1.4，一級比一級看起來快得一樣多；最快 20，低於畫面每秒 30 幀的上限（超過時 PIN 框的按鍵會等輸出）；沒有 `bg` / `fg`：每吃一顆就換顏色，所以顏色是它自己的，跟 bounce 同一組（暗格 surface0、蛇與果子用 gold 與 catppuccin 的九個亮色，蛇開局是純白、果子永遠不是白的（2026-10-07，使用者）；同日修訂，原本是 bg / fg 兩色、預設 Nokia 的兩種綠） | 老 Nokia 的貪食蛇，自己玩、不會輸：場景每 3 px 一格（一個節點、兩格連線），格子沿一條走遍全格的環（Hamiltonian cycle：從第二欄起一列一列來回走，最後沿第一欄回到起點；列數是奇數就把盤面轉 90 度，兩邊都是奇數就留下最後一欄不走）。蛇在環上，頭前面到尾巴之間永遠是空的；抄近路只抄到頭前面那段、不超過果子、離尾巴留蛇長加 3 格，所以蛇超過半個盤面就只照環走——保證填滿。每節一個點，相連的兩節之間亮兩格，並排但不相連的中間留暗。頭（使用者的圖）：閉嘴時節點與它後面那格連線上方各一格是頭頂，線的最前面多一格是鼻尖；吃一顆分三步（使用者的兩張圖：碰到、張嘴）：下一步就吃到時「碰到」——整個頭往果子推進一格，鼻尖貼著果子、頭頂跟著前移，頭朝著果子（轉彎吃的時候也是）；吃到的那一步「張嘴」——頭在果子那格，上顎三格蓋在果子上方往後、下顎兩格托在下方往後，果子還是自己的顏色、在兩顎之間，蛇身還是原來的顏色；再下一步「吞進去」——閉嘴，果子在頭的節點（喉嚨），頭的四周沒有別的（2026-10-07 修訂：原本頭頂後面身體鼓起一格，張嘴那一步也誤畫了出來，使用者：意義不明的格子；他重貼的完整順序裡吞進去那一步也沒有這一格）；橫著走頭頂朝上、直著走朝右，往右走是往左走的鏡像（同日修訂三次：先是 3 × 3 帶眼睛，使用者嫌太大；再是頭頂兩格加下巴；再是平常閉嘴、吃的時候下巴往前伸一格，使用者：怪怪的，改成上下顎；上下顎起初畫在頭原地、頭頂退後一節、節點空著，線的前端因此縮回去、離果子 4 格，使用者：好像中途蛇壞掉，要到豆子前一格才張嘴；再改成上下顎伸到頭和果子之間的連線上，使用者畫了碰到與張嘴兩張圖，成了現在的三步）。果子在剩下的空格隨機出現，用自己的顏色閃，0.32 秒亮、0.32 秒暗，吃下去也一路保持那個顏色：嘴裡、喉嚨（剛吞下時頭的節點）、身體裡的果子都是果子的顏色；包跑過尾巴，蛇才整條變成那個顏色（使用者：看得到豆子穿過身體；同日修訂，原本吃到就變色）。果子的顏色不跟蛇、也不跟任何還在肚子裡的包同色，所以它一路都看得出來——肚子裡同時有九顆以上時顏色不夠，才可能跟前面某一顆同色，前面那顆到尾巴、蛇換色後，這顆會有一段看不出來。吃下的果子在身體裡（使用者：進入腸道）：果子就在身體那條線上、那一節的節點，用果子的顏色，旁邊什麼都不多（2026-10-07 修訂，使用者：想錯了，突出來讓身體轉彎的格子拿掉，只留豆子的顏色和張嘴；之前身體在旁邊鼓起 3 格繞過果子、轉角從外側繞 5 格，更早是節點旁兩格、用果子的顏色）；它有自己的節拍（使用者：每秒幾格是蛇的速度，吞下去後在身體裡是固定的）：蛇走的時候它跟著身體走，另外每秒往尾巴挪一節，蛇的速度設多少都一樣（同日修訂：原本每一步往後兩個位置，在畫面上跟頭以同樣的速度反方向跑，眼睛跟不上）；一節只放一顆：咬到的那一步，嘴裡那顆算在頭那一節，還在喉嚨的那顆往後推一節、擋路的依序推下去；推到尾巴之外的那顆當場變成身體，蛇換成它的顏色、同一步長一節；自己的一秒到了、下一節還被前一顆佔著的，原地等它走（2026-10-07，使用者：擠在尾巴的豆子也處理，一節只放一顆；原本推到尾巴就停，會疊在尾巴，張嘴那一步喉嚨那顆也被嘴裡那顆蓋住）；到了尾巴再一秒，它變成身體：蛇整條換成它的顏色、尾巴同一步長一節（2026-10-07 修訂，使用者：換色和變長同時發生；原本照使用者的圖分三拍，換色後再一秒才長，中間還有鼓起剩一格）；填滿停住的 3 秒裡照樣走。蛇在豆子到尾巴、變成身體的那一步才變長（跟 Nokia 一樣；同日修訂，原本吃到就長）：尾巴停一步不跟著走；但頭這一步要走進尾巴那一格時，尾巴照走、晚一點再長，所以不會撞上自己。開局 3 節、隨機位置、純白（2026-10-07，使用者：蛇的顏色既然都是吃下的果子給的，開場就是純白；原本隨機一色）；填滿停 3 秒再開新局；場景換尺寸也是新局。場景最少 48 × 30 px（16 × 10 格，外加四周各 1 px 給鼻尖、頭頂與下唇），k 最多 2；格子置中 | 每 1 ÷（每秒格數）秒一步（normal 7 格／秒，約 143 ms；super-fast 20 格／秒，50 ms），每步只動頭、尾、包與果子；填滿停 3 秒、果子一閃 0.32 秒、吞下的果子每秒一節，跟速度無關；整張換、不做 reveal |
| tetromino（2026-10-07） | `speed`：跟 snake 同樣五段、同樣的數字，從清單選，預設 `normal`——每秒 5 / 7 / 10 / 14 / 20 步，一步是一個方塊，跟 snake 走一格一樣；沒有 `bg` / `fg`：顏色是它自己的（7 種方塊用慣例的顏色、換成 catppuccin 的：I sky `#89dceb`、O yellow `#f9e2af`、T mauve `#cba6f7`、S green `#a6e3a1`、Z red `#f38ba8`、J blue `#89b4fa`、L peach `#fab387`；消行的白 `#ffffff`、外框灰 overlay1 `#7f849c`、END 的紅 `#f38ba8`） | 俄羅斯方塊自己玩（使用者；名稱避開商標，用 tetromino，畫面與文件都不寫那個字）：不是一個直的井，畫面就是場地，全高、扣掉右邊的小框幾乎全寬；外框四邊都是 1 px（使用者；同日修訂兩次，原本四周 2 px，再改成只有底邊 2 px）。一個方塊 2 × 2 px（使用者：每個形狀以 2x2 為單位），所以 120 × 36 的終端機是 25 × 16 個方塊；不放大，終端機越大方塊越多。接下來的方塊在場地右邊排成一欄、一個一格，最上面是下一個（next），第二格是再下一個（next next），以此類推，外框的高度放得下幾格整格就幾格（120 × 36 是 6 格、80 × 24 是 4 格、200 × 50 是 9 格），最後一格的底線下面、到外框底部剩的空間塗成外框的灰，欄跟場地底部齊平（使用者，86；原本是地面）（使用者，同日修訂兩次：原本只有 next 與 next next，在場地左上、右上角挖兩個盒子、用場地的 2 × 2 畫，場地是盒子剩下的倒 T 形；再改成右邊上方兩個小框）：每格裡 6 × 4 px，方塊一格 1 px、照進場的方向、置中；框跟外框一樣灰、1 px，貼著場地的那一邊就是場地的外框線，相鄰兩格共用一條線；不寫字（照畫面不寫字的規則，靠位置分）。方塊從上框後面、正中間進場，最低那格在場地上面一列。7 種方塊 7 個一袋（每袋每種一個、順序隨機）。AI：從進場的位置往外找方塊走得到的每個停靠處（轉向、左右、往下，會繞過擋路的、鑽到懸空的下面），選落得最低、留下最少洞的那個（一個洞算落高兩列，滿的列先拿掉再算洞）；停靠處會凸出場地的是結局，比哪個都差。走法：每一步轉一次或橫移一格，能維持最短步數時同一步再往下一格，所以斜著過去；路是從目標倒推每個位置還差幾步算出來的，不會卡住。滿列：整列變白 0.28 秒，再從中間往兩端、兩邊同時一格一格消掉 0.4 秒，上面的整排往下掉，下一個方塊才進場。堆到頂（停下來的方塊有一格在場地上面）：所有方塊——場地裡的堆疊與右邊那一欄——從上到下、一列（1 px）一列變成外框的灰，5 秒掃完外框的高度，地面與外框不動（使用者；同日修訂兩次，原本整個畫面一列一列變黑，再改成所有方塊一起褪色），然後畫面正中間（整個畫面的，不是場地的）用 lock 自己的 3x5 字、一個字點一個方塊，寫紅色的 END，直接畫在灰色方塊上、不墊底，2 秒後開新局（使用者）。場景最少 29 × 22 px（10 × 10 個方塊——一般的井那麼寬——加右邊的小框），k 永遠 1；更小就什麼都不畫；外框在場景左上角，剩下不到一個方塊的空間在右邊與下面、是地面 | 每 1 ÷（每秒步數）秒一步（normal 7 步／秒）；消行與結束是自己的節奏：每 40 ms 一幀，跟速度無關；整張換、不做 reveal |
| pets（2026-10-07） | `animals`：是什麼動物，目前只有 `cats`；`count`：幾隻貓，`1`–`5`，從清單選，預設 `3`；`scene`：在哪裡，目前只有 `outdoor`（animals 與 scene 2026-10-08，見 89）；沒有 `bg` / `fg`：顏色是它自己的（背景一列一個顏色：天空從頂端的淺灰藍 `#7e9bbb` 往下漸深到一半高的 `#4c709d`（2026-10-08，見 92；原本是深藍 `#1d2745` 到灰藍 `#3f5a7c`），綠地從地平線的灰綠 `#4e6e46` 到底下的深綠 `#1c2f1a`；地面的草 `#4f7d36`、樹皮 `#847260`、樹葉 `#5e9140` 與葉影 `#3f6c2c`、樹樁的切面 `#b39b78`、睡覺的 z 正黃 `#ffff00`；貓的花色見內容） | 參考 vscode-pets 的玩法、圖自己畫（使用者）：貓在室外自己逛（使用者，87；原本在房間裡）。貓依序是橘紋（淺橘 `#f6b06a`、焦橘條紋 `#a8460c`）、琥珀（深咖啡 `#7a4a2a`、黃條紋 `#e8b830`、胸口與腳掌灰 `#9a958e`）、白（`#f4f1ea`、灰條紋 `#8c8a86`）、灰藍（`#9fb2c8`、深石板灰條紋 `#4e5d70`）、黑白（炭灰 `#55504c`，口鼻、胸口、肚子、腿、腳掌、尾巴尖白），`count` 是幾隻就取前幾隻；眼睛一格，在頭的前上方，綠 `#7ed957`、白貓藍 `#5fa8ff`。一隻貓走路 12 × 8 px（頭 4 × 4），坐著 9 × 10、睡著 12 × 5、爬 6 × 12，朝左時左右翻。室外每次重擺：背景是天空在上、綠地在下的漸層，地平線在一半高（使用者：照 runner 黃昏的做法，用成綠地，上面是天空）；地面一條草線，上面零星的草叢；地上是樹與樹樁，至少一棵樹（使用者：窄的畫面也要），放得下多少就放多少，樹的機會是樹樁的兩倍，順序與間隔隨機。樹：樹幹 3 px，從樹冠中間到地面，根在底下往兩邊多一格；樹冠在頂上，橢圓形的葉子帶葉影，寬 12 到 18、高 7 到 10，頂端在天空上面四分之一以內隨機；樹枝從樹幹兩側輪流長出來，一階比一階高（每階 6 到 8 px），同一側上下兩根之間、最低一根與地面之間都留得下一隻坐著的貓，長 10 到 14 px 隨機，往上長到樹冠下；樹幹兩側都能爬，從地面爬到那一側最高的樹枝，那一側沒有樹枝就爬到樹冠下。樹樁 12 到 14 × 5 到 7 px，樹皮的身體、淺色的切面在上，根在底下往兩邊多一格。樹與樹樁在貓的後面，貓會走在樹樁、樹枝前面。每隻貓挑一個隨機的目的地——地面、樹枝、樹樁頂、樹幹——避開別隻貓一隻貓寬，走最快的路過去：沿著走（遠的有時跑）、跳上（最高 13 px、最遠 16 px，從旁邊起跳、落在近的那端）、跳下（從端點、落在離開的那邊）、爬樹幹，從樹幹上也能跳到附近的樹枝或樹樁；到了就坐（偶爾翹尾巴）、趴著睡（頭上冒 z、一閃一閃往上飄）、在樹幹上就抱著，待一陣子再出發。場景最少 30 × 22 px（一棵樹、兩邊各一根樹枝、周圍的草地），k 永遠 1——貓永遠一樣大，終端機大就是室外大；更小就什麼都不畫 | 每 80 ms 一幀：走每兩幀 1 px、跑每幀 1 px、爬同走；跳最少 4 幀，越遠越久；坐 3 到 8 秒、睡 8 到 16 秒、抱 2 到 4 秒；整張換、不做 reveal |
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
- 新增（`[1]` 的 Savers 區塊在一種 saver 上按 `n`：要名字，預填 saver 自己的名字、用了就加號碼；以那種 saver 的預設值生出來）、duplicate（複製參數、要求新 name）、rename（連動 `profile` 指向）、delete。啟用中的不可刪，最後一個不可刪。
- profile 的 saver 建立後不可改：class 就是 class，要換就新增一個 profile（2026-09-24 定案；同一天曾短暫讓 type 可在 `[2]` 改，那是 dino 剛加進來、還沒有 New 時的權宜）。
- 七種 saver：clock、runner、bounce、snake、tetromino、pets、custom（2026-09-25；bounce、snake 2026-10-06；tetromino、pets 2026-10-07；runner 2026-10-07 前叫 dino）。前六種只是多一個產內容的函式，不動畫布；custom 不經畫布——程式自己畫，見 5.5。使用者自由輸入的 text saver 已移除（2026-09-24），內容不可控。

saver 預設值（2026-09-24，使用者定案）：每種 saver 在 config 的 `savers` 有一組預設值，欄位跟它的 profile 一樣、只是沒有名字。它決定**之後**用這種 saver 新增的 profile 長什麼樣，改它不影響任何已存在的 profile；cursor 在 saver 上時 `[p]` 就用預設值跑一個臨時 profile 預覽。內建值（config 沒寫時）：

| saver | 預設值 |
|---|---|
| clock | layout row、size large、font 3x5、time `HH MM SS`、date `YYYY-MM-DD`、bg `#313244`、fg `#f2b753` |
| runner | participants big、character t-rex、scene grassland、background time-shifting；沒有 bg / fg（2026-10-07） |
| bounce | speed normal、time `HH:MM`；沒有 bg / fg（2026-10-06；speed、time 2026-10-07） |
| snake | speed normal；沒有 bg / fg（2026-10-06） |
| tetromino | speed normal；沒有 bg / fg（2026-10-07） |
| pets | animals cats、count 3、scene outdoor；沒有 bg / fg（2026-10-07；animals 與 scene 2026-10-08） |
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

runner 的畫法（2026-09-24）：場景是整塊板，k 從 3 往下取第一個讓場景（40 × 28 px；2026-09-25 修訂，跳躍弧加高前是 40 × 25）塞得下的，都塞不下就 1；沒有 size 設定；場景 w × h = 板的格數 ÷ k，地面因此貼滿整寬，右邊 / 下面除不盡的格留暗。每幀整張換掉、不做 reveal——世界在移動，不是內容在變。14 fps 不是閒置，CPU 會比時鐘高，這是遊戲 saver 的代價。

會動的 saver 共用同一套（2026-10-06）：每種說出自己要的場景（最小幾 × 幾 px、最多放大幾倍），畫布照 dino 的作法取倍率；每格是一種墨，0 是暗格、其他是亮的，各一個顏色——clock 用 profile 的 bg / fg，其他用它自己的（runner 2026-10-07 起也是）；runner 的暗格還不只一個顏色：一列一個，從上到下是它的天空；pets 也是，天空在上、綠地在下（2026-10-07）。bounce 的場景是三個框寬、三個框高，k 最多 4；snake 最少 48 × 30 px，k 最多 2；tetromino 最少 29 × 22 px，k 永遠 1——一個方塊永遠 2 × 2 px，終端機大就是場地大；pets 最少 30 × 22 px，k 永遠 1——貓永遠一樣大，終端機大就是室外大。沒寫顏色或寫錯的 profile 拿它那種 saver 自己的預設色。

### 5.4 狀態列

已決（2026-09-24）：所有 saver 共用一行狀態列，內容 `user@hostname · 鎖定於 HH:MM`，user 是啟動 `locku lock` 的使用者。預設顯示，config `show_status: false` 可關。管多台 server 時靠它分辨機器與帳號，回來時知道離開多久。

4.3 的「未設定 PIN」提示與解析錯誤原因也在這一列，但不受 show_status 影響，永遠顯示。

### 5.5 custom：使用者自己的程式（2026-09-25，使用者定案）

使用者自己開發保護程式的動畫，locku 管其餘的——鎖、PIN、整合。profile 只有一個 `command`，`sh -c` 跑（可帶參數與 pipe），不做任何 sanitize：是使用者自己機器上自己的指令，README 寫明。只有換行與 Tab 不收：單行的值都不收（2026-10-06，ux.md §2.1）。

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
- profile 管理：new（從一種 saver）、duplicate、rename、delete、編輯參數（5.2）：clock 的 layout、size、font、time、date，runner 的 participants、character、scene、background，custom 的 command，snake 與 tetromino 的 speed，bounce 的 speed、time，pets 的 animals、count、scene；clock 再有 bg / fg 兩個顏色（runner 2026-10-07 前也有），各以 R G B 三個 slider 設定（webu slider 作法，數字清單不打字），config 存 hex。顏色走草稿：滑桿改的是草稿，`S` 才寫檔、`R` 丟掉草稿，其餘欄位立即寫檔（2026-09-24：使用者調歪過一次調不回來）。custom、runner、bounce、snake、tetromino 與 pets 沒有顏色，也就沒有草稿與 `S` / `R`。
- preference：啟用中的 profile（`profile`）、show_status、`pin_prompt_timeout`、`wrong_pin_attempts` / `wrong_pin_attempt_cooldown`。設為啟用在這裡，或側欄 profile 列按 `a`（2026-09-25，使用者：不必每次到 preference 切）。
- Integration（2026-09-25，使用者定案）：側欄第三個區塊，`tmux` 與 `screen` 各一項，`[2]` 的列：
  - **`activate`**（`on` / `off`）：區塊在不在 `config file path` 那個檔案裡、區塊讀的 locku 自己的檔案在不在（2026-10-08），每次畫都讀檔。Enter → confirm → 執行：on 把設定寫進 locku 自己的檔案、把讀它的一行（區塊）寫進檔案（tmux 有 server 在跑就把 locku 的檔案 `source-file` 進去；screen 連 shell rc 一起寫、跑著的 session 即時 `screen -X`），off 把區塊拿掉、刪掉 locku 的檔案（tmux server 上的、跑著的 screen session 上的一併拿掉）；路徑沒填時 disabled（2026-09-26 起只變暗，tdp M6）。
  - **`config file path`**：要寫的檔案，`~/` 可用，提議 `~/.tmux.conf` / `~/.screenrc`（webu 的提議作法，ux.md §2.1）。
  - 一條分隔線：上面是 locku 的設定，下面是寫進工具設定檔的 key。
  - tmux 的 **`lock`**：鎖的**範圍**，`lock-server`（預設，整台 server，鎖著時 attach 任何 session 都被鎖）或 `lock-session`（只鎖觸發的那個 session：它的 client 與之後 attach 它的人，別的 session 照常）。值用 tmux 的指令名，因為 alias 與 bind-key 最後跑的就是它；`?` 說明只講範圍、不講觸發方式（使用者：提到 bind-key 會誤導）。screen 沒有這列：每個 screen 是自己一個 process、LOCKPRG 跟著 shell，沒有範圍可選，不硬造。
  - 閒置鎖，**用工具自己的設定名稱**：tmux 是 `lock-after-time`、screen 是 `idle`（使用者：tmux 就用 tmux 的名字），各自一份、預設 300、0 關閉，config key 同名。
  - tmux 的 **`bind-key`** / screen 的 **`bind`**：prefix / C-a 之後按哪個鍵就鎖，照工具自己的寫法（tmux `l`、`C-l`、`F12`；screen `l`、`^L`），config key `tmux.bind-key` / `screen.bind`；有值就在區塊多寫一行 `bind-key <鍵> <lock>` / `bind <鍵> lockscreen`，空就不綁（screen 內建 `C-a x` 本來就是 lockscreen，`?` 要說）；含空白或 `#` 拒收。
  - **開一次，之後隨設即得**（使用者定案）：`activate` on 時任何一列改動就直接重寫 locku 自己的檔案（使用者的檔案不再動，2026-10-08）、tmux 整份套到 server、screen 送進跑著的 session；`config file path` 改路徑就把區塊從舊檔拿掉、寫進新檔，清空就拿掉；做完 toast 一行結果。off 就只寫 config.yaml，activate 仍由使用者開。兩個純方案各有硬傷：純隨設即得會在路徑打錯時生檔、也不能先填好再開；純按鈕會 stale——改了 lock-after-time 檔案裡還是舊值。
- 每個 `[2]`——profile、saver、tmux / screen、preference——第一列都是表頭 `Property` / `Value`，側欄區塊標題的 Blue，不可停（2026-09-25，使用者：所有 panel 2 都給標題列）。
- 每一列的意思在 `?`：focus 在 preference 或 tmux / screen 的 `[2]` 時，help **只有**那個面板的說明（自動換行），沒有鍵的清單；screen 多一條講 LOCKPRG 住在 shell rc；其他地方的 `?` 是鍵（2026-09-25）。
- 預覽：不驗 PIN，任意鍵回設定畫面（2026-09-25，PIN 是 `locku lock` 的事）。`[2]` 在 profile / saver 上 `P` 是那一個（帶顏色草稿）、在 preference / tmux / screen 上是啟用中的 profile；`[1]` 上 `p` 是游標那列，`P` 不作用。custom 的預覽把終端機交給程式（5.5）。
- 寫出 `~/.config/locku/config.yaml`，權限 600。

TUI 的版面與按鍵放 ui.md / ux.md。

### 6.2 Integration 怎麼寫檔

設定寫進 locku 自己的檔案——設定目錄裡的 `locku.tmux.conf` / `locku.screenrc`，跟 `config.yaml` 放一起——使用者的檔案只寫一個受管區塊，裡面是讀它的一行（2026-10-08，使用者定案，決定 93）。使用者的檔案在區塊外一個字都不動；再寫一次就是替換區塊，冪等。activate 開著時每改一列只重寫 locku 自己的檔案，使用者的檔案只在 activate 開、關（與 config file path 搬家）時動。先寫 locku 的檔案、再寫區塊，那一行不會讀到還不存在的檔案。

| 目標 | 使用者的檔案 | 區塊內容 | locku 自己的檔案 |
|---|---|---|---|
| tmux | Integration › tmux 的 `config file path`（使用者輸入，`~/` 可用，不存在就建；沒設 activate 就 disabled、說先填，不猜） | `source-file -q ~/.config/locku/locku.tmux.conf`（設定目錄不在家目錄下、或路徑有特殊字元時，是單引號包住的絕對路徑），尾巴 `# locku` 註解 | `locku.tmux.conf`：`set -gF lock-command "<locku 的絕對路徑> lock -S '#{socket_path}'"`、`set -g lock-after-time <lock-after-time>`、`set -s "command-alias[90]" "locku=<lock>"`、`set-hook -g "client-attached[90]" "run -C \"#{?#{@locked},lock-client -t #{hook_client},}\""`、`set-hook -g "client-session-changed[90]" "run -C \"#{?#{@locked},lock-client -t #{hook_client},}\""`；`lock` 是 lock-session 時再一行 `set-hook -g "session-created[90]" "set -F lock-command \"<絕對路徑> lock -S '#{socket_path}' -t '#{session_id}'\""`；`bind-key` 有填就再一行 `bind-key <鍵> <lock>`；每一行尾巴都有 `# locku` 註解 |
| screen | Integration › screen 的 `config file path`（同上），加上 shell rc：`$SHELL` 是 zsh 寫 `~/.zshrc`、bash 寫 `~/.bashrc`、fish 寫 `~/.config/fish/config.fish`、其他寫 `~/.profile` | screenrc：`source $HOME/.config/locku/locku.screenrc`（同上，不在家目錄下是單引號包住的絕對路徑）；shell rc：`export LOCKPRG=<絕對路徑>`（fish 是 `set -gx LOCKPRG <絕對路徑>`）；每一行尾巴都有 `# locku` 註解 | `locku.screenrc`：`idle <idle> lockscreen`；`bind` 有填就再一行 `bind <鍵> lockscreen`；每一行尾巴都有 `# locku` 註解 |

區塊標記：

```
# >>> locku >>>
...
# <<< locku <<<
```

- 路徑由使用者在 Integration 各項的 `config file path` 輸入：沒設時 `activate` 是 disabled，什麼都不寫（screen 連 shell rc 也不寫）；不猜路徑。相對路徑拒收，它會落在程式剛好執行的目錄。
- **tmux 有 server 在跑時，把整份設定交給它**（2026-09-25，使用者定案，見決定 39）：`tmux source-file` locku 自己的檔案——server 拿到的就是 tmux.conf 讀到的同一份文字，不另外維護一份指令清單（2026-10-08 前是把區塊的那幾行寫進暫存檔再 source，決定 93）。順序：先 undo 舊區塊做過、新區塊不做的事（檔案裡原本綁的鍵跟這次不同就 `unbind-key` 舊的；`lock` 換檔就 `set -gu @locked` 並拿掉 `session-created` hook——換檔後殘留的全域旗會讓所有 session 看似被鎖），再 source 區塊，最後對每個既有 session 逐一設它自己的 lock-command（lock-session：`set -t <id> -F lock-command "… -t '#{session_id}'"`；lock-server：`set -u -t <id> lock-command`）、換檔時再清每個 session 的 `@locked`——區塊裡的 session-created hook 只管之後建立的 session。activate off 時反向一條對一條拿掉：`set -gu lock-command` / `lock-after-time`、`set -su command-alias[90]`、三個 `set-hook -gu`、綁過的鍵 `unbind-key`、每個 session 的 lock-command 與 `@locked`、再 `set -gu @locked`。綁過與否看的是 locku 自己的檔案裡的 `bind-key` 行（舊版整塊寫進使用者檔案的區塊也讀），不另外記。unbind 之後那個鍵 tmux 內建的功能（例如 `l` 的 last-window）要 server 重啟才回來。沒有 tmux 或沒有 server 就跳過並說明。結果以 toast 一行回報（setup 印的幾行以 ` · ` 接起來）。
- tmux 那五行的道理（2026-09-24，使用者定案，全部以 pty 實測 tmux 3.7c）：
  - **預設不綁熱鍵**。用戶既然在用 tmux 就有自己一套 bind，`bind L` 會撞。改用 command alias：`prefix :` 然後打 `locku`，就是 `lock-server`（整台的 client 全鎖）；shell 裡 `tmux locku` 也一樣。要熱鍵的自己在 Integration › tmux › `bind-key` 填一個（2026-09-25，使用者：prefix shortcut），寫成 `bind-key <鍵> <lock>`——鍵是使用者選的，撞不撞他自己知道。
  - **每行尾巴 `# locku`**，加上受管區塊的頭尾標記，手動要移也認得出來；alias 與 hook 放在陣列的 90 號，不碰使用者自己的 0 號。
  - **lock-command 寫絕對路徑**：tmux client 是用它自己的 shell 環境跑 `sh -c`，`locku` 不一定在那個 PATH 上——找不到就是「畫面閃一下」（使用者 2026-09-24 實際踩到）。寫的是 `Binary()`：PATH 上的 locku（brew 的 symlink）優先，否則就是執行設定畫面的這個檔，跟 screen 的 LOCKPRG 同一套。所以從 repo 跑 `./locku` 開 activate，寫的就是 repo 那個 binary。
  - **lock-command 帶 socket**：lock-command 在 client 進程裡以 `system()` 跑，環境裡沒有 `TMUX`，被鎖的 client 也不在 `list-clients` 裡；`set -gF` 在讀檔時把 `#{socket_path}` 展開進去，`locku lock -S <socket>` 才知道要跟哪個 server 講話（`-L` 開的 server 也對）。標記是全域的，locku 不必反查自己在哪個 session。
  - **`@locked` 由 locku 設與清，全域**：解鎖沒有 hook（3.7c 的 MSG_UNLOCK 只清 flag），所以 `locku lock` 啟動時 `set -g @locked 1`、正常解鎖結束前 `set -gu @locked`，tty 消失不清；沒帶 `-S` 就什麼都不做。兩個 hook 看到 `@locked` 就 `lock-client`（attach 任何 session 與 switch-client 都驗過會觸發）。鎖的是**觸發 hook 的那個 client**，`-t #{hook_client}` 指名（2026-09-28）：不指名的 `lock-client` 鎖的是 tmux 執行當下認定的「目前 client」，兩個 client 同時 attach 時會拿到對方——進已鎖 session 的那個沒被鎖，lock-session 時別的 session 的反而被鎖（實測 3.7c：230 次錯 6 次；指名後 220 次錯 0 次）。`-t` 不吃格式，所以整句包在 `run -C` 裡先展開；`@locked` 也在同一次展開裡讀，沒有標記時 `run -C` 拿到空字串，什麼都不做、也不出訊息（實測）。`run -C` 比裸的 `lock-client` 晚一步：client 已經把它 attach 時的問題送給終端機，回答會落到鎖上——鎖不把它們當按鍵（§2.1）。activate off 順手 `set -gu @locked`。
  - **`lock-session`（2026-09-25，使用者定案；研究 `.local/studies/lock.md` §5）**：alias 與 bind-key 指向 `lock-session`，旗立在 session 上（`set -t <session_id> @locked 1`、解鎖 `set -u -t <session_id> @locked`），hooks 一模一樣——`#{@locked}` 先查 client 當下 session 的 option、沒有才 fallback 全域（實測），所以旗立在哪就是 scope。lock 程式怎麼知道自己是哪個 session：研究 §7 的「用 tty 反查 `display-message -p -c <tty>`」實測**不行**——鎖定中的 client 不在 `list-clients` 裡，`-c` 找不到就 fallback 到最近的 session，回錯的；改成每個 session 自己的 `lock-command`（它是 session option）由 `session-created` hook 在 session 建立時 `set -F` 烘入 `-t '#{session_id}'`，activate 時對既有 session 逐一設；`$0` 一定要加單引號，否則跑 lock-command 的 `sh -c` 會把它吃成自己的名字（實測收到 `-t sh`）。session 模式的定位是「各工作區獨立的視覺遮蔽」，不是安全隔離：`capture-pane -t 別的session` 不經 client、hook 不觸發（研究 §6）。研究 §7 的「解鎖一次全亮」輪詢不做，維持各自輸（9/24 已否決）。
  - lock 程式對 tmux 的呼叫（立旗、清旗）都有 2 秒 timeout、失敗一律靜默：不可能讓鎖起不來或掉下來。裸 tty、screen、沒帶 `-S` 時什麼都不做。
- screen 的 LOCKPRG 只能走 shell 環境（實測 2026-09-24，macOS screen 4.00.03，以探針程式經 pty 驗證）。原本想走 `.screenrc` 的 `setenv LOCKPRG` 一個檔搞定，實測不通：按 `C-a x` 出現的是 screen 內建的 `Key:` 鎖，探針沒被呼叫。原因是 `lockscreen` 由 attacher（接著終端機的前端進程）呼叫 `getenv`，而 `.screenrc` 只有後端讀、`setenv` 改的是後端與視窗內 shell 的環境；attacher 的環境在 `screen` 或 `screen -r` 執行那一刻就固定了。環境變數路線則完全符合設計：LOCKPRG 被 execl、`argv[0]` 是 `SCREEN-LOCK`、stdin 是 tty。所以 activate 寫 shell rc 的受管區塊，並提示：新開 shell 才有這個變數；已在跑的 session 不必重啟，detach 後從新 shell `screen -r` 即可，因為 attacher 是新進程。
- screen 也即時套到跑著的 session（2026-09-25，使用者定案：原理照 tmux 那邊的作法；全部以 pty 實測 macOS screen 4.00.03）：`screen -ls` 列出的每個 session（tab 開頭的 `pid.name` 行；exit code 不看——沒 session 時回 1）各送 `screen -S <pid.name> -X idle <秒> lockscreen`，有 bind 就再送 `-X bind <鍵> lockscreen`，attached 或 detached 都收得到；檔案裡原本綁的鍵跟這次不同，先送 `-X bind <舊鍵>`（不帶指令就是解綁）；activate off 反向 `-X idle 0`、`-X bind <鍵>` 一條對一條。跟 tmux 一樣 best effort：沒有 screen、沒有 session 就跳過並說明，哪個 session 不收就 toast 上一句、其餘照送。能即時套的只有 idle 與鍵，**LOCKPRG 套不進去**——它是 attacher 的環境，在 `screen` / `screen -r` 那一刻就定了；所以一個從沒有 LOCKPRG 的 shell attach 的 session，idle 到了或按了鍵，跑的是 screen 內建的 `Key:` 鎖，直到 detach 後從新 shell 重新 attach；toast 與 `?` 說明都講明這點。實測事實：區塊每行尾巴的 `# locku` 註解 screen 4.00.03 讀得過（`idle 2 lockscreen   # locku: 0 never`、`bind l lockscreen   # locku: …` 都生效）；`bind l lockscreen` 蓋掉 `C-a l` 原本的 redisplay；`-X source <rc>` 也能重讀整個檔，但沒用它——一條對一條才能反向拿掉；`SCREENDIR` 有效，e2e 靠它不碰使用者自己的 session。
- 絕對路徑偏好 PATH 上找到的那個（通常是 brew 的 symlink），不用解析 symlink 後的 Cellar 路徑，升級版本後才不會失效。
- 做完以 toast 回報：改了哪個檔、有沒有即時套用（screen：套到幾個跑著的 session）、還需要做什麼（screen：新開 shell；已在跑的 session detach 後從新 shell 重新 attach，LOCKPRG 才是 locku）。
- 不備份。移除就是 `[2]` 的 `activate` 關掉（confirm 後）：把受管區塊從檔案拿掉、locku 自己的檔案刪掉、有 server 在跑就一併拿掉；區塊前面補的空行也一起拿掉，其餘一個字不動；沒有區塊就說沒有；檔案不存在不會生出來。screen 同時清 screenrc 與 shell rc 的區塊，跑著的 session 也 `-X idle 0`、綁過的鍵 `-X bind <鍵>` 解掉（解掉之後那個鍵 screen 內建的功能——例如 `l` 的 redisplay——要 session 重開才回來，跟 tmux 一樣）。`[2]` 的 `activate` 列每次畫都讀一次檔案，說區塊在不在；區塊讀的 locku 自己的檔案不在時也是 off——一行讀不到東西就鎖不了，再開一次就補回（2026-10-08）。

## 7. 設定與儲存

只有一個檔：`~/.config/locku/config.yaml`

```yaml
version: 1             # 檔案格式的版本（2026-10-07）：沒有就是 0，讀到時轉成 1 並寫回
auth: pin              # v1 只有 pin，保留給 pam 擴充
pin_hash: "$2a$10$..."   # 空或缺欄位 = 未設定 PIN，見 4.3
profile: clock         # 啟用的 profile name，必須存在於 profiles
profiles:
  - name: clock
    saver: clock          # clock / runner / bounce / snake / tetromino / pets / custom：這個 profile 是哪一種 saver，建立後不改
    layout: row           # row / column（依分隔符拆行）
    size: medium          # small / medium / large：一個字型像素佔 1 / 2 / 3 格見方
    font: 3x7             # 3x7 / 3x5：字型高 7 列或 5 列，都是 3 格寬
    time: "HH MM"          # HH MM / HH MM SS（時分秒以空白分組，不畫冒號）
    date: off             # off / YYYY-MM-DD / YYYY-MMM-DD / MM-DD / MMM-DD
    bg: "#313244"          # 這個 profile 的點陣板暗格，預設 surface0
    fg: "#f2b753"          # 亮格，預設 splash gold
  - name: runner
    saver: runner         # runner（原 dino）沒有 layout / size / font / time / date：畫布自己取最大倍率；也沒有 bg / fg（2026-10-07）
    participants: big     # runner 才有：跑者，big / small / big-big / small-small / small-big / big-small（原 runner；舊值 trex / two-trex 自動轉）
    character: t-rex      # runner 才有（2026-10-06）：角色，t-rex / cat / rabbit / giraffe / ghost
    scene: grassland      # runner 與 pets 才有：場景，runner 是 grassland / desert / city，pets 是 outdoor（2026-10-08）
    background: time-shifting # runner 才有（2026-10-07）：天空，day / night / time-shifting
  - name: box
    saver: bounce         # bounce（2026-10-06）：沒有 bg / fg，顏色是它自己的
    time: "HH:MM"         # bounce 的 time（2026-10-07）：HH:MM / HH:MM:SS，畫出冒號
    speed: normal         # slow / normal / fast / very-fast / super-fast
  - name: nokia
    saver: snake          # snake（2026-10-06）：沒有 bg / fg，顏色是它自己的
    speed: normal         # snake 才有：slow / normal / fast / very-fast / super-fast
  - name: blocks
    saver: tetromino      # tetromino（2026-10-07）：沒有 bg / fg，顏色是它自己的
    speed: normal         # slow / normal / fast / very-fast / super-fast
  - name: cats
    saver: pets           # pets（2026-10-07）：沒有 bg / fg，顏色是它自己的
    scene: outdoor        # pets 的場景（2026-10-08）：目前只有 outdoor
    animals: cats         # pets 才有（2026-10-08）：是什麼動物，目前只有 cats
    count: 3              # pets 才有：1 到 5，幾隻貓
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
  runner:
    saver: runner
    participants: big
    character: t-rex
    scene: grassland
    background: time-shifting
  bounce:
    saver: bounce
    time: "HH:MM"
    speed: normal
  snake:
    saver: snake
    speed: normal
  tetromino:
    saver: tetromino
    speed: normal
  pets:
    saver: pets
    scene: outdoor
    animals: cats
    count: 3
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

版本（2026-10-07，使用者）：檔案的第一個 key 是 `version`，現在是 1。沒有 `version` 的檔是 0——在這之前的每一個檔——讀的時候先套 0 → 1 的轉換（上面 2026-09-24、09-25 的改名，與 2026-10-07 的 dino → runner），讀完立刻整份寫回、標上 `version: 1`；寫不回去（例如唯讀）就照讀到的跑，下次再試。是 1 的檔不再套舊名的轉換。比自己新的版本照讀、不寫回，免得蓋掉新版才有的資料；之後在設定畫面存檔，就寫成 1。寫回的是完整的設定：原本省略的欄位補上預設值、手寫的註解不會留著，跟在設定畫面存檔一樣。

- 無 history、無 cache、無 session。
- config 是 profile 的唯一來源，命令列不提供覆蓋。
- `profile` 指向不存在的 name、或 `profiles` 為空：用內建預設 clock，狀態列顯示 config error，不算損毀。
- 讀取失敗的處理見 4.3。
- 閒置多久自動鎖由各工具自己的那一列決定——tmux 的 `lock-after-time`、screen 的 `idle`，名字就是工具自己的（2026-09-25；舊的共用 `idle_lock` 自動轉），activate on 時原樣寫進去、一改就重寫；locku 自己不計時。

## 8. 安裝與整合（README 要交付的內容）

順序不限：不設定就是純螢幕保護，任何鍵解鎖；要密碼再執行 `locku` 設 PIN。

在 `locku` 側欄 Integration › tmux / screen 的 `[2]` 填 `config file path`、把 `activate` 打開（confirm 後）就直接寫進設定檔，做法見 6.2；開著時改任何一列就直接重寫。以下是它寫的內容，手動設定也是同一份：

tmux，Integration › tmux › config file path（慣例 `~/.tmux.conf`）拿到一個區塊、裡面一行：

```
# >>> locku >>>
source-file -q ~/.config/locku/locku.tmux.conf  # locku: the lock, as locku's settings screen sets it
# <<< locku <<<
```

設定在 locku 自己的 `locku.tmux.conf`，跟 `config.yaml` 同一個目錄：

```
# locku's tmux settings, as Integration > tmux on locku's settings
# screen sets them; ~/.tmux.conf reads this file. Set them there:
# this file is written over.
set -gF lock-command "/opt/homebrew/bin/locku lock -S '#{socket_path}'"  # locku
set -g lock-after-time 300                                                  # locku: 0 never
set -s "command-alias[90]" "locku=lock-server"                              # locku: prefix : locku locks every client
set-hook -g "client-attached[90]" "run -C \"#{?#{@locked},lock-client -t #{hook_client},}\"" # locku: attaching while locked locks the client
set-hook -g "client-session-changed[90]" "run -C \"#{?#{@locked},lock-client -t #{hook_client},}\"" # locku: so does switching sessions
bind-key l lock-server                                                      # locku: prefix l locks every client
```

最後那行只在 `bind-key` 有填時才寫（這裡填的是 `l`）；`lock` 選 `lock-session` 時 alias 與 bind-key 指向 `lock-session`，並多一行：

```
set-hook -g "session-created[90]" "set -F lock-command \"/opt/homebrew/bin/locku lock -S '#{socket_path}' -t '#{session_id}'\""  # locku: a session's lock knows its session
```

鎖：`prefix :` 打 `locku`（或 shell 的 `tmux locku`，填了 bind-key 就 `prefix l`）整台的 client 全鎖（lock-session 時只鎖這個 session 的），閒置 300 秒的畫面也鎖；鎖著的時候誰 attach 進來都會看到保護程式。移除：`activate` 關掉。

screen，Integration › screen › config file path（慣例 `~/.screenrc`）拿到同一套區塊標記與一行：

```
# >>> locku >>>
source $HOME/.config/locku/locku.screenrc   # locku: the lock, as locku's settings screen sets it
# <<< locku <<<
```

設定在 locku 自己的 `locku.screenrc`：

```
# locku's screen settings, as Integration > screen on locku's settings
# screen sets them; ~/.screenrc reads this file. Set them there:
# this file is written over.
idle 300 lockscreen   # locku: 0 never
bind l lockscreen   # locku: C-a l locks, as C-a x does
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

42. （2026-10-06，使用者定案）第四種 saver **bounce**：舊錄影機那種螢幕保護，一個框在板子上飄，撞到邊就反彈、換色，撞進角落把所有顏色閃一輪；框裡放 `HH MM`（使用者三選一：時間、只有框、LOCKU 字樣）。多色的 saver **自帶配色、沒有 bg / fg**，跟 custom 一樣不給設定（使用者三選一：自帶配色；否決「墨＋可選的調色盤」與「只用 bg / fg 取漸層」）；bounce 因此沒有任何參數。畫布為此改成每格一種墨、會動的 saver 共用 `saver.Game`，見 5.3。同一批定的還有貪食蛇與 dino 換角色（`character` 新參數，`runner` 照舊是隊形）。

43. （2026-10-06）第五種 saver **snake**：Nokia 的貪食蛇自己玩到填滿再重來（使用者 2026-09-27 的點子，同日選進第一批）。我的判斷（已告知使用者、可推翻）：沒有參數只有 bg / fg；預設色用 Nokia 的兩種綠，深的當底（淡綠當底實機看是一整面亮牆）；每格畫成節點加連線、果子會閃。走法是 Hamiltonian cycle 加安全的近路（只抄到頭前面那段空格、不越過果子、離尾巴留蛇長加 3 格），不會撞到自己，也一定填得滿；參考的作法（John Tapsell 的 Nokia snake AI）另有兩條規則——蛇超過半盤不抄、果子後面空間大時少抄——前者是多出來的（上面那條已經讓它抄不了），後者只影響快慢，都拿掉，240 格填滿從 9550 步變 9628 步。

44. （2026-10-06，使用者定案）dino 換角色：新參數 **`character`**——`t-rex`、`cat`、`rabbit`、`horse`、`ghost`，`runner` 照舊是六種隊形，任何角色配任何隊形（使用者二選一：新參數，否決把角色併進 runner 清單）；第一批四個角色使用者全選，圖先印給使用者看過才接（同日：貓的頭加寬一格、馬改成斜的長脖子）。太空船躲隕石沒有地面、不用跳，比較像另一種 saver，這批不做（我的判斷）。runner 列的 hint 從 `who runs` 改成 `one or two, big or small`，`who runs` 給 character。

45. （2026-10-06，使用者定案）snake 改版：每吃一顆就換顏色——果子先閃「下一個顏色」，吃到整條變成那個顏色（使用者三選一：果子是下一個顏色；否決「整條換、果子同色」與「新長的那節用新顏色」）；照 42 的規則多色就自帶配色，所以 snake 也沒有 bg / fg 了（使用者：照規則拿掉），43 的 Nokia 綠預設色作廢。加頭：使用者畫的 3 × 3、中間眼睛、下巴突出一格成張開的嘴；為了讓頭塞進並排兩段身體之間，格距從 2 px 改成 3 px（一個節點、兩格連線），頭頂與包都落在兩段之間的空處、不壓到任何節點，代價是格數約剩一半（152 × 31 從 608 格變 250 格）；直著走時頭朝右是我的判斷（把使用者的圖轉 90 度）。吃下的果子是身體裡的包，留在原地讓身體滑過、跟著尾巴消失（使用者：進入腸道；我的判斷：照 Nokia 原版，包固定在吞下的那一格）。

46. （2026-10-06，使用者定案）snake 再修：包沿著身體往尾巴跑（原本留在吞下的那一格），每一步在身體上往後兩個位置——身體往前長一節、包自己再跑一節；頭縮成兩格（使用者的圖：節點與後面那格連線上方各一格，下巴照舊在前、上面空著是嘴，沒有眼睛）。我的判斷：蛇照舊在吃到時就變長，不像 Nokia 等包到尾巴，因為那會讓尾巴在頭正要走進去的時候停一步、撞上；格距維持 3 px，頭頂與包跟並排的另一段之間留一列暗格（頭變小後 2 px 也放得下，格數多一倍，但會貼著旁邊那段）。

47. （2026-10-06，使用者定案）snake 的包縮成 2 格寬，跟頭頂同一個位置（原本 3 格寬、以節點為中心）。吃的時候張嘴：下一步就吃到果子、與吃到的那一步，下巴往前伸一格；其他時候閉嘴、頭的前面是平的。我的判斷：閉嘴是把下巴收回，不是把嘴上面那格補滿——補滿的話頭頂變 3 格寬，跟使用者要的 2 格不合；張嘴兩步（160 ms），看得出是咬一口。下一步走哪是定的（不帶亂數），所以畫的時候就算得出要不要張嘴，不另外記狀態。

48. （2026-10-06，使用者定案）snake 多一個參數 **`speed`**：每秒幾格，使用者在 number 框裡輸入（使用者：每秒幾格，讓使用者輸入格數；否決我提的 slow / normal / fast 三段）。我的判斷：範圍 1–30——30 幀一秒是畫面輸出的上限，再快 PIN 框的按鍵會等輸出；預設 12，接近原本的 12.5；檔案裡不合法的值靜靜當 12，跟寫錯的顏色一樣；填滿停的 3 秒與果子的閃爍用時間算、不跟著速度變。張嘴改成使用者畫的上下顎：下一步就吃到時，頭頂退後一節、原處是上唇、線的另一側是下唇、兩唇間的節點空著，鼻尖收掉；吃到的那一步閉嘴、頭頂後面鼓一格；其他時候閉嘴、鼻尖在前（47 的「平常收下巴、吃時伸出」作廢，使用者：怪怪的）。頭那一節的節點因此交給頭的形狀畫——張嘴時它是嘴。

49. （2026-10-06，使用者定案）張嘴要貼著果子：蛇一步跳一整格，「下一步就吃到」已經是離果子最近的一刻（果子在隔壁那格、中間隔兩格連線），所以不改時機、改畫法——上下顎伸到那兩格連線上，嘴是緊貼果子的那一格，線接到嘴後面不會縮短，頭頂只退一格（48 的頭頂退後一節、節點空著作廢：線的前端縮回去，看起來像蛇斷了）。

50. （2026-10-06，使用者定案）吃一顆分三步，照使用者的兩張圖：碰到（下一步就吃到，頭推進一格、鼻尖貼著果子）→ 張嘴（吃到的那一步，上顎三格、下顎兩格夾住果子，果子仍是自己的顏色、蛇身仍是舊色）→ 吞進去（下一步閉嘴、換色、頭頂後鼓一格）。我的判斷：不讓蛇停一步來咬，三個畫面剛好落在原本的三步上，速度不變；換色晚一步，張嘴時果子在嘴裡才看得出來；碰到的頭朝著果子而不是來的方向——蛇常是轉個彎吃到的，朝來的方向的話鼻尖會伸進空格；碰到的頭不畫身後那兩格連線（交給身體畫），轉彎時才不會多畫一段。49 的「上下顎伸到連線上、嘴是空格」作廢。

51. （2026-10-06，使用者定案）吃到果子不變色，果子也不變色，等包跑過尾巴蛇才變成果子的顏色——看得到果子穿過身體。包是那一節的節點加旁邊兩格，用果子的顏色畫；嘴裡的果子、剛吞下的那一格也是。我的判斷：新果子的顏色避開蛇與所有還在肚子裡的包，否則前面同色的包到尾巴、蛇換色後，這顆在身體裡就看不見；肚子裡九顆以上時十色不夠，接受那一小段看不見（只在盤面剩一成左右時出現）。50 的「張嘴那一步蛇身維持舊色、吞進去才換」作廢。

52. （2026-10-06，使用者定案）身體要繞過豆子：吃下的果子在身體那條線上（那一節的節點，果子的顏色），身體在旁邊鼓起 3 格、以果子為中心（蛇的顏色）——使用者的圖 `xxx` 在 `o` 上方；到尾巴後照 51 換色。我的判斷：吞嚥那一步一併改成同一個樣子——果子在頭的節點（喉嚨），頭頂後鼓起的那一格是蛇的顏色（原本那一格是果子的顏色）。

53. （2026-10-06，使用者定案）轉角要補格：果子在轉角那一節時，只在頭那一側鼓 3 格的話，身體接不到轉出去的那段連線、看起來斷了；改成從轉角外側繞過去——果子四周 8 格扣掉兩段連線與內角，其餘 5 格亮（使用者的圖：上方 3 格、外側往下 2 格接到往下那段）。直線照舊鼓 3 格；尾巴那一節只有一段連線，照直線畫。

54. （2026-10-06，使用者定案）吞下去的豆子有自己的節拍，每秒往尾巴一節，不跟蛇的速度（使用者：每秒幾格是蛇的移動速度，吞下去後在身體裡是固定一格的）；它跟著身體走，看起來是被蛇帶著、慢慢往後滑。到尾巴的收尾照使用者的圖分三拍、各一秒：在尾巴（鼓 2 格，不畫到尾巴之外）→ 變成身體（蛇換色、鼓剩 1 格）→ 鼓起消失、尾巴長一節。蛇因此在收尾時才變長。我的判斷：頭要走進尾巴那一格的那一步，尾巴不停、延後再長，否則會撞上自己（有測試造出這個局面）；一秒內吃兩顆時，新的把舊的往後推一節，一節只放一顆，否則兩顆疊在一起、舊的看不見。

55. （2026-10-07，使用者回報）bug：張嘴的那一步，頭後面多出一格蛇的顏色、跟哪裡都不相連（使用者：意義不明的格子，附截圖）。原因：嘴裡的果子位置記成 −2（`snakeBitten`），畫的時候夾成 0、被當成剛吞下，畫上了吞嚥時頭頂後面鼓起的那一格。使用者重貼的完整順序（閉嘴 → 碰到 → 張嘴 → 吞進去 → 在身體裡往尾巴走 → 到尾巴 → 變成身體 → 長一節）裡，吞進去那一步也沒有這一格，所以整個拿掉：剛吞下時只有閉嘴的頭和喉嚨裡的果子。50、52 的「頭頂後鼓一格」作廢。

56. （2026-10-07，使用者定案）身體不再繞過豆子：吃下的果子在身體裡只是那一節的節點換成果子的顏色，旁邊不鼓起——直線的 3 格、轉角的 5 格、尾巴的 2 格與 1 格都拿掉（使用者：我應該是想錯了，突出來讓身體轉彎的格子移除，只保留豆子顏色和張嘴）。52、53 作廢；54 的收尾三拍只剩顏色與長度：在尾巴 → 變成身體、蛇換色 → 再一秒尾巴長一節。豆子照舊停在節點、每秒一節，不在連線上一格一格滑（他重貼的順序裡那樣畫，他說是想錯了）。我的判斷：張嘴照現在的樣子，上顎 3 格、下顎 2 格（重貼的順序裡下顎畫成 1 格，他回覆「保留張嘴」）；收尾的節拍不動，只是不畫；格距維持 3 px，頭頂與上下顎還要用到兩段之間的空處。

57. （2026-10-07，使用者定案）豆子到尾巴一秒後，換色和變長同時發生：蛇換成它的顏色、尾巴同一步長一節（頭這一步要走進尾巴那格時照舊延後再長）；54 的收尾三拍與 56 剩下的兩拍作廢，`lump` 的 `gone` 拿掉。開場的蛇是純白 `#ffffff`（使用者：既然這樣安排了，開場的時候蛇就是純白色就好）——蛇的顏色都是吃下的果子給的，還沒吃過就是白的；白色只給蛇，果子永遠在原本的十色裡抽。我的判斷：白色是 `Inks()` 在十色之後多出的一個墨水，不放進 `ownColours`，bounce 不受影響；換新局（填滿或換尺寸）也回到白色。

58. （2026-10-07，使用者定案）豆子一節只放一顆，尾巴也是（使用者：擠在尾巴的豆子也處理一下）。原本推擠只在吞下後的下一步、推到尾巴就停，蛇很短又連吃幾顆時豆子會疊在尾巴那一節、只看得到一顆，然後各自的一秒到了就輪流換色；張嘴那一步，喉嚨那顆也會被嘴裡那顆蓋住一步。改成：咬到的那一步就推，嘴裡那顆算在頭那一節；推到尾巴之外的那顆當場變成身體，換色與變長跟 57 一樣同一步；一秒到了但下一節還被前一顆佔著的原地等。我的判斷：被推出尾巴的那顆提早變成身體（不等它在尾巴待滿一秒）——每一節都有豆子時新的那顆總得有地方放，排隊等的話嘴裡那顆就會疊在喉嚨那顆上。

59. （2026-10-07，使用者定案）snake 的 `speed` 改成從清單選五段：`slow` / `normal` / `fast` / `very-fast` / `super-fast`（使用者：讓使用者選就好，輸入數字不太直觀），48 的 number 框作廢。使用者原本給 5 / 7 / 9 / 11 / 13，我提每級加 2 越往上越看不出差別（slow 到 normal 快 40%，very fast 到 super fast 只快 18%），建議每級約乘 1.4；使用者同意，但 normal 要是 7、當預設，所以是 5 / 7 / 10 / 14 / 20 格／秒。設定檔存名稱（使用者採用我的建議，跟 `big-small`、`t-rex` 同樣的寫法）；snake 還沒發布過，檔案裡舊的數字或任何不認得的值都靜靜當 `normal`。預設從 12 變成 7，預設的蛇因此慢了一些。

60. （2026-10-07，使用者定案）dino 改名、拿掉 bg / fg：saver `dino` 改叫 **`runner`**，它的 `runner` 改叫 **`participants`**（值不變），角色 `horse` 改叫 **`giraffe`**（使用者：其實比較像長頸鹿；圖不變）。舊檔的 `saver: dino`、`savers` 底下的 `dino:` 與它們的 `runner:` 照樣讀進來，下次存檔寫新名；`horse` 沒發布過，不做舊名對應。runner 拿掉 `bg` / `fg`，改成內建的 **`background`**（使用者定的名字，否決我提的 `palette`）：`day`（背景白、前景 `#313244`）、`night`（原本的預設：背景 `#313244`、前景 `#f2b753`）、`time-shifting`（預設，使用者定）——三分鐘一天，白天、黃昏、夜晚各一分鐘。使用者另外要背景都有漸層、不要單一顏色，所以每種都是一道從畫面頂端到底端的漸層、鋪滿整個板子，time-shifting 到點直接換（使用者：漸層辦得到的話就直接換）。我的判斷：週期照牆上時鐘的分鐘除以 3（使用者同意），預覽與鎖定畫面因此一致；漸層一列一個顏色、在 sRGB 內插；顏色是我提的，等使用者看過再調——day 頂端淡藍 `#cfe8ff` 到白，night `#1e1e2e` → `#313244` → `#45475a`，黃昏是 mauve → red → peach → yellow 的夕陽、前景同白天。

61. （2026-10-07，使用者定案）config 加 **`version`**，這次是 1；讀到沒有 `version` 的檔就是 0，自動轉換、寫回並標上版本（使用者：這樣就可以做 config migration）。原本「讀到舊名當新名讀、下次存檔才寫新名」的轉換（`carryOver`）成了 0 → 1 這一步，只對 0 的檔做。我的判斷：轉換完立刻寫回，不等下次存檔——所以只是跑 `locku lock` 也會改寫檔案；寫不回去就照讀到的跑、下次再試，不在狀態列報錯；比自己新的版本照讀、不寫回，免得降版執行時蓋掉新版的資料；讀不了的檔（YAML 錯、PIN hash 不對）不寫回，維持原狀。

62. （2026-10-07，使用者定案）runner 的天空加上太陽和月亮：day 才有太陽、night 才有月亮（使用者）；time-shifting 改成漸變（使用者：直接切換有點突然，尤其 night → day），60 的「到點直接換」作廢。我的判斷：每一段的前 20 秒從上一段漸變過來、之後 40 秒不變，天空每一列、跑者與世界的顏色、日月都一起；日月固定在右上方、不跟著捲動（它們很遠），月亮在太陽左邊一點——放在同一處的話，淡入的太陽會把淡出的月亮整個蓋掉；雲在它們前面；黃昏沒有日月；太陽用 catppuccin latte 的黃 `#df8e1d`（mocha 的黃在白色天空上看不清楚），月亮用 rosewater `#f5e0dc`，圖是 7 × 7 的圓與月牙，等使用者看過再調。

63. （2026-10-07，使用者定案）黃昏也有夕陽（使用者），62 的「黃昏沒有日月」作廢。我的判斷：夕陽是太陽的上半（4 列），平的那邊貼著地面、在白天太陽的正下方，看起來是太陽落下去了；障礙物從它前面經過；顏色是 catppuccin latte 的橘 `#fe640b`，黃昏下半部的天空偏黃，太陽的黃在上面看不清楚；它有自己的墨，白天到黃昏時天上的太陽淡出、地平線的夕陽淡入，黃昏到夜晚時夕陽淡出、月亮淡入。

64. （2026-10-07，使用者定案）time-shifting 的過場改成一格一格換：從右到左、隨機的格子換成下一段的顏色，不要整面 fade（使用者：目前很順，但原本想的是直接一點），62 的「整面漸變」作廢。每一格不是上一段就是下一段——天空、跑者與世界、日月都是——沒有中間色。我的判斷：時間沿用 20 秒，從最右邊推到最左邊；一格什麼時候換看它離右邊多遠，再加上固定在那一格的隨機（佔 20 秒裡的三成），所以同一欄的格子錯開、帶狀前進，同一格每次都一樣、換過就不換回去，不會閃；交界的 20 秒裡日月兩個都畫，只在有它的那一段的格子裡看得到，一格一格出現、消失。

65. （2026-10-07，使用者定案）過場時間從 20 秒縮成 10 秒；太陽要偏黃一點（原本的 `#df8e1d` 有點橘）；月亮的輪廓像橢圓，要更正圓一點。我的判斷：太陽改成 `#f5c211`（色相約 47°，原本約 35°）——catppuccin 的黃都太淡，在白天淺藍白的天空上會看不見，所以用一個色盤外的、夠深的黃；月亮改成太陽那個圓扣掉往右 3 格、往上 1 格的同一個圓，剩下的部分沿著圓的左邊與下緣、上細下粗，最下兩列跟太陽一樣，看得出是同一個圓被遮住一塊（原本是上下一樣粗的 C，只像橢圓的左半）。

66. （2026-10-07，使用者定案）夜晚的跑者改成外框加眼睛：身體跟白天、黃昏一樣是 `#313244`，外圍補一圈金色外框、眼睛也是金色，腳底下不加外框（使用者：這樣黃昏到夜晚就不會那麼突兀）；月亮改成太陽缺了右上角的小圓，兩頭尖一點、寬一點（使用者）。我的判斷：外框是八個方向——角落也算，輪廓比較完整——身體之外、碰得到身體的每一格，眼睛那種被身體包住的洞也在內，最後一列以下不算；外框只畫在天空上，經過的仙人掌、雲不會被蓋掉；只有角色有外框，地面線、障礙物、雲夜晚照舊是金色；夜空中段剛好是 `#313244`，那一段跑者的身體跟天空同色、只看得到外框與眼睛，這符合「顏色是外框加眼睛」；過場時外框跟著夜晚的格子一格一格出現。月亮的小圓 5 × 5、放在太陽右上角往外一格的位置，剩下的兩個尖端各一格、最寬那一列跟太陽一樣 7 格。

67. （2026-10-07，使用者定案）月亮照使用者的示意圖（紅色的格子拿掉、綠色的補上）：

    ```
    ..##...
    .##....
    ##.....
    ###...#
    ####.##
    .#####.
    ..###..
    ```

    夜晚跑者的外框改成白色、跟月亮同色（使用者：因為月亮是白色），眼睛維持金色。我的判斷：眼睛直接在角色圖裡標成 `e`（畫面上跟空格一樣，夜晚才塗金色），不用「被身體包住的空洞」來認——T-Rex 跑步的第二格兩腿之間也有一格被包住，實機上那裡冒出一個金點；鬼的右眼斜角又通到頭外面，改用八個方向去認又會漏掉它。小貓的圖本來就沒有眼睛，夜晚只有外框（2026-10-08 補上，見 95）。

68. （2026-10-07，使用者定案）跑者跳到最高會碰到月亮，外框又跟月亮同色，看起來黏在一起（使用者：月亮放到左邊？還是外框換一點顏色？或者都調整？）。兩個都調整（我的建議）：月亮跟太陽對調，月亮放最右邊——場景最窄（40 格寬）時，跑者跳到最高的外框到第 18 格，月亮原本從第 20 格開始，現在從第 27 格，隔 8 格；太陽改到裡面，白天沒有外框，深色的跑者跟黃色的太陽不會黏；外框換成 catppuccin 的 text `#cdd6f4`，偏冷的白，月亮是偏暖的白，兩個都看得出是白、碰在一起也分得開。放到跑者左邊不行：那裡只有 6 格寬，月亮 7 格，T-Rex 的尾巴也在那裡。兩隻大跑者（big-big）在最窄的場景佔了第 6 到 33 格，月亮怎麼放都躲不開，那時靠外框的顏色不同，看起來是跑者從月亮前面經過。

69. （2026-10-07，使用者定案）夜晚跑者的身體改成 crust `#11111b`，深色剪影加冷白輪廓（使用者問外框樣式好不好、還是改回實心，採用我的建議）。原因：夜空的漸層中段剛好是 `#313244`，跟身體同色，跑者所在的高度身體幾乎看不見，像空心的線條畫；crust 比夜空最深的頂端 `#1e1e2e` 還深，任何高度都看得出形體，黃昏的 `#313244` 到它也都是深色，過場一樣順。我當時也說明了外框樣式的代價——夜晚的世界是實心金色、只有跑者是外框，小角色每邊多一格會變粗，兩腿、耳朵之間的空隙會被外框填掉——以及改回實心金色的選項。

70. （2026-10-07，使用者回報）夜晚跑者的身體用 crust 時糊成一整塊、看不出一格一格（使用者：是格子的顏色要深，不是整個內容都變 crust）。原因：板子的每一格是一個方塊字元，方塊之間的縫隙露出終端機自己的背景，使用者的終端機背景就是 crust（從使用者的截圖量到 `#11111a`），方塊跟縫隙同色就分不開。改成 base `#1e1e2e`：比 crust 亮一階，方塊分得開；比跑者在地上時背後的夜空（`#313244` 往下到 `#424456`）暗，還是剪影。我的判斷：跳到最高時頭會進到夜空最頂端，也是 `#1e1e2e`，那一下只靠白色外框描出輪廓——catppuccin 在 base 與 surface0 之間沒有別的顏色。終端機背景若是 base，身體一樣會糊在一起，但那時夜空頂端的暗格也一樣看不見，是整個板子的前提。

71. （2026-10-07，使用者定案）夜晚的世界——地面線、仙人掌、金字塔、雲——從金色改成月光的灰藍 overlay1 `#7f849c`（使用者問既然考慮了月光的顏色，金色的雲和障礙物是不是也要換，採用我的建議）：金色像燈光、不像月光；它比月亮與跑者的外框暗，主角比較清楚，又比夜空每一列亮；跑者的金色眼睛成了夜晚唯一的暖色；黃昏的深色世界換到灰藍，落差也比換到金色小。雲比地上的東西亮一階的做法使用者沒說要，先不做。跑者跳起來時腳底下也要有外框（使用者）：外框本來就只畫在天空上，站在地上時腳底下是地面線，所以 `rim()` 直接把腳底那一列算進來就好，不另外分站著、跳著。

72. （2026-10-07，使用者定案）bounce 加兩個參數、框線加粗：**`speed`**，跟 snake 一樣五段、從清單選（使用者）；**`time`**，`HH:MM` 或 `HH:MM:SS`，畫出冒號（使用者：因為 bounce 沒有 size——clock 當初不畫冒號是為了省寬度）；框線從 1 px 加粗到 2 px（使用者：border 給厚一點）。我的判斷：speed 的數字讓 normal 維持原本的每秒 10 px，其他照 snake 的每級約乘 1.4（7 / 10 / 14 / 20 / 28，最快不超過畫面每秒 30 幀）；time 的值叫 `HH:MM` / `HH:MM:SS`，跟 clock 的 `HH MM` / `HH MM SS` 分開，檔案裡寫成 clock 的格式或其他不認得的值都當 `HH:MM`；框線加粗成 2 px；框因此是 25 × 13（`HH:MM`）或 35 × 13（`HH:MM:SS`）。bounce 沒發布過，不做舊值轉換。

73. （2026-10-07，使用者定案）第六種 saver **tetromino**：自己玩的俄羅斯方塊（使用者：tetris 有商標，換成 tetromino；跟 snake 一樣自己玩）。使用者定的：屬性只有 `speed`，跟 snake 一樣用選的；範圍是全寬——不是直的井，整個畫面、2 格外框；填滿整列自己消除，要有消除特效；每個形狀以 2 × 2 為單位；左上角、右上角各一個空間放 next 與 next next；不小心堆到頂要有結束效果：全部從上到下變黑、紅字 END、2 秒後重新開始（取代我原本提的「整面用消行的特效清掉、接著玩」）。我的判斷（已告知使用者、可推翻）：外框灰 overlay1 `#7f849c`（像牆；金色會跟黃的 O、橘的 L 搶）；7 種方塊用慣例的顏色、換成 catppuccin 的；7 個一袋；方塊從開口正中間、上框後面進場；消行是整列閃白再從中間往兩邊消，跟速度無關的固定長度；盒子裡不寫 next 字樣；速度用 snake 那組數字（一步是一個方塊，跟 snake 一格一樣），normal 7；不放大；開口那幾列只要開口填滿就算滿列；結束的黑用 catppuccin 最暗的 crust `#11111b`（75 改成 base）、紅用 catppuccin 的 red `#f38ba8`、變黑 1 秒（75 改成 5 秒）、END 用 lock 自己的 3x5 字、一個字點一個方塊、放正中間；變黑原本只蓋場地、外框與盒子的牆留灰，實機在最小的場地（80 × 24 的終端機）上 END 比開口寬、盒子的牆從字中間穿過去，所以照使用者說的「全部」連外框一起變黑。AI 的評分：先用常見的四項（總高、洞、凹凸、消行），實跑方塊全堆在先到的那個角落、疊進開口——場地這麼寬，放哪裡總高都一樣多 4；換成 Dellacherie 的六項就正常，再一項一項拿掉量，只留「落多低」與「洞」兩項一樣好（28 × 15 的場地 14 分鐘最高 5–6 列、洞最多 2 個、從沒堆到頂），只留落多低則到處是洞、堆到頂，所以留這兩項。已知：場地小時常堆到頂——開口窄，方塊很難塞到盒子下面；14 分鐘裡 18 × 9（80 × 24 的終端機）約 17 次、18 × 13 約 1–4 次、28 × 9 約 0–1 次、28 × 15 以上 0 次。我提了兩個方向——小場地拿掉盒子讓開口全寬，或提高最小尺寸、太小就不畫——並建議接受現況，因為堆到頂本身有結束畫面，看起來就是一局正常結束；使用者同日定案：接受現況，README 用使用者的話寫明。76 把盒子搬到場地外之後就不再發生，這條與 README 那句限制一起作廢。

74. （2026-10-07，使用者定案）tetromino 的外框只有底邊 2 px，上、左、右改 1 px（使用者：border bottom 才給 2 格，其他都給 1 格就好）。我的判斷（已告知使用者、可推翻）：盒子的牆也算「其他」，一起改 1 px；方塊的格線不動，所以牆所在的那個方塊只畫貼著盒子內側的 1 px，另 1 px 是地面；結束時變黑改成一列一列（1 px）往下，不再以方塊為單位，因為上框只剩 1 px。場地因此變大：120 × 36 的終端機從 28 × 15 變成 29 × 16 個方塊，最小場景從 40 × 22 變成 38 × 21 px。外框原本會把剩下不到一個方塊的空間平分到兩邊，其實永遠是 0，拿掉，剩下的一律在右邊與下面。

75. （2026-10-07，使用者定案）tetromino 的結束：變黑太快（使用者：給 5 秒），從 1 秒改成 5 秒，之後 END 照 73 停 2 秒——我把「動畫」讀成變黑的那段；END 時格子不見了（使用者：要用格子呈現 END）——原因是黑用的 crust `#11111b` 跟使用者終端機的背景 `#11111a` 幾乎同色，方塊融進背景，跟 runner 夜晚的身體同一個問題，所以一樣改用 base `#1e1e2e`，格子看得出來，END 的紅方塊照舊。

76. （2026-10-07，使用者定案）tetromino 的 next 與 next next，從場地左上、右上角挖出來的盒子，搬到場地外、右邊上方的兩個小框，上下疊。使用者覺得盒子太大，提了三個方向：縮成 1x1、移除、或移到框外；採用我的建議——移到框外、用 1x1——再加上使用者的：小框也要有框，貼著場地的那一面就用場地的框，兩個放同一邊。我建議的理由：場地變回完整的長方形，原本窄開口讓方塊很難塞到盒子下面、小終端機常堆到頂的問題跟著消失（量過：80 × 24 到 200 × 50 的場地 14 分鐘都 0 次；80 × 24 是 15 × 10 個方塊、最高 8 列），盒子相關的程式（`boxed`、開口、開口那幾列的消行、鑽到盒子下面）也拿掉；預覽一格 1 px，小框只佔 7 px 寬。我的判斷（已告知使用者、可推翻）：放右邊（俄羅斯方塊的慣例）、next 在上；小框裡 6 × 4 px（最長的 I 加四周 1 px）、四邊 1 px；結束時整個畫面變黑、END 在場地正中間；最小場地改成一般井的 10 × 10。代價：場地窄了 4 個方塊（120 × 36 從 29 欄變 25 欄）。

77. （2026-10-07，使用者定案）tetromino 右邊那一欄做成整欄的佇列（使用者：既然都佔了一欄，不如做到一整欄，最上面是 next、第二格是 next next，以此類推；我同意——不多佔寬度，現代的俄羅斯方塊也顯示 5 到 6 個）。我的判斷（已告知使用者、可推翻）：每格跟 76 的小框一樣大（裡面 6 × 4 px、相鄰共用一條線），外框高度放得下幾整格就幾格（120 × 36 是 6 格），欄的外框畫到最後一格的底線為止，剩下不到一格的空間是地面——不把欄畫到地板、在最下面留一個不滿一格的空框，那看起來像少了一個方塊；方塊照樣 7 個一袋發，6 格大約就是一整袋。

78. （2026-10-07，使用者定案）tetromino 的結束與外框再修（使用者看了只為看結束而做的版本——每局開場就堆到只差一列——說效果不對）：結束不是整個畫面一列一列變黑，是內容慢慢 dim 成外框的顏色——所有方塊（堆疊與右邊那一欄）一起在 5 秒內褪成外框的灰 `#7f849c`，地面與外框不動，73、75 的變黑與 base 色作廢；END 放在整個畫面的正中間，不是場地的；外框底邊也改 1 px——右邊多了那一欄，底邊 2 px 沒有意義了，74 作廢。我的判斷（已告知使用者、可推翻）：「慢慢 dim」做成顏色一起漸變（不再從上到下）；褪色是每一幀把 7 種方塊色往灰色混一點（`Inks()` 依進度回傳），所以畫面上的格子不動、只有顏色變；END 底下墊一塊地面色的底板、四周留一個字點，因為那時場地多半已經是滿滿的灰方塊，紅字直接壓上去，筆畫之間的灰格會讓字難讀；外框四邊 1 px 之後 120 × 36 的場地仍是 25 × 16 個方塊，最小場景 29 × 22 px。

79. （2026-10-07，使用者定案）tetromino 的結束再修：不是褪色，是從上到下變色——所有方塊從上往下一列一列變成外框的灰，5 秒掃完（78 的「一起褪色」作廢）；END 出現時不需要底板，因為 END 是紅色（使用者），78 的地面色底板拿掉。我的判斷（已告知使用者、可推翻）：一列是 1 px（一個小方格），不是一整個 2 × 2 方塊，所以掃過時一個方塊會短暫上灰下彩；掃的範圍是外框的高度，右邊那一欄的方塊在上面，先變灰；變灰的做法是把掃過的列裡所有亮著的格子換成外框的灰——那時亮著的只有方塊與本來就灰的外框——`Inks()` 回到固定的顏色。

80. （2026-10-07，使用者定案）第七種 saver **pets**：參考 vscode-pets 的玩法，圖自己畫（它的素材是幾位畫師授權的）。使用者定的：屬性是寵物數量；背景有給寵物攀爬的物件；寵物隨機在畫面中移動。照我的建議：先只做貓、室內、我先畫一版，數量 1 到 5、預設 3（屬性 `count`）。我的判斷（已告知使用者、可推翻）：名稱叫 `pets` 不叫 `cats`，之後加別的動物不用改名；每隻貓挑隨機的目的地、走最快的路過去（房間是一張「地點加上跳法、爬法」的圖，找路用 Dijkstra），到了坐、睡（冒 z）、在柱子上抱著；家具在貓的後面，貓會走在箱子、板子前面，坐在箱子前面看起來像坐在箱子裡（使用者接受）；80 ms 一幀，走每秒約 6 px、遠的有時跑、爬同走，坐 3 到 8 秒、睡 8 到 16 秒、抱 2 到 4 秒；家具不放大，終端機大就是房間大。

81. （2026-10-07，使用者定案）窄寬也要能配適（使用者在 37 × 32 px 的 pane 裡什麼都沒看到，最小原本是 40 × 20）：最小降下、靠牆的留空跟著寬度縮；箱子窄到兩邊沒地方起跳時，貓從牆上跳上去（我的判斷：貓從牆上跳到附近的平台、再跳回牆上，大房間也會）。最小與窄的畫面在 84、85 再修。

82. （2026-10-07，使用者定案）顏色。貓只有幾種花色、不是隨機（使用者）：橘紋、琥珀、白、灰藍、黑白，`count` 取前幾隻（我的判斷：照順序，不從五隻裡隨機挑）。房間要像房間（使用者）：家具是木頭色；背景先改成柔和的沙色，使用者在自己的終端機上看——每個像素是小方塊，中間是黑縫——貓全淹沒了，改回深色（使用者），用深棕 `#2b231e`（我的判斷）。貓雙色、兩色對比要高（使用者）：橘紋淺橘配焦橘條紋；琥珀深咖啡多、黃少、一點灰（使用者）；白配灰、灰藍配深石板灰（我的判斷）；黑白的白多一點（使用者），口鼻、肚子、腿、尾巴尖也白，黑用炭灰——純黑在深色地面上看不見（我的判斷）。一格不同顏色的是眼睛（使用者問）：原本頭只有 3 × 3、眼睛在頭的後下角，看起來在脖子上，使用者說格子不夠就加大——貓從 8 × 6 加到 12 × 8、頭 4 × 4，眼睛在頭的前上方，家具與跳的高度、遠近跟著放大；眼睛不跟背景同色、不跟 z 的黃撞色（使用者）：綠、白貓藍（我的判斷）。睡覺的 z 正黃 `#ffff00`（使用者）。橘貓跟木頭顏色太近（使用者）：家具改偏灰的胡桃木，深 `#6e6052`、淺 `#a39484`（我的判斷：橘貓是使用者定的花色，換的是木頭；也避開琥珀的深咖啡）。

83. （2026-10-07，使用者定案）牆上要有攀爬物（使用者）：兩面牆各一根從地板到頂的麻繩柱，貓抱著它爬（我的判斷，另一個選項是牆上釘踏板）；麻繩要有斜的紋路（使用者）：淺麻 `#d6bc8a` 裡夾深麻 `#9c8158`，每 3 列錯一格，兩根同一個方向（我的判斷）。牆上要有平台（使用者：只有牆、牆上沒有攀爬物）：柱子上架層板，貓從柱子踩上去；層板要有三角支架，不是從牆上突出一個平台（使用者）：每塊下面一根深木色斜撐，跟柱子、層板圍成三角形。

84. （2026-10-07，使用者定案）牆上的層板要有高低（使用者：怎麼都是低的）、不一定都那麼長（使用者）、窄的畫面也要有（使用者）：每面牆放得下幾塊就幾塊、有時少一塊，剩下的高度隨機分到上、中、下，兩面牆錯開；長度 10 到 14 px 隨機，撐桿是層板過柱子那段的一半；地上的東西一律讓開層板，層板最長的寬度跟著畫面縮——14，或縮到中間剛好放一個箱子，最短 10（我的判斷）。

85. （2026-10-07，使用者定案）pets 的最小場景 30 × 22 px：兩面牆、各一塊層板（上面坐得下一隻貓、下面也是）、中間的地板；放得下才放貓跳台與箱子（我的判斷）。之後加別的動物的做法（我的建議，使用者同意先發這一版）：圖、大小、能力（會不會爬、跳多高多遠、步調）與花色收成一份「種類」，連結記下是爬還是跳、多高多遠，找路與到得了的地方依種類跳過做不到的；設定多一個選動物的屬性，沒寫就是貓，不用升 config 版本。

86. （2026-10-07，使用者定案）tetromino 右邊那一欄，最後一格的底線下面如果跟外框底部之間有空間，用外框的顏色補滿（使用者）：欄的寬度、從最後一格的底線到外框的最後一列，整塊外框灰；欄外與外框以下照舊是地面。

87. （2026-10-07，使用者定案）pets 搬到室外（使用者：現在寵物在室內，我想要給他們去室外）。背景參考 runner 黃昏的漸層，用成綠地的漸層，上面是天空（使用者）：一列一個顏色，天空從深藍到灰藍、綠地從地平線的灰綠到底下的深綠，地平線在一半高（我的判斷，使用者看過說天空與地平線照現在；顏色走暗，因為 82 的經驗——亮的背景淹沒貓）。攀爬物變成樹，不一定在左右兩邊，窄的畫面也至少一棵（使用者）：樹幹兩側可爬、樹枝左右輪流長出來當平台、頂上一團樹冠（我的判斷）；箱子換成樹樁（我的判斷，使用者說留著）。83、84 的牆、麻繩柱、層板與支架，80 的貓跳台、箱子與房間的顏色，85 的最小場景的內容，都由這條取代（大小仍是 30 × 22 px）。

88. （2026-10-07，使用者定案）PIN 框的點點要在垂直的正中間（使用者：輸入位置沒有垂直置中）：tdp F7 的錯誤列緊接在點點底下、下面再一列空白，點點因此高了一列。改成連上下框 7 列——兩列空白、點點（第 4 列）、一列空白、錯誤列（第 6 列）貼著下框（使用者定的排法）；鎖定畫面的 PIN prompt 與設定畫面的三個 PIN 框一起改（2026-09-24：PIN 在哪裡打都長一樣）。使用者先提錯誤改用另開的 note popup，我指出 tdp F7 明寫錯誤列留在框裡、不另開 popup（使用者修正時錯誤一直看得到、不必多按一次鍵關掉），因此不採用。畫法見 `ui.md` §3.2。

89. （2026-10-08，使用者定案）pets 加兩個屬性，為之後鋪路：`scene`，目前只有 `outdoor`；`animals`，目前只有 `cats`（使用者定的名字與值）。`scene` 跟 runner 共用同一個 key，各自的值分開——pets 的選單只列 `outdoor`，hint 是 `where they are`（runner 仍是 `where it runs`）。`[2]` 的順序是 animals → count → scene，「什麼、幾隻、在哪」（我的建議，使用者同意）；檔案裡照欄位順序寫成 scene、animals、count。舊檔的 pets profile 沒有這兩個值，或值不認得（例如 runner 的場景），都當作 `outdoor`、`cats`，不升 config 版本；其他 saver 的 `animals` 一律拿掉。畫面不變：目前只有一種選擇，saver 不讀這兩個值。

90. （2026-10-08，使用者定案）runner 加第三個場景 `city`（使用者）：障礙物是平房（小）、大廈（中）、摩天大樓（大）（使用者），照原本的三種大小與跳躍——小的最高 5、中的 7、大的 10，都不超過 13 寬。平房 11 × 5，三列屋頂、兩扇窗一扇門；大廈 9 × 7，每層四扇窗、底下一扇門；摩天大樓 9 × 10，往上收窄兩階到天線（我畫的，使用者看過說形狀可以；摩天大樓另畫了一層層的窗、直條玻璃帷幕兩種比較，選了收窄這種，最像摩天大樓）。每種只有一個樣子（草原與沙漠的小障礙有並排的），地面的斑點比草原疏、比沙漠密。窗戶透出天空；夜裡整個世界是月光的灰藍、樓偏暗，所以一部分窗戶在夜裡亮成跑者眼睛的金色 `#f2b753`（我的建議，使用者在全暗、全亮、亮一部分裡選了亮一部分）：哪幾扇亮固定在圖上、不閃（圖裡標 `w`），用的是眼睛的墨——白天與黃昏沒有眼睛的顏色，窗戶照舊是天空，time-shifting 換場時跟著一格一格換。

91. （2026-10-08，使用者定案）runner 的雲換顏色：白天與黃昏都是白 `#ffffff`、沒有框，夜晚照舊是月光的灰藍。原本雲跟地面、障礙物同色，白天是深灰 `#313244`。使用者先要白天學夜晚跑者的外框——雲身白、外面一圈灰框，黃昏直接白、沒有框；灰框做出來（latte 的 overlay0 `#9ca0b0`，我挑的）使用者看過說有框是錯誤，白天也直接用白色，框整個拿掉。雲有自己的墨，每種天空各自定義，time-shifting 換場時跟著一格一格換；雲照舊在太陽前面。白雲在白天天空頂端的淡藍 `#cfe8ff` 上看不太出來（使用者）——兩個一樣亮；我提了四種：照舊、淺灰的雲 `#ccd0da`、上白下淺灰 `#bcc0cc`、雲照舊白但天空頂端調深，使用者選最後一種：白天天空頂端改成 `#9ccfff`、看過再要深一點，成了 `#7ab8f5`，往下照樣漸層到白（使用者定的「白天背景白」在下半部還在；頂端的淡藍原本是我挑的）。測試守著白雲與天空頂端的亮度差（2R + 7G + B）至少 400，原本只有 257、現在 773。

92. （2026-10-08，使用者定案）pets 的天空換顏色（使用者：pets 的天空太深了，取 runner 白天天空的配色，取個中間值）：每一列取 pets 原本的天空與 runner 白天天空同一高度的顏色各一半（我的算法，使用者同意），方向反過來——runner 上深下淺，pets 上淺下深（使用者）：頂端 `#7e9bbb`，往下漸深到地平線的 `#4c709d`；原本是頂端深藍 `#1d2745` 往下到灰藍 `#3f5a7c`。綠地不動。灰藍貓（`#9fb2c8`）在淺的天空上淡一些，靠條紋分得出來；地平線附近天空較深，樹枝上的貓更清楚。87 的「顏色走暗」由這條修正。

93. （2026-10-08，使用者定案）tmux 與 screen 的設定不再整塊寫進使用者的檔案，改寫進 locku 自己的檔案，使用者的檔案只放讀它的一行（使用者提的：設定寫到 locku 的目錄、在使用者的設定檔 include 它，問會不會比受管區塊好）。檔名 `locku.tmux.conf` / `locku.screenrc`（使用者定；我提的是 `tmux.conf` / `screenrc`），放在設定目錄、跟 `config.yaml` 一起（跟著 `LOCKU__CONFIG`）。理由（我的建議，使用者同意）：使用者的檔案只在 activate 開、關時被動到，改 `lock-after-time`、`bind-key`、`lock` 不再重寫它，用 git 之類管 dotfiles 的人不會每調一次就多一筆 diff；跟機器有關的東西（locku 的絕對路徑）留在本機，同一份 tmux.conf 放到別台機器也是同一行；跑著的 server 直接 `source-file` locku 的檔案，不再寫暫存檔（決定 39 的作法換掉，原則不變：server 與檔案讀同一份）。使用者原本想到用 `if-shell` 判斷檔案在不在，沒用：它每次載入設定都要開一個 sh；tmux 自己的 `source-file -q` 就是檔案在才讀、不在不報錯。受管區塊的標記照留，裡面只剩那一行：寫入與拿掉區塊的程式不用改，舊版整塊寫進去的區塊下次寫入時直接換成一行，不用另外搬（舊區塊照樣算 activate on，它綁的鍵與 lock 照樣讀得到、拿得掉）。那一行從家目錄寫起：tmux 寫 `~/`、screen 寫 `$HOME/`——實測 2026-10-08（tmux 3.7c、screen 4.00.03）tmux 的 `~` 會展開，screen 的 `source` 不展開 `~`、只展開 `$HOME`；設定目錄不在家目錄下、或路徑有空白之類的字元時，寫單引號包住的絕對路徑（兩邊都照字面讀）。screen 沒有 `-q`、也沒有條件式：檔案不在時開頭的訊息列說一聲、照常啟動（實測）；activate 先寫 locku 的檔案再寫那一行，自己不會留下讀不到的行。locku 的檔案被刪了、那一行還在時，activate 顯示 off，再開一次就補回。shell rc 的 LOCKPRG 不動（使用者同意）：本來就只有一行，搬出去也不會變少（要寫成 `[ -r … ] && . …`，fish 還得另一種寫法）。activate off 把區塊拿掉、locku 的檔案刪掉。

94. （2026-10-08，使用者定案）PIN 最長 12 個字元，原本 64；PIN 框跟著縮成框內 29 欄、連框線 31 欄，不再跟每個 popup 一樣寬（tdp F7 的偏離，寫在 dev-remarks）。使用者提的：上限固定了，框就不用那麼寬，左右各留 2 欄。寬度我算的：點點那列一個字元是 `● `、最後再接一格游標，12 個字元是 25 欄，左右各 2 欄是 29；置中是 `(框內寬 − 2n − 1) / 2`，框內寬是奇數時點點加游標打到第幾個字都在正中間。使用者原本想的 27 或 30：27 沒算到最後的空格與游標，打滿時左右只剩 1 欄；30 是偶數，會偏一欄。終端機比 31 欄窄時照 F7 給。框本身在偶數寬的終端機上左右差一欄（80 欄：左邊 25、右邊 24），奇數寬的框放不進偶數寬的正中間；點點在框裡置中比框在畫面上置中顯眼，所以框內寬取奇數。設定畫面的三個 PIN 框與鎖定畫面同一個寬度。鎖定畫面也只收 12 個字元（使用者定案；我提的是鎖定畫面照舊收到 64，讓之前設了更長 PIN 的人還打得出來）：之前設了超過 12 個字元 PIN 的人打不完，要先改短，或從別的 shell 跑 `locku pin reset`，CHANGELOG 寫明。PIN 框裡值有換行或 Tab 時錯誤列改寫 `no line breaks or tabs`（使用者定案）：`PIN can't have line breaks or tabs` 有 34 欄，放不進 29；框的標題已經寫著 PIN。其他框照舊寫 `<邊框的型別> can't have line breaks or tabs`。順帶解掉 bcrypt 的 72 位元組上限：之前的 64 算的是字元，bcrypt 算位元組、超過 72 就拒收，中文 25 個字在 `new PIN` 過得了，到 `confirm PIN` 才失敗，錯誤列是 `bcrypt: password length exceeds 72 bytes`；12 個字元最多 48 位元組。

95. （2026-10-08，使用者定案）runner 的小貓加上眼睛（使用者：小貓沒有眼睛）：十張角色圖只有它沒標 `e`，夜晚只有外框（決定 67 那時就記下了）。眼睛不能直接放進原本的圖：頭只有耳朵下面那一列，眼睛的正上方就是兩耳之間的空隙；白天眼睛是天空色的洞，會跟那個空隙連成頭頂的一道凹口，看不出是眼睛（夜晚的金色照樣看得到）。照大貓縮小，頭加高一列、眼睛上下左右都是身體，圖從 8 × 6 變 8 × 7（我的建議，使用者選；另一個選項是維持 8 × 6、眼睛放在那一列）。仍不比小暴龍（10）高，跳躍弧與時窗不用動。`TestRunnerEyes` 原本就守「眼睛被身體圍住」，小貓的眼睛數從 0 改成 1。

## 11. 待決清單

無。2026-09-24 全部定案。

## 12. MVP 驗收

- tmux 內 `lock-session` 後：prefix d、prefix &、prefix c、prefix x 皆無反應。
- Ctrl-C、Ctrl-Z、Ctrl-\ 無反應。
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
- tmux lock-command 期間 prefix 到不了 tmux（2026-09-24 以探針實測：鎖定中送 prefix d、prefix c 都被鎖定程式吞掉，client 仍 attached；解鎖後 prefix d 才 detach）。
- shell 環境有 LOCKPRG 時 `C-a x` 進的是 locku（2026-09-24 以探針實測通過；`.screenrc` 的 `setenv` 路線實測不通，6.2 已改為 shell rc）。
