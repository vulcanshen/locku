# locku — terminu fix

locku 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.20/principle)（tdp v0.1.20）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `docs/` 裡描述該行為的段落。有意不修的，改寫成 `dev-remarks.md`「偏離 tdp」的一條並附理由。

> **v0.1.20（2026-09-29）**：K11 / D3 —— 模式名夾在兩個框線接頭之間（雙線 `╡Drag╞`、單線 `┤Visual├`），模式色加粗、盡量一個詞，
> 放不下先截標題，panel 膠囊跟著外框換色；D6 —— icon 寬度量的是游標實際前進幾格，參考實作的完整清單、`<APP>_ICON_WIDTH`
> 覆寫、只在 unix 探測、做完的驗收。filu 的參考實作已完成，icon 寬度那一條現在可以做。本清單的每一條已照 v0.1.20 重新核對過。

> **v0.1.19（2026-09-29，這份清單寫完後才出）**：L5 —— focus 不能只靠顏色分辨（模式會換框色），家族預設雙線；K10 ——
> 子程序還沒準備好時可以不轉送一般的鍵，但 `Ctrl-C` 照樣轉送。本清單的每一條已照 v0.1.19 重新核對過。

盤點日期：2026-09-29（對照 tdp v0.1.18）。以 `main` 的 `8f50686` 為準（已對齊 v0.1.17，工作區乾淨）。這一輪**只對 v0.1.17 → v0.1.18
的改動**（`git -C terminu diff v0.1.17 v0.1.18 -- principle/` 與 terminu CHANGELOG 的 v0.1.18 段）：K11 + D2 模式名與 Yellow 外框、F1 + D3
finder 的 focus、D2 失焦 panel 邊框上的 hint、D3 下框 hint 放不下從尾端整組丟、D6 icon 的實際寬度、K10 / K9 的補充。每一條照內容對程式碼
核對過，位置寫檔案與函式，不寫行號；hint 的截斷是在 scratch 複本裡 render 量的（沒有動 locku 的工作樹）。

要修的兩條：下框 hint 截在項目中間（第 1 條，D3）、icon 的實際寬度（第 2 條，D6，**等 filu 做完再做**）。模式、finder、失焦 panel 的 hint、
PTY 都不適用，見最後。

> **2026-09-29**：第 1 條修好、從清單刪掉（locku `drawPopupBox()` 收 pairs，`hintLegend(pairs, w)` 從尾端整組丟）；剩第 2 條，等 filu。

指向 tdp 的五處連結（兩份 README、`docs/ux.md`、`docs/ui.md`、`docs/dev-remarks.md` 開頭）已經改釘 `tree/v0.1.20/principle`，在工作樹裡、
未 commit。


## 先看

- **清單先 commit，再動程式。** 這份清單與改釘的五處連結一起先 commit；程式的 commit 跟清單分開，一條一個 commit，只加自己改的路徑。
- **第 2 條（icon 寬度）等 filu。** filu 要先補完自己的內容列，參考實作（`internal/ui/width.go`、`iconwidth_unix.go`）定下來之後 locku 再照搬；
  在那之前只做第 1 條，第 2 條留在清單上。
- **每修一處補 model test，逐處 mutation**：把修正單獨改回舊行為，確認對應的測試會紅。
- **同一個 commit 同步 README 兩份與 `docs/`（`ux.md`、`ui.md`、`dev-remarks.md`），CHANGELOG 記 `[Unreleased]`**；已經有的條目就併進去。
- **修完拿 v0.1.20 全文再逐條對一次**，不只這兩條與 CHANGELOG。
- **不 push、不發版。**
- **把這一輪寫進 terminu repo 的 `.local/family-fix/locku/README.md`**（本機、不進版控）：開頭的輪次清單加「對照 v0.1.18」一行，「修了什麼」、
  「已經符合」各加一節，「給下一個 app 的經驗」與「發布」補上這一輪。


## 2. icon 的實際寬度 —— D6（filu 已完成，照搬）

**filu 的參考實作已完成**（2026-09-29，`e1de220`，filu 第六輪）。照搬的東西（v0.1.20 的 D6 有同一份清單，細節在 terminu
`.local/family-fix/filu/README.md`「第六輪」最後的「D6 照搬清單」）：

- filu `internal/ui/width.go` 整個檔：`iconCells` / `IconCells()`、`isWideIcon()`、`iconCount()`、`dispWidth()`、`dispClip()`、
  `padDisp()`、`padDispRight()`、`truncate()`、`dispCutLeft()`、`compositeDisp()`（跟 `overlay.Composite` 同介面，直接換掉呼叫）、
  `centerDisp()`（取代 `lipgloss.Place`）、`blockWidth()`、`joinH()` / `joinV()`（取代 lipgloss 的 Join）。
- `iconwidth_unix.go` 的 `DetectIconWidth()`，在 `tea.NewProgram` 之前呼叫；手動覆寫用 `<APP>_ICON_WIDTH`（filu 是
  `FILU_ICON_WIDTH`）。探測只在 unix 做，Windows 預設一格、靠環境變數覆寫。
- 測試照 `d6_test.go`：icon 1 / 2 格下每一種 popup 各開一次，量**單獨的框**（並排的框量單一個）與**疊上去的整個畫面**每一列；
  `compositeDisp()` 的四種邊界（popup 列有 icon、被蓋的列有 icon、icon 被左 / 右框邊切半）。
- 驗收：`grep -n 'lipgloss.Width\|lipgloss.Size\|lipgloss.Place\|ansi.StringWidth\|ansi.Truncate' internal/ui/*.go` 只剩寬度函式本身。
- filu 的提醒：寬度改走 `dispWidth()` 後，在 `iconCells = 1` 的終端機上畫面完全不變（既有測試原封不動通過），只有探測到 2 才作用。


**現況**：locku 量寬度只有一套、而且看不到 icon 的實際寬度：`internal/ui/width.go` 的 `dispW()`（= `lipgloss.Width`）、`truncate()`、
`truncateHead()`、`clipANSI()`（`ansi.Truncate`）、`padRight()`、`padLeft()`、`fitLines()`、`wrap()`，都把 icon 算成一格。

用到的 icon（`theme.go`，都在 PUA）：

| icon | 用在哪 |
|---|---|
| `pixelGlyph`（U+F0C8，nf-fa-square） | 點陣板的每一個像素（clock、dino、`EXIT` / `NONE` 的字板、預覽）、splash、`[2]` 的色票 |
| `glyphLock`、`glyphMenu` / `glyphList`、`glyphHelp`、`glyphWarn`、`glyphInfo`、`glyphInput` | 每個 popup 的上框標題（含 toast、鎖定畫面的 PIN prompt、custom saver 上的 PIN 框） |
| powerline 的 `capLeft`、`capRight`、`dividerHard`、`dividerSoft`（U+E0B4、E0B6、E0BB、E0BC） | panel 標題膠囊；filu 的 `isWideIcon()` 把 U+E0A0–E0D7 排除在外（CJK icon 字型上仍是一格） |

會量到 icon 的地方：

| 地方 | 現在怎麼量 | icon 佔兩格時 |
|---|---|---|
| popup 的上框：`popup.go` `drawPopupBoxPad()` | `truncate(title, …)`、`dispW(title)` 算上框的 `─` 要補幾個 | 上框多一格，右上角 `╮` 被推出去，框歪 |
| 疊 popup：`app.go` `View()`（浮層與 toast）、`lockscreen.go` `View()`（PIN prompt） | `overlay.Composite`（bubbletea-overlay 自己量，看不到 icon 的寬度） | 標題那一列拼接的位置差一格 |
| custom saver 上的 PIN 框：`internal/custom` `paintBox()`、`clearBox()` | `ansi.StringWidth` 量框寬 | 框寬少算一格：置中偏半格、收起時清掉的矩形少一欄 |
| 點陣板：`canvas.go` `pixelCell = pixelGlyph + " "`、`newBoard(cols/2, …)`、`boardRows()` 的 `cols - b.w*2`、`plainRows()` | 一個像素固定兩格（glyph + 空白） | 一個像素變三格，整片板子超出終端機寬 |
| splash：`splash.go` `render()` | 同樣的 `pixelCell`，`logoW := cols * 2`，`lipgloss.PlaceHorizontal` / `Place` 置中 | 同上 |
| `[2]` 的色票列：`detail.go` `detailBody()` 的 `swatch()` 與 `used := lw + 2 + dispW(r.hex)` | 色票算一格 | 那一列多一格，右框被推出去 |
| panel 框：`chrome.go` `panelFrame()` 用 `dispW(l)` 補空白、`joinHorizontal()`（`lipgloss.JoinHorizontal`） | icon-blind | `[2]` 有色票的列跟著歪；`chainW()` 只有 powerline（一格），不受影響 |

不受影響的：`dim.go` `dimANSI()` 只改寫顏色碼，不量寬度；sidebar、menu 的列、key reference、footer、狀態列、PIN 的點、輸入框的值都沒有 icon。
`●`、`…`、`–`、`→`、框線字是 East Asian Ambiguous，不是 icon，D6 與 filu 的 `isWideIcon()` 都不算（終端機把 ambiguous 設成寬的情況不在這一條）。

screensaver 畫面本身：點陣板**算在內** —— 每個像素就是 `pixelGlyph`；custom saver 的程式輸出**不算** —— locku 原樣轉送、不量也不排版，
只有疊在上面的 PIN 框（標題有 `glyphLock`）算。

**規則**：D6（v0.1.18）—— app 啟動時探測 icon 佔幾格，所有量寬度的地方（補空白、截斷、框線、疊 popup）都走同一個顯示寬度函式；
L4 的畫面測試也跑一次「icon 佔兩格」。參考實作：filu `internal/ui/width.go`（`DetectIconWidth()`、`isWideIcon()`、`dispWidth()`、`dispClip()`）。

**怎麼改**（filu 定案後照搬，以下是 locku 要特別處理的地方）：

- 搬 filu 的 `iconCells`、`isWideIcon()`、`dispWidth()`、`dispClip()` 與探測（filu 現在在 `iconwidth_unix.go`，含環境變數覆寫）；locku 的
  `dispW()`、`truncate()`、`truncateHead()`、`clipANSI()`、`padRight()`、`padLeft()`、`fitLines()`、`wrap()` 改走它們，上表每一處跟著換。
  `internal/custom` 在 `ui` 之外，顯示寬度函式要放在兩邊都用得到的地方，或由呼叫端把框寬傳進 `paintBox()`。
- 探測放在三個入口：`cmd/locku/main.go` 的 `runSettings()`、`runLock()`（custom 走 `runLock()` → `runCustom()`，同一次探測）。`runLock()`
  裡放在 `termreply.DropPending()` **之前**：探測自己讀完 CPR 回答；逾時才到的回答被 `DropPending()` 清掉，就算漏進鎖裡，`termreply` 也會把
  以 `R` 結尾的 CSI 當成回報丟掉，不會被當成按鍵叫出 PIN 框或解鎖。鎖不能因為探測卡住：沿用 filu 的逾時（200 ms），失敗就當一格。
- 點陣板：icon 佔兩格時一個像素只畫 glyph、不接空白（仍是兩格），`cols/2` 這些幾何就不用動；splash 同理。
- 疊 popup：`overlay.Composite` 量寬度看不到 icon。filu 現在也還用它，照 filu 屆時的做法。
- 測試：`TestViewFitsTheTerminal`（設定畫面）、`TestViewIsExactlyTheTerminal`（鎖定畫面）、`TestRowsAreExactlyTheTerminalWide`（點陣板）
  各多跑一次 `iconCells = 2`（filu `width_test.go`、`loading_test.go` 的做法），量寬度用新的顯示寬度函式，不用 `lipgloss.Width`；custom 的
  `paintBox()` 量一次「標題有 icon 的框」回傳的寬度。
- 文件：README 兩份的需求段（Nerd Font 那句）可以補「CJK 用、icon 畫成兩格的 Nerd Font 也行」；`dev-remarks.md` 記探測怎麼做、放在
  `DropPending()` 之前的理由；`docs/ui.md` §4 或 §5 記「量寬度只走一個函式」。
- CHANGELOG `[Unreleased]` 加一條 Fixed，例：On a Nerd Font that draws icons two cells wide (the CJK ones), the board, the splash, the popups
  and their titles keep their shape。


## 已經符合、不用修的（對照 v0.1.17 → v0.1.18 的改動）

- **L5、K10（v0.1.19）**：focus 的 panel 畫雙線 `╔═╗`、失焦圓角（`chrome.go` 的框線選擇），不只靠顏色；沒有 PTY，K10 不適用。
- **K11、D2：模式名顯示在框的上框右側，外框換 Yellow**：不適用，locku 沒有模式（v0.1.4、v0.1.10、v0.1.14 已判）。照術語「模式」（panel 或
  popup 裡的暫時狀態，一部分鍵換成它自己的意思，`Esc` 離開）再逐一看過：
  - 預覽：整個畫面換成鎖定畫布、任何鍵回來，不是 panel 或 popup 裡的狀態，是既有的偏離（跟 splash 一樣）。
  - 字典（preference、tmux、screen 的 `[2]` 上的 `?`）：只是 note 的內容不同，鍵的意思沒變。
  - 錯 PIN 凍結的一秒（`inputPopup.frozen`）與鎖定畫面的冷卻倒數：app 把鍵吞掉的等待，不是使用者進去的狀態，也沒有自己的鍵。
  - `gg` 的第一個 `g`（`pendingG`）是組合鍵的一半；顏色草稿（`unsaved`）是資料的狀態，鍵的意思都沒變。
- **F1、D3：finder 的 focus 在哪一邊要看得出來**：不適用，locku 沒有 finder。options 是 menu（不打字）；`config file path` 框的提議是一個灰字建議，
  不是候選清單，`Tab` 只把提議接進來（K2），不在打字與清單之間切換。
- **D2：失焦 panel 邊框上的 hint 用 Overlay0 / Surface2**：不適用，locku 的 panel 邊框沒有 hint。`chrome.go` `panelFrame()` 只在上框畫標題
  膠囊（`docs/ui.md` §5 Chrome 表「Border hint：無」）；失焦時膠囊整條是 Surface2，那是標題，不是 hint。
- **D3 的 footer**：`keyLegend()` 本來就從右邊整組丟（D1），這次只補 popup 的下框（第 1 條）。
- **K10、K9：子程序還沒準備好可以不轉送按鍵；focus 在 PTY 裡時 `q`、`Ctrl-C` 屬於子程序**：不適用，locku 沒有把鍵交給子程序的 surface。
  custom saver 跑在 locku 的 pty 上，但鍵永遠由 locku 讀（鎖上任何鍵叫出 PIN 框，預覽任何鍵結束），不送進程式；`internal/login` 的 `su`
  由 locku 寫密碼；設定畫面與鎖上的 `q`、`Ctrl-C` 都是 locku 自己的（鎖定中 `Ctrl-C` 不離開是既有的偏離）。


## 待確認

沒有。
