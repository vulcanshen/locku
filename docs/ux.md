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
`u`/`d` 半窗、`gg`/`G` 首尾、Enter 執行、letter hotkey 在 menu 裡也有效。唯一有 panel operation 的是 `[2]` 在
saver 上：顏色草稿的 Save / Reset（修訂 2026-09-24）。整合設定是 CLI `locku setup`。

**`[1]` 側欄**

| cursor 在 | item operation |
|---|---|
| saver | `[Enter] Edit`（焦點送到 `[2]`）、`[p] Preview`（鎖定畫布顯示這個 saver，不改啟用）、`[D]uplicate`（name popup，提議原名加 `2`）、`[r]ename`（name popup）、`[X] Delete`（confirm；最後一個或啟用中 disabled 並說明） |
| preference | `[Enter] Edit`（焦點送到 `[2]`） |

**`[2]` 明細**

| cursor 在 | item operation | panel operation |
|---|---|---|
| name | `[Enter] Rename` | `[P] Preview`（這個 saver，帶草稿）、`[S] Save`、`[R] Reset`（顏色草稿；沒草稿時 disabled 並說明） |
| type | 唯讀，不可停 | 同上 |
| layout / size / time / date | `[Enter] Choose` | 同上 |
| bg / fg 色票列 | 唯讀，不可停 | 同上 |
| R / G / B | `[Enter] Pick`（進草稿） | 同上 |
| PIN | 未設：`[Enter] Set PIN`；已設：`[Enter] Change PIN`、`[x] Clear PIN`（current PIN → confirm） | 無 |
| saver | `[Enter] Choose`（列出所有 saver） | 無 |
| show_status | `[Enter] Toggle` | 無 |
| prompt_timeout / lockout_after / lockout_seconds | `[Enter] Edit` | 無 |

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

---

## §1 游標

只有一種 item 游標，兩個面板各一個、各自記位。

| | `[1]` | `[2]` |
|---|---|---|
| 停靠點 | saver 列、`preference`；區塊標題跳過 | 可改的欄位；type 與色票列跳過 |
| `j` / `k` | 上下一列、不繞 | 同 |
| `u` / `d` | 半頁 | 同 |
| `gg` / `G` | 頭 / 尾 | 同 |
| 換位時 | `[2]` 內容即時換成該項的明細、`[2]` 游標回第一個停靠點 | — |

鎖定畫布沒有游標。

---

## §2 輸入

### §2.1 每種欄位怎麼填（2026-09-24）

| 欄位 | 行為 |
|---|---|
| name（rename / duplicate） | 一行 input popup，邊框 `name`，預填目前值（duplicate 預填原名加 `2`）；Enter：空 → 邊框 ` · empty` 框留著、重複 → ` · taken` 框留著、否則寫檔；Esc 不動 |
| layout / size / time / date / saver | options popup，列出所有值、cursor 在目前值；`j`/`k`、Enter 選並寫檔、Esc 不動 |
| show_status | Enter 翻轉並寫檔，不開框 |
| prompt_timeout / lockout_after / lockout_seconds | 一行 input popup，邊框 `number`，預填目前值；清空 = 預設；非整數或負數 → ` · invalid` 框留著 |
| R / G / B | options popup，0 到 255 一列一個數字、10 列一窗、cursor 在目前值置中；`j`/`k`/`u`/`d`/`gg`/`G`、Enter 移過去**進草稿**、不寫檔；色票列即時顯示草稿（webu slider 作法，不打字）。`S` 寫檔、`R` 丟草稿 |
| PIN | 遮罩 input popup 連開，見 §2.2 |

打字中屏蔽所有 hotkey：`Space` 是空白、`?` 是問號、`q` 是 q。Backspace 刪一字；沒有游標移動。

### §2.2 PIN 三連問

| 動作 | 順序 | 失敗 |
|---|---|---|
| Set PIN（未設） | `new PIN` → `confirm PIN` → 寫檔、toast `PIN set` | 兩次不同：toast `PIN mismatch`，回到空的 `new PIN`；長度不在 4 到 64：邊框 ` · 4-64 chars` 框留著 |
| Change PIN（已設） | `current PIN` → `new PIN` → `confirm PIN` → 寫檔 | current 錯：邊框 ` · wrong` 1 秒、清空、留在 current PIN |
| Clear PIN | `current PIN` → confirm `Clear PIN?` → 寫檔（pin_hash 清空） | 同上 |

每一步一個 popup、一次只問一件事；任一步 Esc 取消整串、什麼都不寫。遮罩顯示 `●`，不顯示長度以外的資訊。

### §2.3 鎖定畫布的 PIN prompt

| 事件 | 行為 |
|---|---|
| saver 上任何鍵 | 開 prompt；那個鍵**不算**輸入 |
| 可列印字元 | 追加，最多 64；`●` 遮罩 |
| Backspace | 刪一字 |
| Enter | 比對：對 → 立刻結束進程（exit 0，不等關閉動畫）；錯 → 邊框 ` · wrong` Red 1 秒、吞掉所有輸入、清空 |
| Esc | 回 saver、輸入丟掉 |
| 連錯 `lockout_after` 次（0 = 關） | 邊框 ` · try again in N s` Red 倒數、吞掉所有輸入；Esc 仍可回 saver，再開 prompt 倒數繼續 |
| `prompt_timeout` 秒沒按鍵（0 = 永不） | 關閉動畫回 saver、輸入丟掉；每次按鍵重算 |
| resize | prompt 重新置中 |

無 PIN 模式沒有 prompt：任何鍵直接結束進程。

---

## §3 導覽詞彙

`nav.go` 一份，所有清單共用；導覽字母 `j k u d g G` 不被任何動作佔用。

| 鍵 | 動作 |
|---|---|
| `j` / `k` | 上下一列；面板不繞、menu 與 options 繞 |
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

**先 confirm 的動作**：Delete saver、Clear PIN、有未存顏色草稿時的 Quit。Preview 不 confirm：解鎖就回來，沒有代價。

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
| tick | 只對有變的像素做 shuffle 揭露，≤ 400 ms；prompt 開著時停 tick，關掉後補畫到當下時間 |
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
`Enter` edit · `p` preview this saver · `D` duplicate · `r` rename · `X` delete

### `[2]` 明細
`Enter` rename / choose / toggle / pick / set PIN / change PIN · `x` clear PIN · saver 上 `P` preview this saver · `S` save colours · `R` reset colours

### 鎖定畫布
任何鍵 開 prompt · prompt 內 `Enter` 送出 · `Esc` 回 saver · `Backspace` 刪一字

### 導覽
`j/k` · `u/d` · `gg/G`
