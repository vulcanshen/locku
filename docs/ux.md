# locku — UX

> 本文件講**互動語意**：core-key、Space menu 內容、hotkey 分層、每種輸入怎麼填、浮層行為、時間軸。
> 版面與 surface 在 `ui.md`，功能邊界在 `function.md`。依 VTP（`thoughts/tui-design`）撰寫，章節編號
> 對齊 webu；每條決定標日期。v1.0 定案，2026-09-24。

---

## §A. VTP in locku

### §A.0 揭露對照

| Track | 入口 | 入口自身怎麼被揭露 | 完整性 |
|---|---|---|---|
| **Contextual** | `Space` | footer 常駐 `space menu` | 當前 focus 的 contextual 動作 100% 在 Space menu 內 |
| **Non-contextual** | `?` | footer 常駐 `? help` | 全域動作 100% 在 help 內 |

**規則：一個操作沒進 Space menu 或 help 就等於不存在。** core key 沒有字母可以括，鍵寫進 label：
`[Enter] Edit`、`[Enter] Choose`。

**鎖定畫布不在這張表裡。** 它只有一個動作「開 PIN prompt」，而且任何鍵都是它，沒有第二個動作可揭露，
所以畫布上沒有 Space menu、沒有 `?`、沒有 footer。VTP 只作用在 PIN prompt 這個 popup。

### §A.0.K core-key 語意

| Core-key | 設定畫面 | 鎖定畫布 |
|---|---|---|
| `Tab` | `[1]` ↔ `[2]`；`1` / `2` 直達 | 任何鍵 = 開 PIN prompt |
| `Enter` | `[1]` 任何列：焦點送到 `[2]`（修訂 2026-09-24，saver 也一樣，設為啟用改在 preference）；`[2]` 欄位：開該欄的 popup 或原地翻轉（§2） | saver 上：開 prompt；prompt 內：送出 |
| `Esc` | 關最上層浮層；沒浮層時 no-op，不離開 app | prompt 內：回 saver；saver 上：跟任何鍵一樣開 prompt |
| `Space` | **滑鼠右鍵 context menu**；再按關閉；在 menu / viewport / message 浮層上 = 關掉它；input 打字中是空白字元 | saver 上：開 prompt；prompt 內：PIN 的一個字元 |
| `?` | help；再按關閉；可疊在任何浮層上 | saver 上：開 prompt；prompt 內：PIN 的一個字元 |

無 PIN 模式（`function.md` §4.3）：畫布上任何鍵 = 結束進程，沒有 prompt。

### §A.1 Contextual track — Space menu

region 固定叫 `item operation` / `panel operation`；只有一個 region 就保持扁平；menu 本身 `j`/`k` 走（環繞）、
`u`/`d` 半窗、`gg`/`G` 首尾、Enter 執行、letter hotkey 在 menu 裡也有效。有 panel operation 的是 `[2]` 在
profile / saver 上：顏色草稿的 Save / Reset（修訂 2026-09-24）。tmux / screen 的 Install / Uninstall 不是 panel operation，是 `[2]` 底部分隔線下
按鈕的 Enter（2026-09-25，取代同日早上的 `[S]` / `[X]` 熱鍵——畫面上看不到；更早是 CLI `locku setup`）。

**`[1]` 側欄**

| cursor 在 | item operation |
|---|---|
| saver（class：clock、dino） | `[Enter] Edit`（焦點送到 `[2]`：說明與預設值）、`[p] Preview`（用預設值跑一個臨時 profile）、`[n] New`（name popup，提議 saver 的名字、用了就加號碼；確認後以預設值生一個這種 saver 的 profile、cursor 移過去、焦點送到 `[2]`）（2026-09-24） |
| profile | `[Enter] Edit`（焦點送到 `[2]`）、`[p] Preview`（鎖定畫布顯示這個 profile，不改啟用）、`[D]uplicate`（name popup，提議原名加 `2`）、`[r]ename`（name popup）、`[X] Delete`（confirm；最後一個或啟用中 disabled 並說明） |
| tmux / screen（Integration，2026-09-25） | `[Enter] Edit`（焦點送到 `[2]`：conf、閒置鎖、bind-key 與底部的 Install / Uninstall 按鈕）；同日拿掉 `[S] Setup` / `[X] Remove` 熱鍵 |
| preference | `[Enter] Edit`（焦點送到 `[2]`） |

**`[2]` 明細**

| cursor 在 | item operation | panel operation |
|---|---|---|
| saver 的說明列 | 唯讀，不可停 | saver 上：`[n] New`、`[P] Preview`（用預設值）、`[S] Save`、`[R] Reset`（預設值的顏色草稿） |
| name | `[Enter] Rename` | profile 上：`[P] Preview`（這個 profile，帶草稿）、`[S] Save`、`[R] Reset`（顏色草稿；沒草稿時 disabled 並說明） |
| saver | 唯讀，不可停（profile 的 class；2026-09-24 定案） | 同上 |
| layout / size / font / time / date / runner / scene | `[Enter] Choose`（dino 的列只有 runner / scene；saver 上改的是預設值，只影響之後新增的 profile） | 同上 |
| bg / fg 色票列 | 唯讀，不可停 | 同上 |
| R / G / B | `[Enter] Pick`（進草稿） | 同上 |
| PIN | 未設：`[Enter] Set PIN`；已設：`[Enter] Change PIN`（current PIN → 選單 `New PIN` / `Remove PIN`；2026-09-24 拿掉 `[x] Clear PIN`，取消併進同一條流程） | 無 |
| profile | `[Enter] Choose`（列出所有 profile） | 無 |
| show_status | `[Enter] Toggle` | 無 |
| pin_prompt_timeout / wrong_pin_attempts / wrong_pin_attempt_cooldown | `[Enter] Edit` | 無 |
| conf（tmux / screen 的 `[2]`） | `[Enter] Edit`（裝著時改路徑，區塊搬到新檔；清空就拿掉） | 無 |
| lock-after-time（screen：idle） | `[Enter] Edit`（裝著時直接重寫區塊、tmux 即時套用） | 無 |
| bind-key（tmux） | `[Enter] Edit`（同上） | 無 |
| Install / Uninstall（`[2]` 底部分隔線下的按鈕，置中，cursor 到才點亮） | `[Enter] Install`（confirm 後把區塊寫進 conf；tmux 有 server 在跑就即時套用；screen 連 shell rc）/ `[Enter] Uninstall`（confirm 後拿掉，tmux 連 server 上的一併拿掉）；conf 沒填時 disabled 並說 `set conf first`（2026-09-25） | 無 |

修訂（2026-09-24）：duplicate / delete 改成 `D` / `X` 大寫，對齊 sshu 的紀錄類項目；`d` 仍是半頁。

**popup 內**：options / confirm / viewport / toast 沒有自己的 Space menu，Space 就是關掉。

### §A.2 Non-contextual track — `?` help

| 全域動作 | 鍵 |
|---|---|
| Preview（整個畫面被鎖定畫布取代，帶著顏色草稿，解鎖後回來；在 saver 的 `[2]` 上是那個 saver，其他地方是啟用中的） | `P` |
| 切面板 | `Tab`、`1` / `2` |
| 離開 | `q`（浮層內不作用；有未存的顏色草稿時先 confirm）、`Ctrl+C` 硬退 |
| splash 彩蛋 | `V`（不揭露） |

全域字母在浮層開著、打字中兩種狀態下不作用。鎖定畫布與 Preview 中沒有全域鍵，`Ctrl+C` 也只是一個
按鍵（ISIG 已關，`function.md` §2.1）。

help 的內容看 focus 在哪：`[1]`，以及 profile / saver 的 `[2]`，是鍵（core、global、`[1]` / `[2]` 各區塊的 item / panel operation、
navigate）；preference 的 `[2]` 是**只有** preference 六列（PIN 到 wrong_pin_attempt_cooldown）的說明，tmux / screen 的 `[2]` 是只有
conf、lock-after-time / idle、bind-key（tmux）、Install 的說明——這時 help 是這個面板的字典，取代原本每列下面的說明列（2026-09-25，使用者：focus 在 `[2]` 且項目是
preference 時只要 preference 的說明，只要）。每一項 key 一欄、說明一欄，說明比欄寬長就在欄內換行、key 只在第一行；鍵的清單也一樣換行。

---

## §B. 元素專職化

| 元素 | 唯一語意 |
|---|---|
| 邊框 Blue / 雙線 | focus；側欄的區塊標題也是 Blue（2026-09-24） |
| `[1]` Green `●` | 啟用中的 saver（只顯示，不設） |
| 值 Green | 使用者設了：PIN `set`、`on` |
| 值 Yellow | 還沒設：`not set`；畫布 `no PIN` |
| Red | 錯：PIN wrong、lockout、`· invalid`、`· taken`、`config error` |
| 值 Mauve | 可以改的值 |
| dim | 唯讀（type、色票）、menu 裡 disabled 的列、menu 的 region header |
| 邊框 title 的 ` · xxx` 尾綴 | 這個框現在的狀態（invalid / taken / wrong / try again in N s；`[2]` 的 ` · unsaved`） |
| 色票列的 `→` | 已存的顏色 → 草稿的顏色；沒草稿就沒有箭頭 |
| R / G / B 滑桿的顏色 | 那個通道在目前值的顏色（`#RR0000` / `#00GG00` / `#0000BB`） |

---

## §1 游標

只有一種 item 游標，兩個面板各一個、各自記位。

| | `[1]` | `[2]` |
|---|---|---|
| 停靠點 | saver 列、profile 列、`preference`；區塊標題跳過；開啟時停在啟用中的 profile | 可改的欄位；表頭列（Property / Value）、saver 列與色票列跳過；saver 的說明沒有停靠點；tmux / screen 的最後一個停靠點是底部的按鈕（2026-09-25） |
| `j` / `k` | 上下一列、頭尾相接（2026-09-24 修訂：原本不繞，使用者要面板跟 menu 一樣 loop） | 同 |
| `u` / `d` | 半頁，到頭就停不繞；清單比半頁短時等於跳到頭 / 尾 | 同 |
| `gg` / `G` | 頭 / 尾 | 同 |
| 換位時 | `[2]` 內容即時換成該項的明細、`[2]` 游標回第一個停靠點 | — |

鎖定畫布沒有游標。

---

## §2 輸入

### §2.1 每種欄位怎麼填（2026-09-24）

| 欄位 | 行為 |
|---|---|
| name（new / rename / duplicate） | 一行 input popup，邊框 `name`，預填目前值（new 預填 saver 的名字、用了就加號碼；duplicate 預填原名加 `2`）；Enter：空 → 邊框 ` · empty` 框留著、重複 → ` · taken` 框留著、否則寫檔；Esc 不動 |
| layout / size / font / time / date / runner / scene / profile | options popup，列出所有值、cursor 在目前值；`j`/`k`、Enter 選並寫檔、Esc 不動 |
| show_status | Enter 翻轉並寫檔，不開框 |
| pin_prompt_timeout / wrong_pin_attempts / wrong_pin_attempt_cooldown / lock-after-time（tmux）/ idle（screen） | 一行 input popup，邊框 `number`，預填目前值；清空 = 預設；非整數或負數 → ` · invalid` 框留著 |
| bind-key（tmux；2026-09-25） | 一行 input popup，邊框 `key`，預填目前值；Enter：清空 = 不綁、含空白或 `#` → ` · one key, e.g. l or C-l` 框留著、否則寫檔；裝著就直接進檔案與 server，沒裝只存 config |
| conf（tmux、screen 各一個；2026-09-25 前是 preference 的 tmux_conf / screen_conf） | 一行 input popup，邊框 `path`，webu 的作法（2026-09-24）：框裡 dim 顯示一個**提議**——目前值，沒有就是 `~/.tmux.conf` / `~/.screenrc`——`Tab` 把提議接進來編輯、`Backspace` 拒絕提議（空行 Enter = 清掉，按鈕就 disabled；裝著的話區塊先從舊檔拿掉）、打字就從頭打；Enter 照打的存，沒碰提議就 Enter 不改；不是絕對路徑或 `~/` 開頭 → ` · absolute or ~/ path` 框留著。footer 有提議時多 `Tab edit it · Bksp clear`；裝著時改路徑，區塊搬到新檔（2026-09-25） |
| R / G / B | options popup，0 到 255 一列一個數字、10 列一窗、cursor 在目前值置中；`j`/`k`/`u`/`d`/`gg`/`G`、Enter 移過去**進草稿**、不寫檔；色票列即時顯示草稿（webu slider 作法，不打字）。`S` 寫檔、`R` 丟草稿 |
| PIN | 遮罩 input popup 連開，見 §2.2 |

打字中屏蔽所有 hotkey：`Space` 是空白、`?` 是問號、`q` 是 q。Backspace 刪一字；沒有游標移動。

### §2.2 PIN 三連問

| 動作 | 順序 | 失敗 |
|---|---|---|
| Set PIN（未設） | `new PIN` → `confirm PIN` → 寫檔、toast `PIN set` | 兩次不同：toast `PIN mismatch`，回到空的 `new PIN`；長度不在 4 到 64：邊框 ` · 4-64 chars` 框留著 |
| Change PIN（已設） | `current PIN` → options popup `PIN`：`New PIN` / `Remove PIN` → `New PIN`：`new PIN` → `confirm PIN` → 寫檔；`Remove PIN`：Enter 立即寫檔（pin_hash 清空）、toast `PIN removed`，不再 confirm（2026-09-24） | current 錯：邊框 ` · wrong` 1 秒、清空、留在 current PIN |

每一步一個 popup、一次只問一件事；任一步 Esc 取消整串、什麼都不寫。遮罩顯示 `●`，不顯示長度以外的資訊；`●` 之間空一格、從框中央向兩側長，跟鎖定畫布的 prompt 同一個畫法（2026-09-24）。

### §2.3 鎖定畫布的 PIN prompt

| 事件 | 行為 |
|---|---|
| saver 上任何鍵 | 開 prompt；那個鍵**不算**輸入 |
| 可列印字元 | 追加，最多 64；`●` 遮罩 |
| Backspace | 刪一字 |
| Enter | 比對：對 → 立刻結束進程（exit 0，不等關閉動畫）；錯 → 邊框 ` · wrong` Red 1 秒、吞掉所有輸入、清空 |
| Esc | 回 saver、輸入丟掉 |
| 連錯 `wrong_pin_attempts` 次（0 = 關） | 邊框 ` · try again in N s` Red 倒數（`wrong_pin_attempt_cooldown` 秒）、吞掉所有輸入；Esc 仍可回 saver，再開 prompt 倒數繼續 |
| `pin_prompt_timeout` 秒沒按鍵（0 = 永不） | 關閉動畫回 saver、輸入丟掉；每次按鍵重算 |
| resize | prompt 重新置中 |

無 PIN 模式沒有 prompt：任何鍵直接結束進程。

---

## §3 導覽詞彙

`nav.go` 一份，所有清單共用；導覽字母 `j k u d g G` 不被任何動作佔用。

| 鍵 | 動作 |
|---|---|
| `j` / `k` | 上下一列；面板、menu、options 都繞（2026-09-24） |
| `u` / `d` | 半頁 / 半窗 |
| `gg` / `G` | 頭 / 尾 |
| `Tab`、`1` / `2` | 切面板 |

沒有 `h` / `l`：沒有橫向的東西。沒有 `/`：清單都很短。Mouse：不做。

---

## §4 Hotkey 分層與全表

規則：**小寫 = item operation 或移動**、**大寫 = panel operation 或全域**，例外是 `[1]` 的 `D` / `X`：紀錄類的
duplicate / delete 對齊 sshu 用大寫（修訂 2026-09-24）；bracket 印的就是要按的鍵。

| 層 | 鍵 |
|---|---|
| 全域 | `P`、`q`、`?`、`V`（彩蛋）、`Tab`、`1` / `2` |
| `[1]` item | `p` `D` `r` `X` |
| `[2]` item | `x`（只在 PIN 列） |
| `[2]` panel（saver 上） | `P` `S` `R` |
| 導覽 | `j` `k` `u` `d` `gg` `G` |

撞字檢查（2026-09-24 修訂）：`p` 只在 `[1]`、`P` 全域且在 saver 的 `[2]` 上就是那個 saver（同一件事，不撞），同 webu 的 `n` / `N`；`D` / `X` 只在 `[1]`，`x` 在 `[2]`
只有 PIN 列一處，語意都是「刪 / 清」；`r` 只在 `[1]`、`R` 只在 `[2]` saver 上；`S` 沒有小寫對手；`V` 與 `v`
不衝突（沒有 `v`）；`d` 是半頁不是 delete，同 webu。

---

## §5 浮層行為

沿用 u-family Popup Convention：一個 popup 一個檔一個 animator、`Esc` 只在 `closeTop` 一處解析、
`Space` 在非輸入浮層上 = 關掉它、正在關閉的浮層不握鍵盤。層數最多兩層（Space menu 上開 confirm）。

**先 confirm 的動作**：Delete saver、有未存顏色草稿時的 Quit、tmux / screen 的 Install 與 Uninstall（2026-09-25：寫的是別人的設定檔與跑著的 server）。Preview 不 confirm：任意鍵就回來，沒有代價（2026-09-25：preview 不驗 PIN，PIN 是 `locku lock` 的事）。Remove PIN 也不 confirm（2026-09-24）：它前面已經驗過 current PIN，那就是確認。

**toast**：`PIN set`、`PIN mismatch`、`cannot delete: active` / `cannot delete: last one`、
`nothing to save` / `nothing changed`、`write failed: <reason>`（值退回）。

**鎖定畫布**：PIN prompt 是唯一浮層，backdrop 是亮格降到 Surface2（`ui.md` §2.3）；`q` 在畫布與 prompt
裡都只是字元。

---

## §6 時間軸

### 設定畫面

| 情境 | 行為 |
|---|---|
| 啟動 | 讀 config；不存在或損毀 → 記憶體內用預設值，**不寫檔**，第一次改動才寫 |
| 每次改動 | 立即原子寫檔；失敗 toast、值退回。例外：顏色進草稿，`S` 才寫 |
| 外部同時改 config | 不監看；最後寫的贏 |
| Preview | 畫面被畫布取代（同進程，用記憶體內的 config 加顏色草稿；`P` 是啟用中的 saver、`p` 是游標那個）；解鎖或無 PIN 任意鍵 → 回設定畫面，焦點與兩個游標不變 |
| 離開 | `q` 直接離開；有未存的顏色草稿時先 confirm，Enter 丟掉草稿離開、Esc 留下 |

### 鎖定畫布

| 情境 | 行為 |
|---|---|
| 啟動 | 讀 config → 第一幀直接出現（不動畫）→ 依 saver 的 tick 排程 |
| tick | 只對有變的像素做 shuffle 揭露，≤ 400 ms；prompt 開著時照常 tick、照常揭露，只是亮格退成 backdrop 色（修訂 2026-09-24：原本停 tick、關掉後補畫） |
| 任何鍵 | 開 prompt（§2.3） |
| 解鎖 | Enter 比對成功 → exit 0，tmux / screen 自己重繪 |
| tty 消失 | read 得到 EOF / EIO → exit 0；SIGHUP 本身忽略 |
| resize | 重算 k 整張重畫；prompt 重新置中 |
| 無 PIN 模式 | 狀態列 Yellow 提示；任何鍵 exit 0 |

---

## 附錄 — hotkey 全表

### Core key
`Tab` 切面板 · `Enter` 進 `[2]` / 編輯 / 送出 · `Esc` 關浮層 / 回 saver · `Space` menu · `?` help

### 全域
`P` preview · `q` quit · `1` / `2` 直達面板

### `[1]` 側欄
saver 上 `n` new profile · `p` preview the defaults · profile 上 `Enter` edit · `p` preview this profile · `D` duplicate · `r` rename · `X` delete

### `[2]` 明細
`Enter` rename / choose / toggle / pick / set PIN / change PIN（含 remove） · saver 上 `n` new profile · saver / profile 上 `P` preview · `S` save colours · `R` reset colours

### 鎖定畫布
任何鍵 開 prompt · prompt 內 `Enter` 送出 · `Esc` 回 saver · `Backspace` 刪一字

### 導覽
`j/k` · `u/d` · `gg/G`
