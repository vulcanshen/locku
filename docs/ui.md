# locku — UI

> 本文件講**版面與 surface**：兩個畫面、popup、色帶、chrome、存檔。按鍵語意與流程在 `ux.md`，
> 功能邊界在 `function.md`。依 VTP（`thoughts/tui-design`）與 u-family Popup Convention 撰寫，
> 每條版面決定標日期。v1.0 定案，2026-09-24。

---

## §1 兩個畫面

locku 有兩個彼此獨立的畫面，由 CLI 決定進哪一個，執行期間不互切（Preview 例外，見 §2.1）：

| 畫面 | 指令 | 性質 |
|---|---|---|
| 設定 | `locku` | 兩個面板的 u-family 版面，吃完整 VTP |
| 鎖定 | `locku lock`、argv[0] = SCREEN-LOCK | 全螢幕畫布，零 chrome，VTP 只作用在它的 popup |

### 1.1 設定畫面 grid

```
╔ [1] locku ═════════════╗╭ [2] clock · unsaved ─────────────────────────────╮
║ Profiles               ║│ name              clock                          │
║ ● clock                ║│ saver             clock                          │
║   clock2               ║│ layout            row                            │
║   dino                 ║│ size              medium                         │
║ Savers                 ║│ font              3x7                            │
║   clock                ║│ time              HH MM                          │
║   dino                 ║│ date              off                            │
║ Settings               ║│ bg                ■ #313244  →  ■ #ff3244        │
║   preference           ║│   R               ───────────● 255               │
║                        ║│   G               ──●───────── 50                │
║                        ║│   B               ───●──────── 68                │
║                        ║│ fg                ■ #f2b753                      │
║                        ║│   R               ──────────●─ 242               │
║                        ║│   G               ────────●─── 183               │
║                        ║│   B               ────●─────── 83                │
╚════════════════════════╝╰──────────────── ~/.config/locku/config.yaml ─────╯
 space menu   ? help   tab/1-2 panels   q quit                                  ← footer
```

左 `[1]` 側欄三個區塊，順序 Profiles → Savers → Settings（2026-09-24 定案，使用者以 OOP 分：saver 是 class、profile 是
object，常用的 profile 在上）：**Profiles** 列出使用者設定好的、有名字的 saver 實例，新增（從 Savers 的一種按 `n`）、
複製、改名、刪除都在這裡；**Savers** 列出有哪幾種 saver（clock、dino），它們沒有名字、名字就是自己，不能新增刪除；
**Settings** 一項：`preference`。區塊標題 Blue、是
分隔，不可停，區塊之間不空列；cursor 只在項目之間走，開啟時停在啟用中的 profile。啟用中的 profile 前面一顆 Green `●`，
是側欄唯一的綠色；它只顯示，設為啟用在 `preference › profile`。

右 `[2]` 是**明細**，內容跟著 `[1]` 的 cursor 即時切換，不用 Enter，明細沒有切換成本：cursor 在 saver 上是它的說明
（唯讀）；在 profile 上就是那個 profile 的欄位與顏色；在 `preference` 上是一般設定的列。title chip 跟著換成
`[2] clock · saver`、`[2] clock`、`[2] preference`；profile 的顏色草稿未存時尾綴 ` · unsaved`。

`[2]` 在 saver 上（全部唯讀、沒有停靠點，唯一的動作是 `[n] New`）：

| 列 | 值 |
|---|---|
| saver | `clock` / `dino` |
| what | 一句話：clock 是 the time and the date, on the LED board；dino 是 the offline dino run, jumping by itself, for ever |
| settings | 這種 saver 的 profile 能設什麼 |
| profiles | 是它的 profile 名，逗號分隔；沒有就 `none yet` |

`[2]` 在 profile 上：

| 列 | 值 | Enter |
|---|---|---|
| name | 實例名 | input popup，型別 `name`；重複或空被擋 |
| saver | `clock` / `dino` | 唯讀，dim，不可停：profile 的 class，要換就從 Savers 新增一個 profile（2026-09-24 定案；當天曾短暫可改）；底下的列跟著 saver 換 |
| layout | `row` / `column` | options popup，cursor 在目前值（clock） |
| size | `small` / `medium` / `large`（一個字型像素 1 / 2 / 3 格見方） | options popup，cursor 在目前值（clock；dino 沒有 size，畫布自己取最大） |
| font | `3x7` / `3x5`（字型高 7 列或 5 列） | options popup，cursor 在目前值 |
| time | `HH MM` / `HH MM SS`（2026-09-24：拿掉 12 時制，時間不畫冒號） | options popup，cursor 在目前值 |
| date | `off` / `YYYY-MM-DD` / `YYYY-MMM-DD` / `MM-DD` / `MMM-DD` | options popup，cursor 在目前值；不是 off 時畫布第二列（clock） |
| runner | `trex`（dino；目前唯一） | options popup（2026-09-24） |
| scene | `grassland`（dino；目前唯一） | options popup（2026-09-24） |
| bg / fg | 一格該色的 glyph 當色票 + hex，是**已存**的顏色；草稿不同時右邊接 `→` 加草稿的色票 + hex | 不可停 |
| R / G / B | webu 的 slider 列：12 格軌道 + 草稿的值，軌道用**該通道自己的顏色**畫——R 列是 `#RR0000`、G 列 `#00GG00`、B 列 `#0000BB`，值多大顏色就多亮；軌道底色反向，0 時全白、255 時全黑，暗的值才看得見；數字是 Mauve、沒有底色，跟其他列的值一樣（2026-09-24） | options popup：0 到 255 的數字清單，10 列一窗、游標在目前值置中，Enter 移過去（webu slider 作法，不打字）— 改的是草稿 |

顏色走**草稿**（修訂 2026-09-24，使用者調歪過一次調不回來）：滑桿改草稿，色票列同時看得到已存與草稿，
panel operation `[P] Preview` 預覽這個 saver、`[S] Save` 寫檔、`[R] Reset` 丟掉草稿；草稿跟著 saver 的名字走
（rename 帶走、delete 一起丟）。預覽都帶著草稿。其餘欄位仍立即寫檔。手改 config 的非法 hex 視同預設。
label 欄固定 18 欄，只在面板窄到放不下 label 加值時才縮（修訂 2026-09-24：原本上限是面板寬的三分之一，
一般寬度的終端機就把 `lockout_seconds` 壓到貼著值）。

`[2]` 在 `preference` 上：

| 列 | 值的呈現 | Enter |
|---|---|---|
| PIN | `set`（Green）/ `not set`（Yellow） | 未設：設定流程；已設：current PIN 之後選 `New PIN` / `Remove PIN`（2026-09-24，取消併進同一條流程） |
| profile | 啟用中的 profile 名；指向不存在的加 ` (missing)` Yellow | options popup 列出所有 profile、cursor 在目前值，Enter 寫檔、側欄 `●` 移過去 |
| show_status | `on` / `off` | 原地翻轉，不開 popup |
| prompt_timeout | 數字 | input popup，型別 `number` |
| lockout_after | 數字，0 顯示 `0 (off)` | 同上 |
| lockout_seconds | 數字 | 同上 |
| tmux_conf | 路徑照存的樣子；未設 `not set`（Yellow），`locku setup tmux` 會報錯 | input popup，型別 `path`，webu 的提議作法（`ux.md` §2.1）（2026-09-24） |
| screen_conf | 同上 | 同上 |

`savers` 這個 key 不成列：它就是 `[1]` 本身。其餘每個 config key 一定有一列。
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
| config 錯誤 | 同上 | `· config error: <reason>`，Red，取代無 PIN 提示 |
| 區塊 | 時間先、日期後 | 兩個獨立區塊：row 日期在時間下面，column 日期一欄在左、時間一欄在右；各自先降 size 再去單位，日期塞不下就不畫，`function.md` §5.3 |

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
- **Preview** 有兩個入口：全域 `P` 看啟用中的 saver（`?` 揭露）；側欄 saver 上的 `[p] Preview` 看游標那一個，
  不改啟用（修訂 2026-09-24）。整合設定不在 TUI 裡，是 `locku setup`。

**Preview**：一個動作，整個 TUI 被鎖定畫布取代，就像桌面螢幕保護程式的預覽。同一個進程、用記憶體內的 config
加上顏色草稿，解鎖後回到設定畫面、焦點與 cursor 不變。無 PIN 時任意鍵就回來。


### 2.2 `[2]` 明細

單一職責：顯示並編輯 `[1]` 目前指到的那一項。不做新增、刪除，那是 `[1]` 的事。

### 2.3 鎖定畫布

單一職責：畫內容與狀態列。所有按鍵（含 Ctrl 組合，`function.md` §2.1）只做一件事：開 PIN popup；
無 PIN 模式則結束進程。PIN popup 開著時亮格改畫 Surface2、暗格不變，當 backdrop；saver 照常 tick、揭露照常動（修訂 2026-09-24：原本停 tick、一次重畫不再動，使用者要的是背景變色但不停）。

---

## §3 Popup

全部走 u-family Popup Convention：一個 popup 一個檔一個 animator、title = glyph + 文字、hint 嵌
下邊框、`Esc` 只在 `closeTop` 一處解析、動畫 8 × 16 ms。層數最多兩層。

### 3.1 設定畫面

| Popup | 類型 | 用途 |
|---|---|---|
| Space menu | menu | `[1]` saver / profile 的 item region；`[2]` 欄位的 item region；`[2]` 在 profile 上另有 panel region（Preview / Save / Reset）— 兩個 region 各有 header，只有一個就扁平；saver 的 `[2]` 只有 `[n] New` |
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

╭ PIN · closing ────────────────╮     prompt_timeout 到：正常關閉動畫回 saver
```

錯誤與 lockout 的 Red 是 override 色（VTP §2.4），不參與層級。

---

## §4 色帶

錨點 catppuccin-mocha，與家族相同。點陣板的兩色是**使用者資料**（每個 saver 的 bg / fg），不屬於 app 色帶，預設值取 splash。

| 色帶 | 意思 | 值 |
|---|---|---|
| Blue | focus：焦點面板邊框；側欄的區塊標題（修訂 2026-09-24） | `#89b4fa` |
| Surface2 | unfocused 面板邊框；PIN prompt 開啟時亮格的 backdrop 色 | `#585b70` |
| Green | 使用者足跡：啟用中的 saver `●`、PIN `set`、toggle `on` | `#a6e3a1` |
| Mauve | 可填的：`[2]` 的值 | `#cba6f7` |
| saver 的 fg | 點陣板亮格，使用者可改 | 預設 gold `#f2b753` |
| saver 的 bg | 點陣板暗格，使用者可改 | 預設 surface0 `#313244` |
| Overlay0 | 狀態列、hint、唯讀的 type 列與色票列 | `#6c7086` |
| Yellow（warn） | `not set`、`no PIN · any key unlocks` | override |
| Red（error） | PIN wrong、lockout、`· invalid`、`· taken`、`config error` | override |
| popup layer scale | 浮層邊框，最多兩層 | VTP §2.5 |

focus 二態同 kbu §8.4：雙線 `╔═╗` + Blue ↔ 圓角細線 `╭─╮` + Surface2，零位移。畫布沒有焦點概念，
Blue 不出現在那裡。

---

## §5 Chrome

| 件 | 設定畫面 | 鎖定畫布 |
|---|---|---|
| Border title chip | `[1] locku`、`[2] <saver> · saver`、`[2] <profile name>`（顏色草稿未存時 ` · unsaved`）/ `[2] preference` | 無 |
| Panel tab bar | 無 | 無 |
| Border hint | `[2]` 下框右側：config 路徑 | 無 |
| footer | `space menu   ? help   tab/1-2 panels   q quit` | 無 |

**Nerd Font 是設計、必裝**，與家族相同：畫布像素就是 nf-fa-square。字型在使用者本機的終端機模擬器，
經 SSH 不受影響。`docs/icon.svg` 沿用 u-family mark 的 locku 版；`V` splash 彩蛋家族同鍵，只在設定畫面。

---

## §6 存檔

| 資料 | 形式 | 位置 |
|---|---|---|
| 設定 | `config.yaml`，見 `function.md` §7 | `~/.config/locku`（`$XDG_CONFIG_HOME/locku`、`$LOCKU_CONFIG`） |

沒有 data、沒有 cache、沒有 log。寫檔原子（temp + rename），權限 600；目錄不存在時建立。

---

## §7 待決

無。2026-09-24 全部定案。
