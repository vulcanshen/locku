# locku — UI

> 本文件講**版面與 surface**：兩個畫面、popup、色帶、chrome、存檔。按鍵語意與流程在 `ux.md`，
> 功能邊界在 `function.md`。依 VTP（`thoughts/tui-design`）與 u-family Popup Convention 撰寫，
> 每條版面決定標日期。v1.0 定案，2026-09-24；2026-09-25 對齊程式碼重寫。

---

## §1 兩個畫面

locku 有兩個彼此獨立的畫面，由 CLI 決定進哪一個，執行期間不互切（Preview 例外，見 §2.1）：

| 畫面 | 指令 | 性質 |
|---|---|---|
| 設定 | `locku` | 兩個面板的 u-family 版面，吃完整 VTP |
| 鎖定 | `locku lock`、argv[0] = SCREEN-LOCK | 全螢幕畫布，零 chrome，VTP 只作用在它的 popup |

### 1.1 設定畫面 grid

```
╔[1] locku═════════════╗╭[2] clock  unsaved─────────────────────────────╮
║ Profiles               ║│ Property          Value                          │
║ ● clock                ║│ name              clock                          │
║   clock2               ║│ saver             clock                          │
║   dino                 ║│ layout            row                            │
║ Savers                 ║│ size              medium                         │
║   clock                ║│ font              3x7                            │
║   dino                 ║│ time              HH MM                          │
║ Integration            ║│ date              off                            │
║   tmux                 ║│ bg                ■ #313244  →  ■ #ff3244        │
║   screen               ║│   R               ───────────● 255               │
║ Settings               ║│   G               ──●───────── 50                │
║   preference           ║│   B               ───●──────── 68                │
║                        ║│ fg                ■ #f2b753                      │
║                        ║│   R               ──────────●─ 242               │
║                        ║│   G               ────────●─── 183               │
╚════════════════════════╝╰──────────────────────────────────────────────────╯
 space menu   ? help   tab/1-2 panels   q quit                                  ← footer
```

每個 `[2]` 第一列是表頭：`Property` 與 `Value`（單數，2026-09-25 同日改），用側欄區塊標題的 Blue，不可停，cursor 從第二列起（2026-09-25，使用者：所有 panel 2
都給標題列）。

左 `[1]` 側欄四個區塊，順序 Profiles → Savers → Integration → Settings（2026-09-24 定案前三個，使用者以 OOP 分：saver 是 class、profile 是
object，常用的 profile 在上；2026-09-25 加 Integration）：**Profiles** 列出使用者設定好的、有名字的 saver 實例，新增（從 Savers 的一種按 `n`）、
複製、改名、刪除都在這裡；**Savers** 列出有哪幾種 saver（clock、dino、custom），它們沒有名字、名字就是自己，不能新增刪除；
**Integration** 兩項：`tmux`、`screen`，`[2]` 是 `activate`（on / off，就是區塊在不在設定檔裡）、`config file path`、一條分隔線、然後工具自己的 key（tmux 的 lock、lock-after-time、bind-key；screen 的 idle、bind）（2026-09-25，使用者定案）；
**Settings** 一項：`preference`。區塊標題 Blue、是
分隔，不可停，區塊之間不空列；cursor 只在項目之間走，開啟時停在啟用中的 profile。啟用中的 profile 前面一顆 Green `●`，
是側欄唯一的綠色；它只顯示，設為啟用在 `preference › profile`。

右 `[2]` 是**明細**，內容跟著 `[1]` 的 cursor 即時切換，不用 Enter，明細沒有切換成本：cursor 在 saver 上是它的說明加預設值；在 profile 上就是那個 profile 的欄位與顏色（custom 只有 `command`，沒有顏色列，2026-09-25）；在 tmux / screen 上是整合的列；在 `preference` 上是一般設定的列。title 是膠囊（§5）：`[2] clock`、`[2] tmux`、`[2] preference`——只有名字，不標種類、不放狀態（2026-09-25，使用者：分類資訊多餘）；profile / saver 的顏色草稿未存時接一顆黃色 `unsaved`。

`[2]` 在 saver 上：先三列唯讀說明，再一列 dim 的 `defaults` 標題，然後是**這種 saver 的預設值**——跟 profile 一模一樣的欄位列與
色票 / slider 列（沒有 name），同樣的 options popup、顏色草稿、`S` / `R`、` · unsaved`（2026-09-24 定案）。改預設值只影響之後
用這種 saver 新增的 profile，不動既有的；`[p]` / `[P]` 用預設值跑一個臨時 profile 預覽；`[n] New` 生 profile。

| 列 | 值 |
|---|---|
| saver | `clock` / `dino` / `custom` |
| what | 一句話：clock 是 the time and the date, on the LED board；dino 是 the offline dino run, jumping by itself, for ever；custom 是 your own program, on a terminal of its own, as the saver |
| profiles | 是它的 profile 名，逗號分隔；沒有就 `none yet` |
| defaults | dim 標題：`for profiles made of it from now on` |
| （預設值） | clock：layout / size / font / time / date；dino：runner / scene；這兩種再 bg / fg 各一色票列加 R G B；custom：只有 command，沒有顏色列 |

`[2]` 在 profile 上：

| 列 | 值 | Enter |
|---|---|---|
| name | 實例名 | input popup，型別 `name`；重複或空被擋 |
| saver | `clock` / `dino` / `custom` | 唯讀，dim，不可停：profile 的 class，要換就從 Savers 新增一個 profile（2026-09-24 定案；當天曾短暫可改）；底下的列跟著 saver 換 |
| layout | `row` / `column` | options popup，cursor 在目前值（clock） |
| size | `small` / `medium` / `large`（一個字型像素 1 / 2 / 3 格見方） | options popup，cursor 在目前值（clock；dino 沒有 size，畫布自己取最大） |
| font | `3x7` / `3x5`（字型高 7 列或 5 列） | options popup，cursor 在目前值 |
| time | `HH MM` / `HH MM SS`（2026-09-24：拿掉 12 時制，時間不畫冒號） | options popup，cursor 在目前值 |
| date | `off` / `YYYY-MM-DD` / `YYYY-MMM-DD` / `MM-DD` / `MMM-DD` | options popup，cursor 在目前值；不是 off 時畫布第二列（clock） |
| runner | `big` / `small` / `big-big` / `small-small` / `small-big` / `big-small`（dino：一隻大或小暴龍，或兩隻一前一後、名字就是畫面由左到右的順序、各自跳；2026-09-25 修訂，舊值 `trex` / `two-trex` 自動轉成 `big` / `big-small`） | options popup（2026-09-24） |
| scene | `grassland` / `desert`（dino：草原是仙人掌，沙漠是金字塔） | options popup（2026-09-24） |
| command | custom（2026-09-25）：使用者自己的指令，`sh -c` 跑；未設 `not set`（Yellow） | input popup，型別 `command`，預填目前值；清空 = 未設 |
| bg / fg | 一格該色的 glyph 當色票 + hex，是**已存**的顏色；草稿不同時右邊接 `→` 加草稿的色票 + hex | 不可停 |
| R / G / B | webu 的 slider 列：12 格軌道 + 草稿的值，軌道用**該通道自己的顏色**畫——R 列是 `#RR0000`、G 列 `#00GG00`、B 列 `#0000BB`，值多大顏色就多亮；軌道底色反向，0 時全白、255 時全黑，暗的值才看得見；數字是 Mauve、沒有底色，跟其他列的值一樣（2026-09-24） | options popup：0 到 255 的數字清單，10 列一窗、游標在目前值置中，Enter 移過去（webu slider 作法，不打字）— 改的是草稿 |

顏色走**草稿**（修訂 2026-09-24，使用者調歪過一次調不回來）：滑桿改草稿，色票列同時看得到已存與草稿，
panel operation `[P] Preview` 預覽這個 saver、`[S] Save` 寫檔、`[R] Reset` 丟掉草稿；草稿跟著 saver 的名字走
（rename 帶走、delete 一起丟）。預覽都帶著草稿。其餘欄位仍立即寫檔。手改 config 的非法 hex 視同預設。
label 欄固定 18 欄，只在面板窄到放不下 label 加值時才縮（修訂 2026-09-24：原本上限是面板寬的三分之一，
一般寬度的終端機就把最長的 label 壓到貼著值；2026-09-24 改名後最長的是 `wrong_pin_attempt_cooldown`，26 字，label 欄放寬到 28）。

`[2]` 在 `preference` 上：

| 列 | 值的呈現 | Enter |
|---|---|---|
| PIN | `set`（Green）/ `not set`（Yellow） | 未設：設定流程；已設：current PIN 之後選 `New PIN` / `Remove PIN`（2026-09-24，取消併進同一條流程） |
| profile | 啟用中的 profile 名；指向不存在的加 ` (missing)` Yellow | options popup 列出所有 profile、cursor 在目前值，Enter 寫檔、側欄 `●` 移過去 |
| show_status | `on` / `off` | 原地翻轉，不開 popup |
| pin_prompt_timeout | 數字 | input popup，型別 `number` |
| wrong_pin_attempts | 數字，0 顯示 `0 (off)` | 同上 |
| wrong_pin_attempt_cooldown | 數字 | 同上 |

每一列是什麼，focus 在這個 `[2]` 時按 `?`：help 裡**只有**這幾列的說明，一列一項、太長就在說明欄內自動換行，沒有鍵的清單
（2026-09-25：原本每列下面接一列 dim 說明，2026-09-24 加的，使用者要搬到 help，而且 `[2]` 上只要這些；`[2]` 因此只剩設定列）：PIN `what the lock asks for; with none, any key unlocks`、profile `the profile the lock shows`、
show_status `user@host and the time, on the lock's last row`、pin_prompt_timeout `seconds without a key before the PIN box
closes; 0 never`、wrong_pin_attempts `wrong PINs in a row before a cooldown; 0 off`、wrong_pin_attempt_cooldown `seconds the
cooldown lasts`。列數超過面板時跟著 cursor 捲。

`[2]` 在 Integration 的 tmux / screen 上（2026-09-25，使用者定案：property / value 兩欄，跟 profile 一樣），標題 `[2] tmux` / `[2] screen`：

| 列 | 呈現 | 編輯 |
|---|---|---|
| activate | `on`（Green）/ `off`（Mauve）：區塊在不在 config file path 裡，每次畫都讀一次 | Enter → confirm popup 才執行：on 把區塊寫進檔案（tmux 有 server 在跑就整塊 `source-file` 上去；screen 連 shell rc，跑著的 session 即時 `screen -X`），off 拿掉（server 上的、跑著的 session 上的一併拿掉）；config file path 沒填時 disabled 並說 `set the config file path first` |
| config file path | 路徑照存的樣子；未設 `not set`（Yellow），activate 因此 disabled | input popup，型別 `path`，webu 的提議作法（`ux.md` §2.1），提議 `~/.tmux.conf` / `~/.screenrc`；on 的時候改路徑，區塊搬到新檔、清空就拿掉 |
| ── 分隔線 | Surface2 一條線，不可停：上面是 locku 的設定，下面是寫進工具設定檔的 key | 無 |
| lock（只有 tmux） | `lock-server` / `lock-session`：鎖的範圍——整台 server，或只有觸發的那個 session（別的 session 照常）；值用 tmux 的指令名；`?` 說明只講範圍。screen 沒有 server、沒有範圍可選，不硬造這列 | options popup，兩個值；on 時改了立刻重寫區塊、server 換旗 |
| lock-after-time（tmux）/ idle（screen） | 數字，0 顯示 `0 (off)`：閒置幾秒自動鎖，列名就是工具自己的設定名稱，activate on 時原樣寫進去、一改就重寫，各工具一份 | input popup，型別 `number`，清空 = 300 |
| bind-key（tmux）/ bind（screen） | 鍵照工具自己的寫法——tmux `l`、`C-l`、`F12`，screen `l`、`^L`；空顯示 `none`：prefix / C-a 之後按它就鎖，寫成 `bind-key <鍵> <lock>` / `bind <鍵> lockscreen`；screen 的 `C-a x` 內建就鎖，`?` 說明會講 | input popup，型別 `key`，預填目前值；清空 = 不綁；含空白或 `#` → ` · one key, e.g. l or C-l`（screen：`l or ^L`）框留著 |

**開一次，之後隨設即得**（2026-09-25，使用者定案）：`activate` on 時任何一列改動就直接重寫區塊、tmux 整塊套到 server、screen 即時送進跑著的 session，config file path 改路徑就把區塊從舊檔搬到新檔、清空就拿掉，做完 toast 一行結果；off 就只寫 config.yaml，activate 仍由使用者開。
`?` help 的鍵清單說 Enter 在 activate 上做什麼；focus 在這個 `[2]` 時 `?` 只有 activate、config file path、lock（tmux）、lock-after-time / idle、bind-key / bind 的說明，screen 再多一條 `LOCKPRG`：鎖本體不是一列、住在 shell rc、新開 shell 才有。

`profiles` 與 `savers` 這兩個 key 不成列：它們就是 `[1]` 本身。其餘每個 config key 一定有一列。
除了顏色草稿，每次改完立即寫檔，沒有 Save 鍵，沒有 dirty 狀態。寫檔失敗以 toast 報錯，值退回。

### 1.2 鎖定畫布 grid

```
□ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □
□ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □
□ □ □ □ □ ■ □ □ ■ ■ ■ □ □ □ □ ■ ■ ■ □ □ ■ □ ■ □ □ □ □ □ □ □ □ □   ← 亮格 saver 的 fg，預設 gold
□ □ □ □ ■ ■ □ □ □ □ ■ □ ■ □ □ □ □ ■ □ □ ■ □ ■ □ □ □ □ □ □ □ □ □     暗格 saver 的 bg，預設 surface0
□ □ □ □ □ ■ □ □ ■ ■ ■ □ □ □ □ ■ ■ ■ □ □ ■ ■ ■ □ □ □ □ □ □ □ □ □     每格 = nf-fa-square + 空格
□ □ □ □ □ ■ □ □ ■ □ □ □ ■ □ □ □ □ ■ □ □ □ □ ■ □ □ □ □ □ □ □ □ □
□ □ □ □ □ ■ □ □ ■ ■ ■ □ □ □ □ ■ ■ ■ □ □ □ □ ■ □ □ □ □ □ □ □ □ □
□ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □
□ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □
 vulcan@prod-db-01 · locked since 13:58                           ← 狀態列，一般文字
```

整個終端機是一塊 LED 點陣板：狀態列以外的每一格都是 nf-fa-square 加空格，沒亮的用 saver 的 bg，亮的用
它的 fg，字由亮格組出來。沒有邊框、沒有 title chip、沒有 footer。footer 在這裡沒有意義：唯一的動作是
「按任何鍵」，不需要揭露。**只有一種畫法**：saver 給幾行 ASCII（row 一到兩行，column 依分隔符拆行），渲染器
（`function.md` §5.3）依 saver 的 size 以 1 / 2 / 3 倍畫，字距、行距與時間裡的空白是獨立的間隔單元（1 / 1 / 2 格，不跟著等比放大），塞不下先降 size 再去單位，置中。
splash 底下的名字、版本、開發者都不出現，只取它的 glyph 畫法。

| 元素 | 位置 | 規則 |
|---|---|---|
| 點陣板 | 狀態列以外全部 | 每格一個 glyph 加空格；終端機寬為奇數時最右一欄留白 |
| 暗格 | 沒亮的格 | saver 的 bg，預設 surface0 |
| 內容 | 點陣板置中 | 亮格 saver 的 fg，預設 gold；1 倍也塞不下時點陣板照鋪，內容改用一般文字以 fg 色置中疊在板上 |
| 狀態列 | 最後一列，左起 1 欄 | `user@host · locked since HH:MM`；`show_status: false` 時整列空白 |
| 無 PIN 提示 | 狀態列右側接續 | `· no PIN · any key unlocks`，Yellow，不受 show_status 影響 |
| config 錯誤 | 同上 | `· config error: <reason>`，Red |
| custom 結束原因 | 同上 | `· custom saver: exit 3 · boom`，Red（2026-09-25） |
| 區塊 | 時間先、日期後 | 兩個獨立區塊：row 日期在時間下面，column 日期一欄在左、時間一欄在右；各自先降 size 再去單位，日期塞不下就不畫，`function.md` §5.3 |

custom saver 的程式結束時（`function.md` §5.5）板子照實寫 **`EXIT <code>`** 或 **`NONE`**：clock 的 3x7 字型，large → medium → small 退階，放不下就純文字；`EXIT` 四個字母是預設的 gold，數字 0 是 Green、其他是 Peach，`NONE` 整個 Red；暗格預設 surface0（custom 沒有自己的顏色）；狀態列紅字寫原因（2026-09-25，使用者定案）。程式還在跑時畫布上沒有 locku 的東西：畫面是程式的，PIN prompt 的框直接疊在上面（§2.3、§3.2）。

resize：重算 k 整張重畫。動畫：第一幀直接出現不動畫；之後內容變更（clock tick）只對有變的像素做 splash 式
shuffle 揭露，沒變的像素不動，一次變更 ≤ 400 ms。

### 1.3 尺寸規則

| 項目 | 規則 |
|---|---|
| 側欄寬 | 固定 24 欄，與 webu 同 |
| `[1]` 高 | footer 以上全部；saver 多到放不下時捲動 |
| 窄寬 | `w < 60` 只畫焦點那一側；鎖定畫布依 1.2 退化 |
| chrome | 設定畫面 footer 1 列鎖死；鎖定畫布 0 列 |
| 寬度穩定 | 每一列恰好終端機寬（跨尺寸測試同 webu）；label 不截斷、值截斷不折行 |

---

## §2 職責

### 2.1 `[1]` 側欄

- 每一列的 Enter 都只是把焦點送到 `[2]`，saver 與 profile 也一樣（修訂 2026-09-24，原為 profile 上 Enter = 設為啟用；
  設為啟用改在 `preference › profile`，`●` 純顯示；saver 上的新增是 `n`，不佔用 Enter）。preview / duplicate / rename / delete 在 Space menu 的
  item region。
- **Preview** 有兩個入口：`[2]` 上的全域 `P`——在 profile / saver 的 `[2]` 是那一個，在 preference / tmux / screen 的 `[2]` 是啟用中的 profile；側欄 saver / profile 上的 `[p] Preview` 看游標那一個，不改啟用；`[1]` 上 `P` 不作用（2026-09-25，使用者）。整合設定在側欄 Integration 的 tmux / screen 的 `[2]` 第一列 `activate`（2026-09-25）。

**Preview**：一個動作，整個 TUI 被鎖定畫布取代，就像桌面螢幕保護程式的預覽。同一個進程、用記憶體內的 config
加上顏色草稿，任意鍵回到設定畫面、焦點與 cursor 不變，不驗 PIN（2026-09-25）。custom 的預覽把終端機交給程式（同進程的 exec），任意鍵殺掉回來；程式結束或沒填指令就在畫面內以板子上的字（`EXIT <code>` / `NONE`）預覽。


### 2.2 `[2]` 明細

單一職責：顯示並編輯 `[1]` 目前指到的那一項。不做新增、刪除，那是 `[1]` 的事。

### 2.3 鎖定畫布

單一職責：畫內容與狀態列。所有按鍵（含 Ctrl 組合，`function.md` §2.1）只做一件事：開 PIN popup；
無 PIN 模式則結束進程。PIN popup 開著時亮格改畫 Surface2、暗格不變，當 backdrop；saver 照常 tick、揭露照常動（修訂 2026-09-24：原本停 tick、一次重畫不再動，使用者要的是背景變色但不停）。custom saver（`function.md` §5.5）：畫布是程式的畫面，PIN prompt 的框直接疊在還在動的畫面上——沒有 backdrop、沒有變色，因為底下沒有 locku 的格；框收起時它佔過的位置清掉，畫面自己補回來（2026-09-25）。

---

## §3 Popup

全部走 u-family Popup Convention：一個 popup 一個檔一個 animator、title = glyph + 文字、hint 嵌
下邊框、`Esc` 只在 `closeTop` 一處解析、動畫 8 × 16 ms。層數最多兩層。

### 3.1 設定畫面

| Popup | 類型 | 用途 |
|---|---|---|
| Space menu | menu | `[1]` saver / profile 的 item region；`[2]` 欄位的 item region；`[2]` 在 profile 或 saver 上另有 panel region（Preview / Save / Reset，saver 再加 New）— 兩個 region 各有 header，只有一個就扁平 |
| `?` help | viewport | 全域動作表 |
| input | input | **邊框寫型別**（`name`、`number`、`path`、`number · invalid`、`name · taken`），框內一行是欄位名，目前值當提議；清空 = 預設值；new profile 的 `name` 提議 saver 自己的名字、被用了就加號碼 |
| PIN input | input，遮罩 | 邊框 `current PIN`、`new PIN`、`confirm PIN`；**畫法與鎖定畫布的 PIN prompt 完全相同**（§3.2）：48 欄、上下留一列、`●` 之間空一格、從中央向兩側長（2026-09-24，使用者要求解鎖與設定一樣） |
| options | menu | layout / size / font / time / date / runner / scene / profile 的清單；R G B 的 0–255 清單 10 列一窗；current PIN 之後的 `New PIN` / `Remove PIN`（2026-09-24） |
| confirm | message | Delete profile、Quit（有未存的顏色草稿時） |
| toast | message | 寫檔失敗、PIN 不一致、不可刪（啟用中 / 最後一個）、nothing to save / nothing changed |

new 與 duplicate 都是 `name` input popup：new 提議 saver 的名字（`dino`，用了就 `dino2`），確認後以那種 saver 的預設值生一個
profile；duplicate 提議原名加 `2`，確認後複製參數。兩者都把 cursor 移到新 profile、焦點送到 `[2]`。
PIN 設定與更改是**同一種 popup 連續開**（`current PIN` → options `New PIN` / `Remove PIN` → `new PIN` → `confirm PIN`），
一次只問一件事，錯在哪一步就停在哪一步；`Remove PIN` 按 Enter 立即生效、不 confirm（2026-09-24）。

### 3.2 鎖定畫布

| Popup | 類型 | 用途 |
|---|---|---|
| PIN prompt | input，遮罩，寬固定 48 欄置中，框內上下各留一列；`●` 之間空一格，從框的橫向中央開始、向兩側長；設定畫面的三個 PIN 框同一個畫法（2026-09-24：原本 32 欄、靠左、不空格） | 唯一的 popup |
| PIN prompt（custom saver） | 同一個框、同一套狀態，由一個沒有 renderer 的 lock 程式透過 callback 交給 custom 的 screen writer 畫在終端機正中央、疊在程式還在動的畫面上：程式每送一段輸出就在後面補畫一次（DECSC / DECRC 包住、一次 `?2026` synchronised update）；收起時清空它佔過的矩形（2026-09-25，使用者定案） | custom 鎖定中唯一的 popup |

四個狀態，全部只改**邊框**與 title，框內一行不變：

```
╭ PIN ──────────────────────────╮     idle：layer 色
│ ●●●●                          │
╰──── enter unlock · esc back ──╯

╭ PIN · wrong ──────────────────╮     wrong：Red，1 秒，框內清空，吞輸入
│                               │
╰───────────────────────────────╯

╭ PIN · try again in 27 s ──────╮     lockout：Red，倒數，吞輸入，Esc 仍可回 saver
│                               │
╰───────────────────────────────╯

╭ PIN · closing ────────────────╮     pin_prompt_timeout 到：正常關閉動畫回 saver
```

錯誤與 lockout 的 Red 是 override 色（VTP §2.4），不參與層級。

---

## §4 色帶

錨點 catppuccin-mocha，與家族相同。點陣板的兩色是**使用者資料**（每個 saver 的 bg / fg），不屬於 app 色帶，預設值取 splash。

| 色帶 | 意思 | 值 |
|---|---|---|
| Blue | focus：焦點面板邊框；側欄的區塊標題（修訂 2026-09-24） | `#89b4fa` |
| Surface2 | unfocused 面板邊框；PIN prompt 開啟時亮格的 backdrop 色 | `#585b70` |
| Green | 使用者足跡：啟用中的 profile `●`、PIN `set`、toggle `on`、activate `on`；custom 板子上 `EXIT 0` 的 0（2026-09-25） | `#a6e3a1` |
| Mauve | 可填的：`[2]` 的值 | `#cba6f7` |
| saver 的 fg | 點陣板亮格，使用者可改 | 預設 gold `#f2b753` |
| saver 的 bg | 點陣板暗格，使用者可改 | 預設 surface0 `#313244` |
| Overlay0 | 狀態列、hint、唯讀的 type 列與色票列 | `#6c7086` |
| Peach | custom 板子上非 0 的結束碼（`EXIT 3` 的 3）（2026-09-25） | `#fab387` |
| Yellow（warn） | `not set`（PIN、command、config file path）、`no PIN · any key unlocks`、`unsaved` | override |
| Red（error） | PIN wrong、lockout、`· invalid`、`· taken`、`config error`；custom 板子上的 `NONE` 與狀態列的結束原因（2026-09-25） | override |
| popup layer scale | 浮層邊框，最多兩層 | VTP §2.5 |

focus 二態同 kbu §8.4：雙線 `╔═╗` + Blue ↔ 圓角細線 `╭─╮` + Surface2，零位移。畫布沒有焦點概念，
Blue 不出現在那裡。

---

## §5 Chrome

| 件 | 設定畫面 | 鎖定畫布 |
|---|---|---|
| Border title chain | 家族的 powerline 膠囊鏈（2026-09-25，取代 ` · ` 分隔的純文字：sshu `panelChip`、filu `singleChip`、webu `tabChain` 都是膠囊）：`[1] locku` 一顆；`[2]` 左上角 `[2] <名字>`（邊框色底、深色字），顏色草稿未存時接一顆 `unsaved`（focus 時 Yellow，沒 focus 整條 Surface2）；沒有種類、沒有狀態、沒有 config 路徑（使用者：分類資訊多餘，狀態是 `activate` 列）；膠囊字緊貼圓頭 cap、不留空白（同 sshu / filu），接縫兩側各一格、底色不同是左邊那顆的實心斜切、相同是 canvas 色細斜線；寬度不夠先丟 `unsaved` | 無 |
| Panel tab bar | 無 | 無 |
| Border hint | 無 | 無 |
| footer | `space menu   ? help   tab/1-2 panels   q quit` | 無 |

custom saver 鎖定中，終端機上只有程式的畫面與（開著時）PIN 框，沒有 locku 的任何 chrome（2026-09-25）。

**Nerd Font 是設計、必裝**，與家族相同：畫布像素就是 nf-fa-square。字型在使用者本機的終端機模擬器，
經 SSH 不受影響。`docs/icon.svg` 沿用 u-family mark 的 locku 版；`V` splash 彩蛋家族同鍵，只在設定畫面。

---

## §6 存檔

| 資料 | 形式 | 位置 |
|---|---|---|
| 設定 | `config.yaml`，見 `function.md` §7 | `~/.config/locku`（`$XDG_CONFIG_HOME/locku`、`$LOCKU_CONFIG`） |
| PIN reset 紀錄 | `pin-resets.log`，一次 reset 一行（時間、`user@host`、結果），不含 PIN（2026-09-25） | `~/.locku/data`（`$LOCKU_DATA`） |

沒有 cache、沒有 history。config 寫檔原子（temp + rename），權限 600；log 追加寫、600，目錄 700；目錄不存在時建立。

---

## §7 待決

無。2026-09-24 全部定案。
