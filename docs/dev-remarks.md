# locku 開發者備忘

開發 locku 時要提醒自己、以及與 AI 協作時記下的決策。locku 遵循
[terminu design](https://github.com/vulcanshen/terminu/tree/v0.8.0)（tdp v0.8.0；2026-10-19 照重構回報的定案跟上），設定畫面建在家族的 lib
[terminu-go](https://github.com/vulcanshen/terminu-go) v0.1.13 上（2026-10-18 起，照 terminu 的交代整個重構）；
使用者要知道的在 README，這裡只放開發者需要的。

---

## 運作方式

- **鎖在 tmux / screen 之外，不在 pane 內。** pane 內的程式永遠看不到 prefix；tmux 的 lock-command 由 client 進程以 `system()` 同步執行、期間不讀任何鍵，screen 的 LOCKPRG 同理。這是唯一正確的 hook。
- **tmux：預設整台 server 一起鎖（`lock` 可改成只鎖這個 session，2026-09-25），預設不綁熱鍵，鎖著的時候誰進來都被鎖（2026-09-24）。** `prefix :` 打 `locku` 就是 `lock-server`（command alias，不跟你的 bind 撞；要熱鍵就在 `bind-key` 自己填一個，2026-09-25），所有 session 的所有 client 一起變保護程式；tmux 本身沒有「鎖著」的狀態，locku 在 `locku lock` 啟動時把全域 `@locked` 設起來，`client-attached` / `client-session-changed` hook 看到就 `lock-client -t #{hook_client}`（指名觸發的那個 client；不指名時兩個 client 同時 attach 會鎖錯人，2026-09-28 實測，理由見 function.md §6.2）——attach 哪個 session 都一樣，PIN 對了才清掉，tty 消失不清。閒置鎖是 tmux 每個 session 各自計時，哪個畫面閒置就鎖哪個畫面。lock-command 是 locku 的絕對路徑，由 tmux 展開 `#{socket_path}` 帶給 `locku lock -S`，非預設 socket 也對。`lock-session` 時旗立在 session 上、hooks 不變，別的 session 照常用；lock 程式從自己 session 的 lock-command 得知 session（`-t`），因為鎖定中用 tty 反查會拿到錯的 session（實測）。設定寫在 locku 自己的 `locku.tmux.conf`（跟 `config.yaml` 同一個目錄），你的 tmux.conf 只多一行 `source-file -q` 讀它（2026-10-08）：改一列只重寫 locku 的檔案，你的檔案只在 activate 開、關時動。跑著的 server 拿到的是同一份檔案（`source-file`，2026-09-25）：先 undo 舊設定做過、新設定不做的事，再對每個既有 session 設它自己的 lock-command。
- **screen：原理照 tmux、名字用 screen 的（2026-09-25）。** `idle N lockscreen` 與 `bind <鍵> lockscreen` 寫進 locku 自己的 `locku.screenrc`、你的 screenrc 只多一行 `source` 讀它（2026-10-08），跑著的每個 session 即時 `screen -X`；LOCKPRG 只能走 shell 環境（2026-09-24 實測 `.screenrc` 的 `setenv` 對 lock 無效：lock 是 attacher 呼叫 `getenv`），所以 activate 連 shell rc 一起寫，新開 shell 生效，已在跑的 session detach 後從新 shell `screen -r`；在那之前那些 session 鎖到的是 screen 內建的 `Key:` 鎖，toast 與 `?` 都說明。沒有 `lock`：screen 沒有 server，沒有範圍可選。
- **custom saver：你自己的程式當保護程式的動畫（2026-09-25）。** profile 填一個 `command`（`sh -c` 跑，例如 `cmatrix -b`），locku 管鎖、PIN、整合。程式跑在 locku 開的 pty 上、自己一個 process group，輸出直通終端機，按鍵永遠在 locku 手上；locku 不重啟它、不讀它的畫面，解鎖就 SIGKILL 整個 group。PIN 框疊在還在動的畫面上：每一幀後面補畫一次框、DECSC / DECRC 包住、一次 synchronised update，沒有一幀被丟；框只插在 escape sequence / UTF-8 完整的切點（`safeCut`）；收起時清掉框的位置，只對閒置半秒以上的程式送 SIGWINCH（正在畫的被要求重畫會閃一下從頭來，cmatrix 實測）。程式結束（它不該結束）鎖不退：locku 的點陣板照實寫 `EXIT <code>`（`EXIT` 金字，數字 0 綠、其他 peach；被訊號殺是 128 + 號碼，沒填指令是紅色的 `NONE`），狀態列紅字寫原因。custom 沒有 bg / fg：畫面是程式的。指令不做任何 sanitize：是使用者自己機器上自己的指令；只有換行與 Tab 不收，單行的值都不收（2026-10-06）。每個 PIN 框是自己一個 lock 程式，連錯次數與冷卻的結束時間放在 `runCustom`，開框時帶進去（`ui.Tries`）、收起時帶回來，跟內建 saver 同一個 lock 一樣（2026-10-06）。PIN 框在的時候 locku 自己送 `?2004h` 開 bracketed paste，收起與交還終端機時關：框是沒有 renderer 的 Bubble Tea 程式，開它的是 renderer，不開的話貼上的換行以 `Enter` 進來，跟按 `Enter` 分不出來（2026-10-06）。設定畫面的預覽把終端機整個交給程式，任意鍵回來。
- **忘記 PIN：`locku pin reset`（2026-09-25）。** `[y/N]` 後要登入密碼（靜態 binary 沒有 PAM，用 `su` 在 pty 上驗），過了就產生新的八位數 PIN、bcrypt 覆蓋 config 的 `pin_hash` 並顯示一次（照 elasticsearch 的 reset password，不把 `pin_hash` 清空、不留一個開著的鎖），紀錄寫 `~/.locku/data/pin-resets.log`（`$LOCKU__DATA` 可改；成功或密碼錯被拒都記，不含 PIN）；鎖定中的 locku 每一鍵重讀 `pin_hash`，下一鍵就認得新 PIN、舊的不能用；檔案壞掉沿用原本的 hash，`pin_hash` 被清空視同無 PIN。沒有終端機不跑；config 讀不到不跑。能這樣做的人本來就能殺掉鎖——locku 不是安全邊界，帳號才是。
- **進程活著 = 鎖著，結束 = 解鎖。** 任何錯誤都不得讓進程結束；只有 PIN 正確、無 PIN 模式任意鍵、tty 消失三種情況會結束。
- **畫布只有一種畫法：整面 LED 點陣板。** 每格 nf-fa-square 加空格（icon 畫成兩格的字型上只有方塊，仍是兩格，見下面 icon 的實際寬度），暗格 saver 的 bg、亮格它的 fg。字形像七段顯示器：全部直角、沒有斜線、0 沒有中間斜線，數字 3 × 7，依 size 放大；字距、行距與時間裡的空白是獨立的間隔單元（small / medium 1 格、large 2 格），不跟著像素等比放大。時間與日期是兩個獨立區塊：時間先排、日期拿剩下的空間（row 在下、column 在左），各自先降 size 再去單位（時間去秒、日期去年），日期塞不下就不畫，時間塞不下才一般文字。第一幀不動畫，之後只對有變的像素做 splash 式 shuffle。字元集 39 個。各 size 需要的終端機尺寸表在 README「你會看到什麼」。
- **狀態列**：`user@host · locked since HH:MM`（`show_status` 可關）→ config error → `no PIN · any key unlocks` → custom 的結束原因，這個順序，超過終端機寬度從尾端截掉。已知的邊角（2026-09-25）：hostname 很長加窄終端時被截的是尾端的 `no PIN` / custom note——GitHub 的 macOS runner hostname 62 字元沒有點，狀態列在 80 欄放不下，`internal/ui` 的測試在 `TestMain` 把 `whoami` 固定成 `user@host`。是設計決定，未改。

- **設定畫面建在 terminu-go 上（2026-10-18）。** `internal/settings` 只登記：一個 `Screen`、兩個 `Panel`——`[1]` 是 `List`（`Row.Group` 分成 Profiles / Savers / Integration / Settings，啟用中的 profile 在格子裡畫 Green `●`），`[2]` 是 `Values`（一列一個 `Field`）——`[1]` 的 `OnMove` 換 `[2]` 的內容、label、動作與 `?` 最後一節，`Values.OnMove` 換 `[2]` 的 `Enter` 那一列。Space menu、`?`、hint、footer、confirm、toast、splash、`Esc` 一次一層、淡化與開關動畫都是 lib 從這份登記畫的，locku 不再有自己的 popup 疊（之前 `internal/ui` 的 app.go、actions.go、spacemenu.go、helppopup.go、popup.go……全刪）。欄位：五個以內的選項畫成原地的 radio（`Field.Options`），超過五個（`participants`、profile 多於五個時的 `profile`）開 `Select`；開關（`show_status`、`activate`）是單一選項的 checkbox；顏色是 `PickColor`；PIN 是 `PasswordInput` 的解鎖樣式；只顯示的列（saver 的說明、profile 的 saver）是停用的 `Field`，cursor 跳過。每個值在 popup 的 `Done` 裡改 config、`config.Save`、`refresh()` 重建兩邊的列（失敗退回並 toast）。預覽用 `tea.Exec` 把終端機交出去：`previewRun.Run()` 在同一個 tty 上跑 `ui.RunPreview`（一個自己的 Bubble Tea 程式，任意鍵 `tea.Quit`），custom 跑 `custom.Preview`，結束了再跑字的板子。
- **icon 的實際寬度（tdp E5；2026-10-18 起是 lib 的）。** 設定畫面的 `App.Run` 與鎖定畫面的 `ui.DetectIconWidth()` 都是 lib 的 `terminu.Probe("locku")`：icon 與 powerline 符號各量各的，`LOCKU__ICON_WIDTH` / `LOCKU__POWERLINE_WIDTH` → `TERMINU__ICON_WIDTH` / `TERMINU__POWERLINE_WIDTH` → CPR 探測；同時用 OSC 4 問 16 色的調色盤給淡化用。鎖上照舊放在 `tmux.SetLocked` 之後、`termreply.DropPending` **之前**——逾時才到的回答被清掉，就算漏進鎖裡，`termreply` 也把終端機的回報丟掉，不會叫出 PIN 框或解鎖。`internal/ui` 的寬度只剩幾個轉給 lib 的函式（`width.go`：`dispW` = `terminu.Width`、`truncate`、`clipANSI` = `terminu.Cut`），框疊到板子上用 `terminu.Composite`，淡化用 `terminu.DimLines`（`dim.go`）。custom saver 的程式在 `custom.Start()` 拿到兩個 `TERMINU__` 變數（外面帶進來的同名變數被蓋掉，exec 同名取最後一個；powerline 的從 2026-10-18）。bubbletea v1 的 renderer 會在它量起來比終端機窄的列後面補 `ESC [ K`（icon 算一格），icon 佔兩格時會擦掉最後一格：lib 的 `App` 在輸出那層拿掉它，鎖定畫面自己開 program，所以 `ui.ScreenOutput` 照抄一份（`output.go`，TODO(lib LK1)）。2026-09-29 到 2026-10-17 是 locku 自己照 filu 搬的 `width.go`、`dim.go`、`iconwidth_unix.go`。

### 程式碼目錄

```
locku/
├── cmd/locku/          進入點：lock / pin reset / version / 設定 TUI；argv[0] SCREEN-LOCK
├── internal/
│   ├── config/         config.yaml 的讀寫：fail open、原子寫、0600、bcrypt PIN、pin_hash 重讀、NewPIN、data 目錄
│   ├── custom/         custom saver：程式在 pty 上、輸出經 screen writer 直通、PIN 框疊在畫面上、按鍵留給 locku、結束的字（2026-09-25）
│   ├── login/          `locku pin reset` 的登入密碼驗證：su 在 pty 上（2026-09-25）
│   ├── saver/          內容：clock 的兩種 time × 五種 date × row / column、tick；會動的 saver 共用的 Game 介面與場景（game.go）；runner（原 dino）的跑者、場景、障礙與自動跳躍、天空（`sky.go`）；bounce 的框；snake 的環與近路；tetromino 的方塊、AI 與消行；pets 的房間、找路與貓；custom 結束的 Word
│   ├── setup/          整合寫檔：locku 自己的 locku.tmux.conf / locku.screenrc，使用者檔案裡讀它的一行受管區塊（跑著的 tmux server 整份 source-file），shell rc 的 LOCKPRG
│   ├── termreply/      鎖定時從讀到的輸入拿掉終端機的回答（OSC、DCS、CSI 回報），只留按鍵；鎖開始時清空已在等的輸入（2026-09-28）
│   ├── tmux/           鎖定中對 tmux 立 / 清 @locked 旗
│   ├── settings/       設定畫面：在 terminu-go 上登記 panel、欄位、動作、popup；預覽（2026-10-18）
│   ├── ui/             鎖定畫面：渲染器（font / canvas / reveal）、PIN 框（照 lib 的樣子自己畫，LK1）、預覽程式
│   └── version/        版本字串（goreleaser 以 ldflags 注入，本機 build 是 dev）
├── e2e/                pty 端到端：tmux_attach.py、custom_lock.py、screen_lock.py
└── docs/               function.md、ui.md、ux.md、dev-remarks.md、icon.svg、social-preview.png、demo.gif、savers/（README 每種 saver 的靜態圖）
```

## 設計決定

- **非安全邊界。** 另開一條 SSH 就能 kill。定位是螢幕保護與防誤觸，config 缺失或損毀一律 fail open。攔不到的東西（function.md §0.1、§2.3）：ssh 的 `~.`、Linux VT 切換、另一條 SSH 的 `tmux attach -d` / `kill -9`、終端機模擬器自身的快捷鍵。
- **驗證只有自家 PIN**，bcrypt 存 config；PAM 留 `auth: pam` 擴充位，shadow 不做。理由：locku 是單一靜態 binary（`CGO_ENABLED=0`），連 PAM 就要 cgo 與各平台的 PAM 設定；讀 shadow 要 root。PIN 本來就不是要擋住帳號本身（非安全邊界，見上一條），bcrypt 雜湊放在 0600 的 config 就夠。錯誤 PIN 固定 1 秒 debounce：一次錯了要看得到「錯了」，也不能用連按 `Enter` 閃過訊息；連續錯誤鎖定可設定、預設關：螢幕保護多半是自己在用，預設就鎖人只會鎖到自己（function.md §4.4）。
- **saver 是 class、profile 是 object（2026-09-24 定案）。** 七種 saver：clock、runner（2026-10-07 前叫 dino）、bounce、snake（2026-10-06）、tetromino（2026-10-07）、pets（2026-10-07），與 custom——你自己的程式（2026-09-25）；profile 是設定好、有名字的一份，config 裡 `profile` 指向的就是它，鎖定畫面顯示的也是它。新增 profile 從一種 saver 按 `n`，profile 的 saver 建立後不改。dino 是 Chrome 離線小恐龍遊戲當螢幕保護：地面與仙人掌向左捲、暴龍自己跳過去，無限循環、隨機障礙、隨機跳躍、不會死，不記分也不畫時間；參數只有 runner（big / small 一隻大或小暴龍，big-big / small-small / small-big / big-small 兩隻一前一後、名字就是畫面由左到右的順序、各自跳各自的；2026-09-25 定案，舊的 trex / two-trex 自動轉）、character（t-rex、cat、rabbit、horse、ghost，2026-10-06）、scene（grassland 仙人掌，或 desert 金字塔）、bg / fg，沒有 size，畫布自己取塞得下的最大倍率；每 70 ms 一幀。clock 的 layout row / column（直排把 `HH` / `MM` / `SS` 拆行，字大好幾倍）、size small / medium / large（一個字型像素 1 / 2 / 3 格見方）、font 3x7 / 3x5、time `HH MM` / `HH MM SS`（24 時制，不畫冒號、以空白分組）、date off 或四選一、bg / fg 兩色；沒有任何自由輸入；可 duplicate / rename / delete。
- **顏色是每個 saver 自己的**，bg / fg 各一個 color picker（lib 的 `PickColor`：old 與 new 並排、R G B 用 hex 從 select 選），選好就寫檔，跟其他每一個值一樣（2026-10-18；之前是三條 RGB slider 改草稿、`S` 寫檔、`R` 丟掉、`[2]` 標題接 `unsaved`，`q` 遇到未存草稿先問——tdp 的一列一個值沒有送出，草稿就是送出，見 LK5）。custom 沒有顏色；bounce、snake、tetromino 與 pets 的顏色是它們自己的，也沒有（2026-10-06，`saver.Coloured`）。
- **bounce：反彈框（2026-10-06）。** `internal/saver/bounce.go`。框是時間的 3x5 字加 2 px 留白加 2 px 框線（`bounceBorder`，2026-10-07 前 1 px）：`HH:MM` 25 × 13 px、`HH:MM:SS` 35 × 13 px，用 `time.Time{}.Format(layout)` 的寬度算；速度是 `bounceSpeeds`（名稱跟 snake 共用 `saver.Speeds`，normal 每秒 10 px），`Next` 照它；點陣字留在 ui，`NewBounce` 收一個 `spell`（ui 的 `spell(faceShort, …)`，跟 clock 畫在板子上的逐像素相同，有測試）與取時間的函式，所以 `Game` 介面不用帶時間。每軸 `bounceAxis`：走 1 px，到 0 或 span 就反向並回報碰到；兩軸同一幀都碰到是角落，開始閃（`flash` 幀數 = 顏色數 × 2，每幀下一色），只有一軸是換一個不同的隨機色。場景比框小的那一軸 span ≤ 0 就不動、不算碰到，否則窄終端會每幀換色。resize 時框留在原處、塞不下就靠邊。
- **tetromino：自己玩的俄羅斯方塊（2026-10-07）。** `internal/saver/tetromino.go`。場地以方塊為單位（2 × 2 px），是完整的長方形；`reset` 用場景扣掉外框（四邊 1 px，`tetroEdge`）與右邊小框的 7 px（`tetroBoxW` 加 1 px 框）算出 `cols × rows`，外框在場景左上角。接下來的方塊是 `queue`，個數是外框高度放得下的整格數（`reset` 算，`come` 從前面拿、從袋子補到後面）；欄貼在場地外框的右邊、共用那條線：`Draw` 畫每格上面一條橫線、最後一格下面一條、最右邊一條直線到最後一格為止，最後一格下面到外框底部整塊塗外框灰（2026-10-07），預覽一格 1 px（`preview`）。`open`：場地外與堆疊擋住，場地上面的隱藏列整排開著（方塊進場用）。方塊的轉向在各自的框裡順時針轉（`tetroTurns`，O 只有一個）。一步 = 轉一次或橫移一格（或都不做）、再往下一格或不往下（`step`）。`choose` 從進場位置 BFS 走遍所有走得到的位置，停得住的（再往下不行）用 `worth` 評分取最高；`route` 從目標倒著 BFS，記每個位置還差幾步，`toward` 每一步挑剛好少一步的那個，偏好「轉向或往目標橫移、同時往下」，所以會斜著走、也一定走得到（鑽到懸空的下面也是）。`worth` 只有兩項：落點高度（兩倍的高度、整數）與洞（一個 4），滿列先拿掉再算；試過常見的四項（總高、洞、凹凸、消行）會堆在角落，Dellacherie 的六項正常，但一項一項拿掉量下來只有這兩項有作用。消行與結束不照速度：`Next` 在 `going` 或 `over` 時回 40 ms，長度用 `frames(d)` 換算；消行先 0.28 秒全白，`wiped` 用「離較近那端的距離」分圈、從中間往外；結束時 `drawEnd` 依幀數把外框高度裡從上面算起的幾列、所有亮著的格子換成外框的灰（那時亮著的只有方塊與本來就灰的外框），5 秒掃到底，之後才在整個場景的正中間畫 END（UI 傳進來的 `spell`，跟 bounce 一樣用 3x5，直接畫在灰色上），再 2 秒 `reset`。測試守著：轉向、7 個一袋、外框與那一欄的位置（逐像素比）、欄裡依序是接下來的方塊且置中、格數跟著高度、拿走一個其他往上挪、3000 步裡每一步最多動一格不往上、不重疊、堆疊最高 7 列洞最多 5 個（實測 5 列、0–1 個）、消行逐幀、鑽到懸空的下面、評分滿列先拿掉、有選擇時不選結局、結束時逐幀變灰到第幾列、顏色不變、END 的像素、速度、同 seed 同局、太小不畫。2026-10-07 前只顯示 next 與 next next，起先在場地左上、右上角的盒子裡，場地是盒子剩下的倒 T 形，小場地常堆到頂（function.md 決定 73、76）。
- **pets：室外的貓（2026-10-07）。** `internal/saver/pets.go`。室外在 `reset` 時擺一次（`furnish`）：地面一條草線與零星草叢；地上的東西先算寬度（`tree`：樹冠的高、寬、頂，樹枝從最低一根往上、兩側輪流，各自的長度，樹兩邊各伸多遠；`stump`：寬與高），至少一棵樹，`spread` 把剩下的空隙隨機分，再 `putTree`（樹幹、樹冠的橢圓與葉影、樹枝各一個 ledge、樹幹兩側各一個 climb，`trunk` 標記，爬到那側最高的樹枝或樹冠下）與 `putStump`。背景不在 `room` 裡：`Shade` 回一個 `Look`，`Ground` 是每一列的顏色（`petBackdrop` 的色標之間用 `mix` 內插），畫布照 runner 的天空那樣逐列上色。貓所在的地方是 `petPlace`：ledge 是一條線，貓的中點範圍 `lo..hi`，可以懸出端點 2 px；climb 是樹幹的一側，貓的腳的列範圍。`join` 把跳得到的連起來：往上最高 `petJump`，從 B 的旁邊起跳（貓身不在 B 底下）、落在近的那端；往下或平著，從 A 的端點跳、落在 A 外面；兩者橫向不超過 `petReach`；樹幹跟每個橫向夠近的 ledge 也連（`feet = max(L.at, C.lo)`），樹幹的腳由 `mount` 連到地面。`way` 是 Dijkstra：節點是起點、終點與每條連結的兩端，同一個地方上的移動每 px 2 幀、跳是它自己的幀數（`leap`：拋物線，頂點比較高的那端再高 2 px，`k = |dy| + 4 + √(8|dy| + 16)`）。`reach` 是從地面到得了的地方，`pick` 從裡面挑、避開別隻貓一隻貓寬。貓的圖朝右，`petBlit` 翻面；圖上的記號 `#` 毛、`s` 條紋、`w` 胸口與腳掌、`x` 黑白貓多出來的白、`e` 眼睛，各是 `petCoats` 那隻的一個墨（`petCoat(c)`，每隻 5 個）。測試守著：樹與樹樁的像素（golden）、每個室外的完整性（草線、至少一棵樹、上下兩個 ledge 之間留得下坐著的貓、樹幹的範圍、每個地方都到得了）、背景的漸層、跳的高度與距離與「不從底下穿過」、走最快的路、步調、坐與睡的圖與 z、樹的數量、高度與樹枝長短有變化；mutation 逐處改壞全部變紅。之後要加別的動物：圖、大小、能力（會不會爬、跳多高多遠、步調）與花色收成一份「種類」，連結記下是爬還是跳、多高多遠，`way` 與 `reach` 依種類跳過做不到的；設定多一個選動物的屬性，沒寫就是貓，不用升 config 版本。
- **snake：貪食蛇（2026-10-06）。** `internal/saver/snake.go`。`hamilton(cw, ch)` 造環：列數偶數就從第二欄起來回走、最後沿第一欄回去；列數奇數就轉 90 度；兩邊都奇數就丟最後一欄（測試掃過 1–12 × 1–12）。不撞自己只靠一個不變量：沿著環，身體依序在頭後面，頭前面到尾巴之間全空；抄近路距離 < 到尾巴的距離就維持得住（測試每一步驗：尾巴往前經過每一節到頭、再到尾巴，剛好一圈）。近路的上限 `min(到尾巴 − 蛇長 − 3, 到果子)`，蛇過半盤時這個值自然小於 0，所以參考作法（John Tapsell 的 Nokia snake AI）裡「過半不抄」與「果子後空間大時少抄」兩條拿掉。退路是環上的下一格：空的，或尾巴（同一步會讓開）。格距 3 px（`snakePitch`）：節點之間兩格連線，並排兩段之間兩列暗格；頭（`head()` 挑形狀：吃到的那一步是張嘴 `snakeOpen`——頭在果子那格，上顎三格、下顎兩格夾住它，節點畫成果子的顏色（包的位置是 `snakeBitten` = −2，下一步 +2 成 0 就是吞嚥，吞嚥時果子畫在頭的節點，頭的四周沒有別的（2026-10-07 前頭頂後面還有 `snakeGulp` 鼓起的一格，`Draw` 把 −2 夾成 0，張嘴那一步也畫了它，使用者回報後整個拿掉；測試把使用者畫的閉嘴、張嘴、吞進去寫成字串，連外圍一圈暗格逐格比對）；在身體裡時果子就是那一節的節點，旁邊什麼都不畫——2026-10-07 前身體繞著它鼓起：直線 `snakeLump` 3 格、轉角 `round(i)` 從外側繞 5 格、尾巴 `snakeEnd` 2 格與 `snakeFade` 1 格，使用者看過實機後拿掉；測試在 20 個時間點把果子輪流放進頭以外的每一節，確認它四周 8 格只有兩段連線是亮的）；下一步的 `move()` 就是果子是碰到 `snakeTouch`——朝果子（`way(頭, 果子)`，可能轉彎）推進一格、鼻尖貼著果子，身後連線交給身體畫；其他是閉嘴 `snakeHead`——鼻尖、節點、連線、頭頂；頭那一節的節點由形狀畫，所以 `Draw` 不點第 0 節的節點（上一版張嘴時它是嘴）；下唇在線的另一側，四周留的 1 px 也是給它的）畫在節點旁的空處、只凸出 1 px，跟並排的另一段之間還有一列暗，`turn` 依方向翻轉或轉 90 度——頭頂只用到頭自己往那一側的連線位置（頭的另一節在背後，那條連線不會亮）與節點之間永遠暗的內格；下巴是往前那條連線的第一格，也一定是暗的，它上面那格（嘴）是內格。格子四周各留 1 px（`reset` 的 `ox` / `oy`）。包是身體上的位置（`lumps`，0 是頭；蛇走的時候位置不變，所以跟著身體走），各自計時（`wait`），每 `snakeDigest`（一秒）往後一節；在尾巴再一秒就丟掉、蛇換成它的顏色、`grow++`，同一個 `Step` 裡尾巴就不動（2026-10-07 前中間還有一拍 `gone`）；`Step` 有 `grow` 時尾巴不動，但下一步是尾巴那一格時照走（頭不會撞上），延後再長。一節一顆：咬到的那一步 `Step` 先把嘴裡那顆加進 `lumps` 再呼叫 `queue()`——嘴裡那顆算在第 0 節，由後往前把擋路的往後推一節；推出尾巴的那顆當場是身體（蛇換色、`grow++`），而且這發生在尾巴的取捨之前，所以同一步就長；`digest` 裡一秒到了、下一節還被前一顆佔著的原地等（`wait` 留在 1，下一步再試）——被推過的豆子走在自己的節拍前面，會在尾巴被後面那顆追上。2026-10-07 前推擠在下一步的 `digest`、推到尾巴就停，豆子會疊在尾巴（使用者回報）；測試每一步檢查豆子的位置由舊到新嚴格遞減。測試有一條守恆：身長 + `grow` + 肚子裡的豆子數 = 3 + 吃到的數。顏色跟 bounce 共用 `ownColours`（`game.go`），另加開局的白色 `snakeWhite`：顏色編號 `len(ownColours)`，`Inks()` 接在十色後面，`fresh()` 的 `used` 多一格把它佔住，所以果子不會是白的；每個包帶著自己的顏色（`lump{at, colour}`），`digest` 讓包跑過尾巴時把蛇換成它的顏色；新果子的顏色由 `fresh()` 抽，避開蛇與所有包的顏色，十色用完才退回 `otherColour`；速度（`speed`，五段名稱，`snakeSpeeds` 換成每秒 5 / 7 / 10 / 14 / 20 格，不認得的當 `normal`）決定 `frame`；填滿停的 3 秒與果子的閃爍是時間，`frames()` 換成步數，最少 1。第一版（同日）是 2 px 格距、bg / fg 兩色、沒有頭；第二版頭是 3 × 3 帶眼睛（使用者：太大）、包留在吞下的那一格；第三版平常收下巴、吃時下巴往前伸（使用者：怪怪的）；預設色先是 Nokia 淡綠當底（vhs 實拍整面亮牆）再改深綠當底，加換色後整個拿掉。
- **設定畫面的細節**：`[1]` 的 `P` 無效（`p` 才是游標那列），profile 列 `a` 直接設為啟用（決定 41，2026-09-25）；每個 `[2]` 第一列是 `Property` / `Value` 表頭（lib 畫成 Overlay0）；panel 的膠囊是 lib 的 `[1] LOCKU`、`[2] <名字>`；`config file path` 的框是 lib 的 text：有值時是灰字提議（`Tab` 接手、`Backspace` 拒絕、沒碰就 `Enter` 不改），沒有值時沒有提議、說明列寫通常是哪個檔（2026-10-18；之前沒有值時提議 `~/.tmux.conf` / `~/.screenrc`，tdp 的灰字只能是舊值）；`activate` 是「區塊在不在檔案裡」，每次重畫都重讀，勾它先 confirm 才寫，路徑沒填就停用；區塊只碰 `# >>> locku >>>` … `# <<< locku <<<`，每行尾巴 `# locku`，冪等；preview 不驗 PIN。PIN：沒有時 `Set PIN` 打兩次；有時 `Change PIN` 先驗目前的，再開「新的 PIN」框，框裡是 8 顆灰點代表現有的——直接 `Enter` 保留，`Backspace` 再 `Enter` 拿掉（2026-10-18；之前驗完選 `New PIN` / `Remove PIN`）；三步是一步關了才開下一步（LK7）。
- **Enter = 進 `[2]` / 編輯 / 送出，Esc 只做取消，`X` 刪除，`d` 是半頁。** 理由：`Enter` 在 `[1]` 上一律進 `[2]`，在 `[2]` 上是那一列最直觀的動作（tdp K3），同一種東西永遠同一個動作，所以設為啟用不用 `Enter`、改成 `a` 與 preference › profile；`Esc` 只當取消（tdp K4）；`d` 是移動的半頁（tdp K12 保留給移動），刪除就給 `X`，跟 sshu 的紀錄類項目一樣用大寫，免得一按就刪（2026-09-24）。畫布上任何鍵只開 prompt，第一個鍵不算輸入：叫醒畫面的那一鍵通常是隨手按的，算進 PIN 只會多一次錯。
- **config 有版本（2026-10-07）。** `config.ConfigVersion`（1）。`LoadFile` 先把檔解析成樹，`versionOf` 讀頂層的 `version`（沒有就是 0），小於 1 才跑 `carryOver`（0 → 1：2026-09-24、09-25 的改名與 dino → runner），decode、`sanitized` 之後，版本小於現行的就用 `SaveFile` 整份寫回；`SaveFile` 一律寫成現行版本，所以 `version` 永遠是檔案的第一個 key。下一次格式要變：加一個 1 → 2 的轉換接在 `carryOver` 後面、只對小於 2 的檔做，`ConfigVersion` 加一。測試守著：0 讀完寫回、再讀不動；1 的舊名不轉、不寫回；2 照讀不寫回、存檔寫成 1；讀不了的檔不寫回。家族裡 locku 是第一個做 config 版本的。
- **環境變數照家族的 `LOCKU__NAME`（2026-09-29，tdp E2）。** app 名後兩個底線：`LOCKU__CONFIG`（設定目錄）、`LOCKU__DATA`（資料目錄）、`LOCKU__ICON_WIDTH`、`LOCKU__POWERLINE_WIDTH`（2026-10-18，lib 讀）。舊名 `LOCKU_CONFIG`、`LOCKU_DATA`（v0.1.3 已發佈）不再讀、不留相容（user 裁定），CHANGELOG 寫了新舊對照；`uninstall.sh` 與三個 e2e 腳本一起改。
- **暫定（2026-10-18，重構時自己定的，等 user 確認）。** tdp 交給 app 的地方：
  - 離開不問：沒有沒存的東西（顏色草稿拿掉之後），`App.Quit` 是 nil，`q` 直接離開（tdp K9：離開流程由 app 決定；之前有未存草稿才問）。
  - popup 的標題與 glyph（components/layout/popup：input 的 glyph 由 app 選）：New profile / Duplicate / Rename 用 `nf-md-pencil`、數字用 `nf-md-timer_outline`、顏色 `nf-md-palette`、select `nf-md-tag`、路徑 `nf-md-file`、command `nf-md-console`、按鍵 `nf-md-keyboard`、PIN `nf-md-lock`；設定畫面的 PIN 框標題 `SET PIN` / `CHANGE PIN`，鎖定畫面的是 `UNLOCK`（glyph 沿用 `nf-fa-lock`）、說明列 `The PIN`。
  - 只顯示的列（saver 的 `saver`、`what`、`profiles`、`defaults`，profile 的 `saver`）畫成停用的欄位：label Surface2、cursor 跳過。tdp 的一列一個值沒有「唯讀列」，停用是最接近的。
  - 兩個 panel 的寬：`[1]` 24 欄（`min(24, w/2)`），59 欄以下一次只畫有 focus 的那一個（`NarrowBelow: 60`，舊的 `w < 60`）。

## 已否決，不要重提

pane 內攔截 prefix、attach 使用者現有 session、config 缺失時鎖死、PAM / shadow 進 v1、自由文字 saver、strftime 自由格式、
跑馬燈、拿掉像素間空格、`[2]` 內的 preview 框、底板 sheet、Integration popup、`--saver` 命令列覆蓋、
`locku init`、只印不寫的 setup、`locku setup` 指令、後來的 `S` / `X` 熱鍵（畫面上看不到）與底部的 Install / Uninstall 按鈕（風格不對；改成 `[2]` 第一列 `activate` on / off）、preference 每列下面的說明列（搬進 `?` help）、`.screenrc setenv LOCKPRG`（實測不通）、全域的 style 設定（顏色改為每個 saver 自己的）、
側欄 Enter 設為啟用（改在 preference › profile 選，後來加 `a` 熱鍵）、12 時制 AM/PM、時間的冒號、有斜線的字形、
tmux 預設的 `bind L`（跟使用者既有熱鍵撞，改 command alias；要綁的自己填 `bind-key`）、session 等級的 tmux 鎖（換個 session 就繞過，改整台）、
閒置鎖升級成整台（雙螢幕會被另一邊鎖到）、一個 PIN 解全部 client（要輪詢，維持各自輸）、
純隨設即得的整合寫入（conf 打錯就生檔）、純按鈕不同步（改了值檔案 stale）、`[2]` 裡的 status 列（狀態就是 `activate` 列）、` · ` 分隔的純文字標題（改膠囊鏈）、preview 也驗 PIN、標題膠囊裡的 installed / uninstalled 狀態與串在標題後面的種類（狀態是 `activate` 列；種類搬到右上角一顆獨立膠囊後也拿掉，分類資訊多餘）、`[2]` 下框右側的 config 路徑（第一版就有，不是家族慣例）、lock 程式用 tty 反查 session（鎖定中 list-clients 是空的，display-message -c 回錯的 session）、
custom 的 VT 終端機模擬器路線（多一個依賴、忠實度與效能都要驗；改直通）、custom 程式的自動重啟（不干涉生命週期）、黑畫面加紅字（用板子）、結束的字叫 COMPLETED / ERROR 再叫 DONE / ERROR（改直接寫退出碼）、PIN 框下面凍住畫面加 locku 的底色（框下面沒有動畫）、框收起時真的縮一欄再放回去逼程式重畫（改成不凍畫面就不用補畫）、對正在畫的程式送 SIGWINCH（cmatrix 會閃一下從頭來）、tmux 即時套用維護一份跟區塊平行的指令清單（改整塊 `source-file`）、整合設定整塊寫進使用者的檔案（改寫進 locku 自己的檔案、使用者的檔案只放讀它的一行，2026-10-08）、用 `if-shell` 判斷 locku 的檔案在不在（`source-file -q` 就是，不必每次開 sh）、shell rc 的 LOCKPRG 也搬進 locku 的檔案（一行搬出去不會變少）、鎖定畫面上的忘記密碼入口、恢復碼、清空 `pin_hash` 當 reset、`[1]` 上的全域 `P`（`p` 才是游標那列）、原生 Windows。

## 已知的牆與未做

- **等 lib 的地方（2026-10-18 回報 terminu，2026-10-19 跟 user 定案；程式裡標 `TODO(lib LK<n>)`，等 terminu-go 第三批）**：LK1 鎖定畫面借 lib 的 PasswordInput；LK2 新建後把 focus 帶到新的 profile（tdp S19）；LK3 lib 的「每一列」一節（tdp S34）；LK4 灰字的建議名（tdp S6）；LK6 設定畫面 PIN 框的說明列拿掉（tdp S36，標題已改成 `CURRENT PIN`、`NEW PIN`、`CONFIRM PIN`）；LK7 PIN 三步疊起來；LK8 數字框可以空（tdp S7）；LK10 寫檔失敗疊在框上；LK11 捲動位置照欄位（tdp S11）；S47 輸入框的單位列（panel 上已經寫 `seconds`）。照現在、不用改的：LK5 顏色 Choose 就存（tdp S35）、LK15 hex 一律大寫（tdp S38）。2026-10-18 時的寫法：LK1 鎖定畫面借不到 lib 的 input（PIN 框照 lib 的樣子自己畫，`ScreenOutput` 照抄）；LK2 lib 沒辦法從程式換 focus（New / Duplicate 之後 focus 留在 `[1]`）；LK3 設定 panel 的每列說明放在哪（暫放 key reference 最後一節）；LK4 新東西的名字提議（寫在說明列）；LK5 顏色先預覽再存（草稿拿掉了）；LK6 鎖定畫面的 PIN 框從 7 列變 9 列；LK7 多步驟的輸入不能保留上一步；LK8 number 框清空不再是預設值。還沒符合 tdp 的逐條在 [`locku-terminu-fix.md`](locku-terminu-fix.md)。
- **偶發的測試**：`TestBounceLockWearsItsOwnColours`（`internal/ui`）在 `make check` 整包 `-race` 跑時偶爾失敗（`the box did not move`），單獨跑十次都過；2026-10-18 重構時遇到，跟重構無關，未查。

- **平台**：macOS 與 Linux（WSL 可）。不支援 Windows：鎖站在 tty、pty、`su` 與 tmux / screen 上，原生移植是另一個產品（2026-09-25，使用者定案）。
- **狀態列在窄終端被截**：hostname 很長加窄終端時，被截的是尾端的 `no PIN` / custom note（見「運作方式」的狀態列），是設計決定，未改。
- **未做**：8 小時 CPU / 記憶體觀察（`function.md` §12 最後一項，尚未做）。
- **未做：更多 saver 的點子**（2026-09-27 與使用者發想，沒有要動手）。前提是畫面全部自己跑（任何鍵就離開）；貴的不是像素多，而是每幀變動的像素多——只重畫有變的，所以稀疏的便宜、整面變的貴，後者要壓更新頻率。
  - **runner 的時段配色**：做了簡化版（2026-10-07，`background`）——`day` / `night` / `time-shifting`，每種一道從上到下的漸層（每一組 `Look` 的地面一列一個顏色，鋪滿整個板子），前景一色；time-shifting 三分鐘一天，牆上時鐘的分鐘除以 3，每段前 10 秒從上一段一格一格換過來（`skies()` 回傳從哪一段、到哪一段、走了多少；`saver.Graded` 的 `Shade(cols, rows)` 給畫布兩組顏色 `Look`——每列的地面與每個墨——和每一格用哪一組；`turnsAt` 決定一格在過場的哪個時間點換：離右邊越近越早，再加上固定在格子上的隨機（hash，同一格每次一樣，不會閃）；`boardRows` 依墨與組別分段上色）；夜晚的跑者是 `inkRunner`（白天、黃昏 `#313244`，夜晚 base `#1e1e2e`，比它跑過的夜空深、比 crust 亮一階——crust 是終端機常見的背景，方塊會跟縫隙糊在一起）加 `inkOutline`：`rim()` 算出精靈圖四周八個方向、不屬於它的格子（腳底下那一列也算：站在地上時那裡是地面線，外框只畫在天空上，所以只有跳起來時看得到）當外框（`inkOutline`，偏冷的白 `#cdd6f4`，跟偏暖的月亮分得開），角色圖裡標成 `e` 的格子是眼睛（`inkEye`，金色）——不用「被身體包住的洞」來猜：T-Rex 跑步時兩腿之間也有被包住的一格，鬼的眼睛斜角又是開的；外框與眼睛先畫、只畫在空的格子上，再畫身體；太陽、月亮、夕陽是場景的第 2、3、4 號墨，先畫、雲與障礙物蓋在上面——太陽月亮固定在右上方，夕陽（太陽的上半）在太陽正下方的地面上；某一組天空裡沒有的日月，那一組的墨是空的、畫成地面，所以過場時一格一格出現、消失；`bg` / `fg` 拿掉了（使用者），`saver.Coloured` 只剩 clock。原本的想法沒做的部分：跟本機時間走五個時段、恐龍地板障礙物雲日月各自的顏色、時段之間的過場做成 10 秒、一格一格換（中間做過整面在 sRGB 裡漸變，很順，但使用者要的是直接一點）、日月走弧線與星星。256 色終端機上漸層會跳階。
  - **換角色**：貓、兔子、長頸鹿（原本叫馬，2026-10-07 改名）、小精靈的鬼做了（2026-10-06，`character`）：`participants`（原 `runner`）只剩隊形（`formations`：大小與前後），角色的大小兩張圖在 `cast`；寬度跟同尺寸的暴龍一樣、不比它高，跳躍弧與「跳得過」的時窗不用動（測試每個角色 × 隊形 × 場景跑 6000 幀不撞）。太空船躲隕石沒有地面、不用跳，是另一種 saver，沒做。
  - **Nokia 風格**：貪食蛇做了（2026-10-06）；還有 Conway 生命遊戲、Langton 螞蟻、自己打的 Pong / 打磚塊 / 俄羅斯方塊。
  - **其他**：便宜的有星空飛行、雨 / 雪、煙火、迷宮生成與走迷宮、2D 水管、排序視覺化；貴的有 Doom 火焰、plasma、熔岩燈。
  - **貴的效果照做，告訴使用者**（2026-09-27，使用者）：不因耗電而不做，在該 saver 的說明寫「比較耗電」，讓使用者自己選。但幀率仍要有上限：輸出塞滿時 PIN 框的按鍵回應會變慢，經 SSH 更明顯，那是功能問題，不只是耗電。
  - **底層**：這些共用同一套——像素各有顏色、定時更新、只重畫有變的。2026-10-06 動手前定案：板子的每格是一種**墨**（`board.ink`，0 是暗、其他是亮，各一個顏色），場景也是（`saver.Scene.Pix`）；顏色由畫板子的人給（`boardRows(b, inks, cols)`）。會動的 saver 共用 `saver.Game`（`Room` / `Step` / `Draw` / `Next`），場景要多大、最多放大幾倍是各自的 `Room`，dino 是第一個搬上去的。顏色（使用者定案）：**多色的 saver 自帶配色，沒有 bg / fg**，跟 custom 一樣不給設定（實作 `saver.Inked`）；只用兩色的照舊是 profile 的 bg / fg。搬的時候拿改之前後各種 clock、EXIT、dino 的鎖定畫面逐位元組比過，一樣。
- **未做：使用者自己描述的 saver**（2026-09-27 與使用者發想，沒有要動手）。custom 已經讓人跑自己的程式，但用不到 locku 的畫布；中間可以有一種資料檔，像 `.obj` 描述形狀那樣，描述 2D 的圖與運動。能描述的分三級：
  1. 逐格動畫：調色盤加幾張圖，照固定速度輪播。
  2. 物件加運動規則：每個物件是一組圖，加位置、速度、碰邊界時反彈 / 繞回 / 消失，可在範圍內隨機。反彈框、捲動背景、雲、星空、煙火都寫得出來。
  3. 邏輯：貪食蛇轉彎、生命遊戲、恐龍何時跳。這已經是程式，要嵌腳本語言（如 Lua），不做，留給 custom。

  格式用 YAML，跟 config 一致；圖用多行字串畫，一個字元對應調色盤一色，打開檔案就看得到圖。尺寸只寫邏輯像素，縮放交給畫布（跟 clock、dino 取最大倍率一樣）。它是資料，不執行任何東西。編輯器不在 locku 裡做：用 Aseprite、Piskel 或文字編輯器畫，locku 提供存檔就能看的 preview。

  ```yaml
  palette: { "#": "#ffffff", "o": "#ff8800", ".": none }
  sprites:
    ghost:
      frames:
        - |
          .###.
          #o#o#
          #####
  objects:
    - sprite: ghost
      at: [random, random]
      velocity: [1, 1]
      edge: bounce
      on_bounce: next_color
  ```

## 偏離 tdp

（2026-10-19 起不再是偏離的：鎖定畫面與預覽不套 core key、鎖定中 `Ctrl-C` 不離開——tdp K1 補定：不在 TUI 裡的畫面不是 surface，
不照 core key 表，鎖定畫面與預覽都是；鎖定畫面 PIN 框的連錯冷卻倒數寫在錯誤列——tdp S37：錯誤列可以放「值錯太多次而暫停收值」
的倒數。）

- **鎖定畫面的 PIN 框是 locku 自己畫的（tdp E8；2026-10-18）。** tdp E8 說畫面的規定 lib 給的，用 lib 的 app 自己不另外做。鎖定畫面是全螢幕的板子、沒有 panel 也沒有 footer，lib 的 input 只能從 `App` 打開、`App` 一定畫 footer；PIN 框還要有 lib 的 `PasswordInput` 沒有的連錯冷卻、錯一次吞鍵一秒、逾時收起、疊在 custom 程式的畫面上。所以 `internal/ui/pinprompt.go` 照 lib 的畫法（components/input/password 的解鎖樣式、layout/popup 的框、標題膠囊、hint、錯誤列、開關動畫）自己畫一份，輸出層的 `ScreenOutput` 也照抄。回報成 LK1；lib 有入口就換掉。
- **custom saver 的 PIN prompt 底下不變暗（tdp F8；2026-09-28，使用者定案）。** tdp F8 要最上層以外的一切都 dim，但這時底下是使用者自己的程式的畫面：
  程式的輸出直接送到終端機，locku 只在上面畫框（`internal/custom` 的 screen writer），手上沒有那張畫面的副本。要 dim 就得在 locku 裡放
  一個終端機模擬器重畫程式的畫面——2026-09-25 否決過（多一個依賴、忠實度與效能都要驗）；使用者也定過「框之外什麼都不畫」。
  一般 saver 的畫布照 tdp F8 淡化：畫好的畫面每個顏色各自往 base 淡化（`dimANSI`）。同樣的淡化其實可以套在程式送出的輸出串流上，但只有
  PIN 框打開之後程式重畫的格子會變暗，靜態的畫面可能一直不變——仍不做（2026-09-28，使用者：維持偏離）。
（2026-10-18 以前還有兩條，已經不是偏離了：preference、tmux、screen 的 `[2]` 上 `?` 只有每列的說明、不是 key reference——lib 的 `?` 一定是 key reference，說明接在最後一節，見 LK3；PIN 框比 tdp F7 窄（框內 29 欄）——現在照 tdp F7 算，45 欄。）

## 設計文件導讀

三份設計文件每條決定就地標日期、被否決的做法也留著；`function.md` 的決定清單 96 條。2026-10-18 起，設定畫面長什麼樣、怎麼按，大多是 tdp 與 terminu-go 的事，`ui.md`、`ux.md` 只記 locku 自己的：每個 `[2]` 有哪些列、鎖定畫布、PIN 框、預覽。

| 檔案 | 回答什麼 | 順序 |
|---|---|---|
| [`function.md`](function.md) | 為什麼鎖站在 tmux / screen 之外、三種進入點同一契約、訊號表、狀態機、PIN 與無 PIN 模式、三種 saver、畫布渲染器、CLI、config、Integration 怎麼寫檔、決定清單、驗收 | 1 |
| [`ui.md`](ui.md) | 設定畫面兩個面板的 grid、鎖定畫布、每個欄位怎麼呈現、popup、PIN prompt 四個狀態、色帶、存檔 | 2 |
| [`ux.md`](ux.md) | core-key 語意、Space menu 內容、`?` 全域、每種欄位怎麼填、PIN 三連問、hotkey 分層、浮層、時間軸 | 3 |
| [`icon.svg`](icon.svg) | 圖示：黑底方塊上家族的方塊字 mark，深藍 U 包住金色的 L、O、C、K；splash 照它畫 | — |
| [`social-preview.png`](social-preview.png) | GitHub 的 social preview：icon 在上、下面 Tiny5 字樣 `locku`，`/social-preview locku` 產生（2026-10-08） | — |

Go、[terminu-go](https://github.com/vulcanshen/terminu-go)（設定畫面、寬度、淡化、探測，2026-10-18）、[Bubble Tea](https://github.com/charmbracelet/bubbletea) 與 [Lip Gloss](https://github.com/charmbracelet/lipgloss)（鎖定畫面）、custom saver 的程式與登入密碼驗證用 [creack/pty](https://github.com/creack/pty)、`charmbracelet/x/term` 與 `charmbracelet/x/ansi`、`muesli/cancelreader`、PIN 用 `golang.org/x/crypto/bcrypt`、config 用 `gopkg.in/yaml.v3`。色系 catppuccin-mocha。

## 建置與開發

從原始碼建置（Go 1.26+，`CGO_ENABLED=0` 靜態）：

```bash
git clone https://github.com/vulcanshen/locku.git
cd locku
make build      # → ./locku（-trimpath、strip）
make install    # → $GOBIN；make uninstall 移除
```

`go install …@latest` 也能裝，但版本字串只由 goreleaser 的 ldflags 注入，這樣裝出來的 `locku version` 是 `dev`。`make` 列出所有 target。

```bash
make check                                       # fmt-check + vet + go test -race；有 go.work 時再加 check-lib
make check-lib                                   # 關掉 go.work，照 go.mod 的 terminu-go 版本編譯與測試
make lock                                        # 編譯並鎖住這個終端機
make e2e                                         # 端到端：真的 tmux（e2e/tmux_attach.py，自己的 TMUX_TMPDIR）、custom saver（e2e/custom_lock.py）、真的 screen（e2e/screen_lock.py，自己的 SCREENDIR）；不碰你的 server 與 session
make gif                                         # 重錄 docs/demo.gif，見下方「demo gif」
make shots                                       # 重拍 docs/savers/*.png，見下方「saver 的靜態圖」
```

家族共用的 target 在 `family.mk`（tdp project 的範本，照原樣複製、不在這裡改），`Makefile` 只設 `BINARY := locku`、include 它，locku 自己的 `lock`、`e2e`、`shots` 放在最後一段（2026-10-18）。開發時用不進版控的 `go.work` 接本機的 lib：`go work init . ../terminu-go`；go.mod 寫的是發布過的 tag（v0.1.13），不用 `replace`。`make tidy`、`make snapshot`、`check-lib` 都關掉 `go.work`，跟 CI 一樣。

頂層目錄裡範本以外的：`e2e/`（python 的 pty 端到端）、`CLAUDE.md`（一行指向 2026-10-18 的重構交代）、`.claude/`（Claude Code 的 `social-preview` 指令與 agent，產 `docs/social-preview.png` 用，2026-10-08；`settings.local.json` 與 worktrees 不進版控）。

TUI 行為全部用 programmatic model test 驗證（不需要 tty）：設定畫面的測試（`internal/settings/settings_test.go` 的 `driver`）照 Bubble Tea 那樣餵訊息給 lib 的 `App`、把回來的指令跑完（popup 的動畫也是），再看 `App.Frame()` 與寫出去的 config.yaml，比一個 popup 的動畫長的計時器（toast）不等；`TestEveryRowIsTheTerminal` 在五種尺寸、icon 一格與兩格、每一種 popup 開著時量每一列的寬（tdp L4、tdp E5），四十秒上下。`make check` 帶 race detector（custom 的 pump 與 screen writer、login 的 su、custom 鎖的 prompt 各有 goroutine，沒有 race detector 看不出來，多花十幾秒）。tmux / custom / screen 的驗收（`docs/function.md` §12）以 python pty harness 跑真的 binary 完成：鎖定中 prefix d、prefix c、Ctrl-C 被吞、對 PIN 後 client 回來、鎖著的時候 attach 也被鎖、畫面上切 lock-session 時跑著的 server 整塊換掉、custom 的框疊在動畫上且程式被殺乾淨、screen 的 `C-a x` 與 `bind` 的鍵進 locku、`idle` 自動鎖、跑著的 session 即時收到設定、tty 關閉進程結束。

任何看狀態列文字的 ui 測試都要固定 `whoami`（`lockscreen_test.go` 的 `TestMain`），原因見「運作方式」的狀態列那條。

`V` 的 splash 彩蛋就是 `docs/icon.svg`，一格對一格（2026-09-26）：第一版在 icon 出現前一天畫，裡面是一個自創的掛鎖，icon 進來後沒跟上。2026-10-18 起是 lib 的 `NewSplash`：`internal/settings/icon.svg` 是 `docs/icon.svg` 的副本（`go:embed` 讀不到 `docs/`），`TestIconIsTheDocsIcon` 守住兩份一樣，icon 改了就要複製過去。揭露照 tdp S5：底 → 金色的 L、O、C、K（svg 裡註解的順序）→ 深藍 U 由下往上，下面是名稱、版本、`A screensaver with a PIN, for the terminal` 與作者。

### demo gif（2026-09-26；2026-10-07 重錄）

README 只放一個 gif，`docs/demo.gif`，用 VHS 錄。tape 與展示用 config 照家族慣例放在 `.local/demos/`（gitignore，不進版控）：`demo.tape` 與 `config.yaml`（PIN 1234，`htpasswd -nbBC 10 x 1234` 產生）。每次錄都把 config 蓋到 `.local/demos/config/config.yaml`、以 `LOCKU__CONFIG` / `LOCKU__DATA` 指過去，不碰真正的設定；tape 從不按 `activate`，所以不會寫到真的 `~/.tmux.conf`。需要 VHS、JetBrainsMono Nerd Font。

2026-10-07 重錄，每種 saver 都出場，順序使用者定：runner → pets → tetromino → snake → bounce → clock。結構也是使用者選的：鎖在 runner 開場（PIN 框疊在畫面上，錯一次再對），接著在設定畫面依序預覽其他五種，最後照舊是 Space 選單與 tmux；custom（cmatrix）拿掉，錄影因此不再需要 cmatrix。展示 config 的 profile 照這個順序排、名字就是 saver 的種類，啟用中的是 runner，所以設定畫面按 `j` 就照順序走，從 clock 再按 8 次 `j` 經過 Savers 到 tmux。pets 五隻、runner 用 `big-small`（我的判斷：五種花色都看得到、跑者有大有小）。runner 的天空是 time-shifting，照時鐘輪替，使用者要錄到黃昏：tape 在鏡頭外先跑一個迴圈，等到分鐘 % 3 = 1 那一分鐘的第 22 到 35 秒才開鏡（前 20 秒是從白天換過來的過場；runner 在畫面上約 11 秒，所以結束時還在黃昏），因此 `make gif` 最多要先等三分鐘。成品約 58 秒、6.4 到 6.9 MB（舊的 3.2 MB，多的主要是 runner 與 pets 的漸層；pets 的樹每次隨機，大小跟著差一點）。

VHS 0.12.0 在這台機器上會印 `Creating docs/demo.gif...` 卻不出檔（webu 也踩過），當時改用 0.11.0；2026-10-07 機器上只剩 0.12.1，正常出檔。VHS 的 `Type "…"` 不吃反斜線跳脫，字串裡要引號就用單引號；鏡頭外的指令用 `Type@5ms` 打，不然要等很久；等某個輸出用 `Wait+Screen@<逾時> /regex/`，要等的字在指令裡寫成 `RE''ADY`，免得一打出指令就符合。展示 config 設 `show_status: false`：狀態列會照實顯示錄影機器的 `user@host`，不公開進 README（2026-09-26）。

兩支 tape 的主題是 Catppuccin Mocha 的色票、終端機底色換成 crust `#11111b`（2026-10-08）：Mocha 原本的底色是 base `#1e1e2e`，跟夜晚跑者的身體同色，方塊之間的縫看不出來，README 的 runner 圖像一整塊暗色——正是使用者否決過的樣子（跑者用 crust 時方塊糊成一團）。使用者的終端機底色是更深的 `#11111a`，身體的方塊比縫亮一階、看得到格點；換成 crust 後錄出來的跟他看到的一樣，gif 與靜態圖一起換。

### saver 的靜態圖（2026-10-08）

README 的「What you see」每種 saver 一節，各配一張靜態圖 `docs/savers/<種類>.png`（使用者：不需要 gif）。用 VHS 的 `Screenshot` 拍：`.local/demos/savers.tape` 與 `savers.yaml`（七個 profile 照 Savers 的順序，`make shots`），在設定畫面逐一 `p` 預覽、等畫面有內容再拍。runner 是 city 配 night（使用者選的，新場景、夜裡一部分窗戶亮著）；custom 跑 cmatrix（使用者選的，所以拍照要裝 cmatrix）；snake 與 tetromino 用 `super-fast`、分別等 15 與 25 秒，畫面才不會空空的（我的判斷）；clock 用 `large`（使用者：全螢幕應該放得下 3 × 3），`HH MM SS` 的 large 要 148 欄，所以拍照的視窗是 1300 × 720、151 欄（gif 的 1200 寬只有 140 欄，放不下會先拿掉秒），七張一樣大。

畫面是隨機的，`make shots` 一次拍七張，挑過才進版控：runner 要有一棟完整的樓在跑者右邊；snake、tetromino 要沒有殘影。VHS 的畫面（headless 瀏覽器裡的 xterm.js）偶爾會在預覽開始後留著設定畫面左上角幾個字，snake、tetromino 的上面幾列不會再畫，殘影就一直留著；custom 偶爾整張是空的。同樣的按鍵節奏在 tmux 裡試 40 次都沒有殘影——tmux 記的是 locku 實際寫出的內容——所以是 VHS 顯示的問題，不是 locku 的（2026-10-08）。

README「The settings screen」那張 ASCII 圖是 tmux 擷取的真畫面（84 × 23，demo 的 profile、cursor 在 clock、焦點在 `[1]`；2026-10-18 換成 lib 畫的樣子），Nerd Font 的符號換成純文字：膠囊的兩端換成空格、radio 換成 `◉` / `○`，每列 84 欄（2026-10-08 那張是舊的設定畫面，78 × 21）。

## 發布

踩過的坑：

- **推 `.github/workflows/` 要有 workflow scope 的 token**（2026-09-25）：shell 的 `GITHUB_TOKEN` 是沒有 workflow scope 的 PAT，推到 workflow 檔會被 GitHub 拒絕；那一次改用 gh 的 keyring token 推（`env -u GITHUB_TOKEN` 再走 `gh auth git-credential`）。其他 push 照常用 shell 的。
- **Homebrew tap 的 formula 在 tap repo 的根目錄**（`locku.rb`），不在 `Formula/` 底下；核對 sha256 時看那裡。
- **看 Actions 時 `gh run list`、`gh run watch <id> --exit-status`、`gh run view <id>` 分開跑**：串成一行的 shell 指令被安全檢查誤判成刪檔而擋下（2026-10-08）。
- **五個 app 推同一個 homebrew-tap**：一次只發一個，前一個的 release 跑完、tap 更新了才打下一個的 tag，不然 tap 的 push 會互撞。
- **發版前 `make check-lib`**：本機有 `go.work` 指著還沒發版的 terminu-go 時照樣過；關掉 `go.work` 照 go.mod 的版本也要過，CI 沒有 `go.work`。

發版走家族的流程（tdp project 的「發布」）：`make check` → CHANGELOG 的 `## Unreleased` 改成 `## vX.Y.Z — tdp vA.B.C` → commit、打 tag `vX.Y.Z` → push。GitHub Actions（`.github/workflows/release.yml`，範本）在 ubuntu 與 macOS 跑 `go test -race`，過了 goreleaser 打包四個平台的 tarball 與 checksums、更新 vulcanshen/homebrew-tap 的 `locku.rb`，再用 CHANGELOG 那一節蓋過 release 的內文（awk 認 `$2 == tag`，所以標題開頭一定是 tag）。v0.2.7 以前的標題照當時的寫法（`## [0.2.7] - 2026-10-08`），不改。CHANGELOG 只記使用者感覺得到的變動；純文件、打包、CI 的改動不記。
