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
╔ [1] locku ═════════════╗╭ [2] style ───────────────────────────────────────╮
║ Savers                 ║│ bg    ■■ #313244                                 │
║ ● clock                ║│   R   ──●───────── 49                            │
║   clock2               ║│   G   ──●───────── 50                            │
║                        ║│   B   ───●──────── 68                            │
║                        ║│ fg    ■■ #f2b753                                 │
║ Settings               ║│   R   ──────────●─ 242                           │
║   config               ║│   G   ────────●─── 183                           │
║   style                ║│   B   ────●─────── 83                            │
╚════════════════════════╝╰──────────────── ~/.config/locku/config.yaml ─────╯
 space menu   ? help   tab/1-2 panels   q quit                                  ← footer
```

左 `[1]` 側欄兩個區塊：**Savers** 列出所有 saver 實例，**Settings** 兩項：`config` 與 `style`。區塊標題是
分隔，不可停；cursor 只在項目之間走。啟用中的 saver 前面一顆 Green `●`，是側欄唯一的綠色。

右 `[2]` 是**明細**，內容跟著 `[1]` 的 cursor 即時切換，不用 Enter，明細沒有切換成本：cursor 在 saver 上就是那個 saver
的欄位；在 `config` 上是一般設定的列，在 `style` 上是兩個顏色。title chip 跟著換成 `[2] clock`、`[2] config`、`[2] style`。

`[2]` 在 saver 上：

| 列 | 值 | Enter |
|---|---|---|
| name | 實例名 | input popup，型別 `name`；重複或空被擋 |
| type | `clock` | 唯讀，dim；v1 只有這一個 type |
| time | `HH:MM` / `HH:MM AM/PM` / `HH:MM:SS` / `HH:MM:SS AM/PM` | options popup，cursor 在目前值 |
| date | `off` / `YYYY-MM-DD` / `YYYY-MMM-DD` / `MM-DD` / `MMM-DD` | options popup，cursor 在目前值；不是 off 時畫布第二列 |

`[2]` 在 `config` 上：

| 列 | 值的呈現 | Enter |
|---|---|---|
| PIN | `set`（Green）/ `not set`（Yellow） | 未設：設定流程；已設：更改流程。清除走 Space menu |
| show_status | `on` / `off` | 原地翻轉，不開 popup |
| prompt_timeout | 數字 | input popup，型別 `number` |
| lockout_after | 數字，0 顯示 `0 (off)` | 同上 |
| lockout_seconds | 數字 | 同上 |

`[2]` 在 `style` 上，兩組各四列：

| 列 | 值的呈現 | Enter |
|---|---|---|
| bg / fg | 兩格該色的 glyph 當色票 + hex，唯讀，隨下面三列即時變 | 不可停 |
| R / G / B | webu 的 slider 列：12 格軌道 + 目前值 | options popup：0 到 255 的數字清單，10 列一窗、游標在目前值置中，Enter 移過去（webu slider 作法，不打字） |

改一個 channel 就立即寫檔，config 存 hex。手改 config 的非法 hex 視同預設。

`saver` 與 `savers` 兩個 key 不成列：它們就是 `[1]` 本身。其餘每個 config key 一定有一列。
每次改完立即寫檔，沒有 Save 鍵，沒有 dirty 狀態。寫檔失敗以 toast 報錯，值退回。

### 1.2 鎖定畫布 grid

```
□ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □
□ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □
□ □ □ □ □ ■ □ □ ■ ■ ■ □ □ □ □ ■ ■ ■ □ □ ■ □ ■ □ □ □ □ □ □ □ □ □   ← 亮格 style.fg，預設 gold
□ □ □ □ ■ ■ □ □ □ □ ■ □ ■ □ □ □ □ ■ □ □ ■ □ ■ □ □ □ □ □ □ □ □ □     暗格 style.bg，預設 surface0
□ □ □ □ □ ■ □ □ ■ ■ ■ □ □ □ □ ■ ■ ■ □ □ ■ ■ ■ □ □ □ □ □ □ □ □ □     每格 = nf-fa-square + 空格
□ □ □ □ □ ■ □ □ ■ □ □ □ ■ □ □ □ □ ■ □ □ □ □ ■ □ □ □ □ □ □ □ □ □
□ □ □ □ □ ■ □ □ ■ ■ ■ □ □ □ □ ■ ■ ■ □ □ □ □ ■ □ □ □ □ □ □ □ □ □
□ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □
□ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □ □
 vulcan@prod-db-01 · locked since 13:58                           ← 狀態列，一般文字
```

整個終端機是一塊 LED 點陣板：狀態列以外的每一格都是 nf-fa-square 加空格，沒亮的用 style.bg，亮的用
style.fg，字由亮格組出來。沒有邊框、沒有 title chip、沒有 footer。footer 在這裡沒有意義：唯一的動作是
「按任何鍵」，不需要揭露。**只有一種畫法**：saver 給幾行 ASCII，渲染器（`function.md` §5.3）選最大
的整數倍 k 讓它塞進格數，置中。splash 底下的名字、版本、開發者都不出現，只取它的 glyph 畫法。

| 元素 | 位置 | 規則 |
|---|---|---|
| 點陣板 | 狀態列以外全部 | 每格一個 glyph 加空格；終端機寬為奇數時最右一欄留白 |
| 暗格 | 沒亮的格 | style.bg，預設 surface0 |
| 內容 | 點陣板置中 | 亮格 style.fg，預設 gold；k < 1 時點陣板照鋪，內容改用一般文字以 fg 色置中疊在板上 |
| 狀態列 | 最後一列，左起 1 欄 | `user@host · locked since HH:MM`；`show_status: false` 時整列空白 |
| 無 PIN 提示 | 狀態列右側接續 | `· no PIN · any key unlocks`，Yellow，不受 show_status 影響 |
| config 錯誤 | 同上 | `· config error: <reason>`，Red，取代無 PIN 提示 |
| 退階 | | 塞不下時去年 → 去秒 → 去日期 → 一般文字，`function.md` §5.3 |

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

- Savers：Enter = 設為啟用（`●` 移過去、config `saver` 立即寫檔），與 webu `[1]` Tabs 的 Enter
  切換同一個語意。duplicate / rename / delete 在 Space menu 的 item region。
- Settings › config：Enter 只是把焦點送到 `[2]`。
- **Preview**（全螢幕試鎖）是全域動作，走 `?` 與大寫鍵 `P`，不佔列。理由：它作用於整個 app 而非 `[1]` 的
  某一項，VTP 把這類動作放 non-contextual track（2026-09-24）。整合設定不在 TUI 裡，是 `locku setup`。

**Preview**：一個動作，整個 TUI 被鎖定畫布取代，就像桌面螢幕保護程式的預覽。同一個進程、用目前 config
（全部已落盤），解鎖後回到設定畫面、焦點與 cursor 不變。無 PIN 時任意鍵就回來。


### 2.2 `[2]` 明細

單一職責：顯示並編輯 `[1]` 目前指到的那一項。不做新增、刪除，那是 `[1]` 的事。

### 2.3 鎖定畫布

單一職責：畫內容與狀態列。所有按鍵（含 Ctrl 組合，`function.md` §2.1）只做一件事：開 PIN popup；
無 PIN 模式則結束進程。PIN popup 開著時 saver 停 tick，亮格改畫 Surface2、暗格不變，當 backdrop，一次重畫、不再動。

---

## §3 Popup

全部走 u-family Popup Convention：一個 popup 一個檔一個 animator、title = glyph + 文字、hint 嵌
下邊框、`Esc` 只在 `closeTop` 一處解析、動畫 8 × 16 ms。層數最多兩層。

### 3.1 設定畫面

| Popup | 類型 | 用途 |
|---|---|---|
| Space menu | menu | `[1]` saver 的 item region；`[2]` 欄位的 item region；只有一個 region 就扁平 |
| `?` help | viewport | 全域動作表 |
| input | input | **邊框寫型別**（`name`、`number`、`number · invalid`、`name · taken`），框內一行是欄位名，目前值當提議；清空 = 預設值 |
| PIN input | input，遮罩 | 邊框 `current PIN`、`new PIN`、`confirm PIN`；輸入顯示 `●` |
| confirm | message | Delete saver、Clear PIN |
| toast | message | 寫檔失敗、PIN 不一致、不可刪（啟用中 / 最後一個） |

duplicate 是 `name` input popup：提議值是原名加 `2`，確認後複製參數並把 cursor 移到新實例。
PIN 設定與更改是**同一種 popup 連續開**（`current PIN` → `new PIN` → `confirm PIN`），一次只問一件事，
錯在哪一步就停在哪一步。

### 3.2 鎖定畫布

| Popup | 類型 | 用途 |
|---|---|---|
| PIN prompt | input，遮罩，寬固定 32 欄置中 | 唯一的 popup |

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

錨點 catppuccin-mocha，與家族相同。點陣板的兩色是**使用者資料**（Settings › style），不屬於 app 色帶，預設值取 splash。

| 色帶 | 意思 | 值 |
|---|---|---|
| Blue | focus：焦點面板邊框 | `#89b4fa` |
| Surface2 | unfocused 面板邊框；PIN prompt 開啟時亮格的 backdrop 色 | `#585b70` |
| Green | 使用者足跡：啟用中的 saver `●`、PIN `set`、toggle `on` | `#a6e3a1` |
| Mauve | 可填的：`[2]` 的值 | `#cba6f7` |
| style.fg | 點陣板亮格，使用者可改 | 預設 gold `#f2b753` |
| style.bg | 點陣板暗格，使用者可改 | 預設 surface0 `#313244` |
| Overlay0 | 狀態列、hint、唯讀的 type 列、區塊標題 | `#6c7086` |
| Yellow（warn） | `not set`、`no PIN · any key unlocks` | override |
| Red（error） | PIN wrong、lockout、`· invalid`、`· taken`、`config error` | override |
| popup layer scale | 浮層邊框，最多兩層 | VTP §2.5 |

focus 二態同 kbu §8.4：雙線 `╔═╗` + Blue ↔ 圓角細線 `╭─╮` + Surface2，零位移。畫布沒有焦點概念，
Blue 不出現在那裡。

---

## §5 Chrome

| 件 | 設定畫面 | 鎖定畫布 |
|---|---|---|
| Border title chip | `[1] locku`、`[2] <saver name>` / `[2] config` / `[2] style` | 無 |
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
