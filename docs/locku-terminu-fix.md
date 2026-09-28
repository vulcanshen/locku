# locku — terminu fix

locku 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.12/principle)（tdp v0.1.12）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與描述該行為的設計文件段落。有意不修的，改寫成
`dev-remarks.md`「偏離 tdp」的一條並附理由。

盤點日期：2026-09-28（對照 tdp v0.1.11）。v0.1.11 改了 K3（沒有項目的內容 panel 上的 `Enter`）、F6（picker 的選擇可以算確認）、
F7（高度只在 loading 與使用者自己的操作時可變，loading 要有轉圈 icon）、F8（dim 是把每個顏色——前景與背景——往底色淡化，
不准剝色重畫）與 D2（dim 的算法 `c × 0.45 + base × 0.55`，filu 的 `dim.go` 是參考實作）。
locku 這次要修的只有 **F8 / D2 的 dim 做法**，兩處：設定畫面（第 1 條）與鎖定畫布的 PIN prompt 底下（第 2 條）；
其餘逐條對照過，見「已經符合」。

---

## 先看

- **做成一個共用的 dim，不要一處一處修。** 把 filu 的 `internal/ui/dim.go`（`dimANSI` / `dimSGR` / `dimRGB` / `xterm256`）
  搬進 locku 的 `internal/ui`，改寫**已經畫好的字串**裡每一個 SGR 顏色碼。設定畫面的 `View()` 與鎖定畫布的 `View()` 都只呼叫它；
  `popup.go` 的 `dimBase`、`dimPopup`、`dimOf` 三個都被它取代（底下 popup 的邊框本來就是用層色畫的，淡化後自然就是
  「層色的 dim 版」，不必再特別處理邊框）。base 用 `theme.go` 的 `baseHex`，沒有前景的文字給 `dim(textColor)`。
- **要看顏色才驗得到。** 測試的輸出不是終端機，lipgloss 預設不畫顏色：照 `dim_test.go` 的 `colours(t)` 打開 TrueColor。
  改完把 `View()` 印出來，**不要 strip**，直接貼到終端機看：Space menu 打開時，`[1] locku` 與 `[2] <name>` 的膠囊、黃色 `unsaved`
  膠囊、`[1]` / `[2]` 的 cursor bar 都要還在，只是變暗（`LOCKU_DUMP=1 go test ./internal/ui -run TestDump -v` 也要先打開顏色）。
  至少 80 × 40（L1）看一次。
- **測試要能看背景。** `dim_test.go` 的 `has()` 只比對前景（`38;2;…`），要另外一個比對背景（`48;2;…`）的，膠囊與 cursor bar 靠的是背景。
- **逐處 mutation**：每一處修正單獨改回舊行為（換回 `ansi.Strip` 重畫、`dimSGR` 丟掉背景那一支、把所有前景換成同一個色、
  鎖定畫布的亮格換回 `backdropColor`），跑對應的測試，確認會紅。
- **守舊 dim 的測試改寫，不是刪掉**：`dim_test.go` `TestPopupsDimBelowTheTop` 與 `lockdim_test.go` `TestPromptDimsTheLock`
  現在量的是 `dimColor`（Overlay0）與 `dimOf`（混一半），改成量 D2 的淡化結果；「哪一層亮」「toast 不 dim」「關閉中那一刻底下就亮回來」
  那幾段照留。
- **描述 dim 的文件一起改**：`ui.md` §2.3（鎖定畫布，PIN popup 開著時「亮格改畫 Overlay0、暗格不變」）、§4 色帶表的 Surface2、Overlay0、
  popup layer scale 三列（「往 base 混一半」）；`ux.md` §B 的 `dim` 那一列、§5「最上層以外全部變暗」與最後的「鎖定畫布」段；
  `dev-remarks.md`「偏離 tdp」custom saver 那條的最後一句（「一般 saver 的畫布照 F8 變暗（亮格與狀態列改 Overlay0）」）。
  帶日期的歷史紀錄不改，照慣例在後面補一句新的日期與做法。README 兩份沒有寫 dim，不用改。
- **不發版**：等家族全部 app 與 tdp 都穩定後一起發（CHANGELOG 記在 `[Unreleased]`）。

---

## 1. 設定畫面的 dim 剝掉顏色重畫，拆掉膠囊、cursor bar 與色票 —— F8、D2

- **現況**：`app.go` `View()`（493–503 行）有 popup 時呼叫 `popup.go` 的兩個函式：
  - `dimBase`（201 行）：每一列 `ansi.Strip` 之後整列用 `Foreground(dimColor)` 重畫。**所有前景變成同一個 Overlay0、所有背景丟掉**；
    沒有前景的文字也是 Overlay0（D2 要的是 `dim(Text)`）。
  - `dimPopup`（212 行）：同樣 strip，上下框整列與每列頭尾一個字畫 `dimOf(層色)`，框內全部 Overlay0。
  - `dimOf`（229 行）：`(c + base) / 2`，混一半；D2 是 `c × 0.45 + base × 0.55`。

  靠背景畫出來、因此現在一 dim 就壞掉的東西：
  - **panel 標題膠囊**（`chrome.go` `titleChain`）：每一段是 base 色的字壓在填色背景上（focus 的藍、Surface2、`unsaved` 的黃），
    圓角 cap 是填色當前景，接縫三角形是「左段色前景 + 右段色背景」。dim 後背景沒了：`[1] locku`、`[2] <name>`、
    `unsaved` 變成兩個灰色半圓夾著的灰字，膠囊的本體不見。
  - **cursor bar**：`sidebar.go` 116–117 行、`detail.go` 437–438 行（base 字壓在 `handColor` / `borderDim` 背景上）——Space menu 一開，
    底下看不出 cursor 在哪一列；`spacemenu.go` 256–257 行的 menu / options cursor bar，menu 被 confirm、input、`?` 蓋住時一樣消失。
  - **輸入框的游標塊**：`inputpopup.go` 168 行、`pinprompt.go` 116 行 `pinRow`（base 字壓在 `editColor` 背景上）——PIN 那串往上疊時
    （`pinCurrent` 在 options 下、`new PIN` 在 `pinConfirm` 下）、輸入框在離開的 confirm 下，游標塊消失。
  - **`[2]` 的色票與滑桿**：`detail.go` 的 `swatch`（色票格用那個顏色本身當前景）變成 Overlay0，看不出是什麼顏色；滑桿的 `tint`
    （486–489 行，通道色前景 + 反向灰階背景）背景丟掉。
  - **狀態色**：`unsaved` 的黃、activate 的 Green / Mauve、`not set` 的黃、錯誤的紅，全部變成同一個 Overlay0，不是各自的 dim 版。
- **規則**：F8：dim 是把每一個顏色——前景與背景——往底色淡化，形狀與版面不動；不可以剝色重畫、丟背景、把前景換成同一個 dim 色；
  邊框是自己層色的 dim 版。D2：`dim(c) = c × 0.45 + base × 0.55`，前景背景都算，16 / 256 色先換 RGB，沒有前景的文字給 `dim(Text)`。
- **怎麼改**：
  - 搬 filu 的 `dim.go`（見「先看」），`View()` 裡 `out = dimBase(out)` 換成 `dimANSI(out)`、`dimPopup(v, f.layer)` 換成 `dimANSI(v)`；
    刪掉 `dimBase`、`dimPopup`、`dimOf`，以及 `popup.go` 因此沒人用的 `ansi` import（只有這兩個函式用到）。「最上層」照舊用 `owns()` 判斷。
  - 測試（改寫 `TestPopupsDimBelowTheTop`）：Space menu 打開時，`[1] locku` 那一列有背景 `dim(focusColor)`、沒有 `focusColor`；
    focus 在 `[2]`、有未存草稿時標題列有背景 `dim(yellowColor)`；`[1]` 的 cursor 列有背景 `dim(handColor)`；confirm 疊上去後 menu 的框是
    `dim(popupLayerColor(1))`（D2 算法，不是 `dimOf`），menu 的 cursor bar 背景還在；最上層照原色；toast 不 dim。
  - mutation：換回 strip 重畫、`dimSGR` 不處理 `48`、所有前景換成一個色，各自要紅。
  - 文件：`ux.md` §B `dim` 列與 §5「最上層以外全部變暗」（「改用 Overlay0 畫」「往 base 混一半」）、`ui.md` §4 的 Overlay0 與
    popup layer scale 兩列。

## 2. 鎖定畫布：PIN prompt 底下的亮格與 accent 換成 Overlay0、暗格不動、狀態列剝色 —— F8、D2

- **現況**：`lockscreen.go` `View()`（454–481 行），PIN prompt 開著時：
  - `boardRows` / `plainRows`（`canvas.go` 359、395 行）收 `dimmed`，把 fg 與 accent 都換成 `backdropColor`（`theme.go` 37 行，就是
    Overlay0）——**兩種前景收成同一個色**：clock 的 accent 段、`EXIT <code>` 的 peach / 綠與白字分不出來了；暗格（profile 的 bg，
    畫成前景色的像素格）**完全不淡化**。
  - 狀態列走 `dimBase`（476 行）：strip 後整列 Overlay0，`config error` 的紅、custom 結束原因的紅都變成同一個灰。
  - 2026-09-28 使用者定的是「跟設定畫面同一個暗色」；設定畫面的做法照第 1 條換掉之後，這裡也要跟著換，意思才一樣。
- **規則**：F8、D2（同第 1 條）；F8 明寫警示色與串流內容（一直在 tick 的畫布）照同一個淡化。
- **怎麼改**：
  - `boardRows` / `plainRows` 拿掉 `dimmed` 參數，照原色畫；`View()` 把畫布各列加上狀態列組好之後，prompt 開著就整塊過一次
    第 1 條的 `dimANSI`，再疊上 prompt。結果是亮格 `dim(fg)`、accent `dim(accent)`、暗格 `dim(bg)`、狀態列各色各自淡化。
    `backdropColor` 之後沒人用就刪掉。
  - 注意：很暗的 bg（例如 `#000000`）淡化後是 `#111119`，比原本略亮——這就是 D2「往底色淡化」的意思，不是 bug
    （舊 fix 清單當時為了這個讓暗格不變，現在 F8 明寫背景也要淡化）。
  - 測試（改寫 `TestPromptDimsTheLock`）：prompt 開著時畫布有 `dim(fg)`、沒有 `fg`、沒有 Overlay0；有 accent 的 saver（例如 `EXIT`
    的字）兩種前景淡化後仍是兩個色；狀態列有 `dim(warnColor)`、沒有 `warnColor`；關掉後恢復原色（這段照留）。
  - mutation：亮格換回 `backdropColor`、狀態列換回 strip，各自要紅。
  - 文件：`ui.md` §2.3 與 §4 的 Surface2 / Overlay0 列、`ux.md` §5 最後的「鎖定畫布」段、`dev-remarks.md`「偏離 tdp」custom saver 那條的
    最後一句。
  - **custom saver 的偏離照留**（重新對照過新的 F8）：新的做法仍然是「改寫已經畫好的畫面」，而 custom saver 的畫面是程式直接寫到
    終端機的，locku 手上沒有那張畫面，偏離的理由沒變。只改那條最後一句對一般 saver 的描述（見待確認 1）。

---

## 已經符合、不用修的（對照 v0.1.11）

- **K3（沒有項目的內容 panel 上的 `Enter`）**：locku 沒有這種 panel。`[1]` 永遠有列；`[2]` 每一頁都有可停的列——profile 與 saver 的
  欄位列（custom saver 只有 `command`，但那一列可停，`detail.go` 199 行）、preference、tmux / screen 的設定列。`detail.go` `rowAt()` 的
  「沒有 stop」只是防呆，走不到。
- **F6（picker 的選擇可以算確認）**：locku 要 confirm 的是 Delete profile、有未存草稿時的 Quit、tmux / screen 的 activate on / off
  （`actions.go` 383、407、416 行，與 `quitAsk`），都不是從 picker 選出來的。從 picker 選了就生效的只有 options 裡的值（layout、size、
  lock、profile……）與改 PIN 那串的 `Remove PIN`（`ux.md` §5：前面已經驗過 current PIN）；這些本來就不 confirm，v0.1.11 明寫這樣可以。
- **F7（高度只在 loading 與使用者的操作時可變；loading 要轉圈 icon）**：locku 沒有打開時內容還不確定的 popup——沒有串流、沒有網路、
  沒有搜尋，PIN 比對與存檔都是同步的（`internal/ui` 裡除了動畫、toast、閒置與 lockout 的計時，沒有非同步的 `tea.Cmd`），所以不需要
  loading icon。開著時列數會變的也沒有：Space menu、global operation popup、options 的列在打開時定好；input 與 PIN 框固定兩列（值 +
  錯誤列）；lockout 的倒數只換錯誤列的字；`?` 在固定高度的框裡捲動。也沒有背景事件會改 popup 的列數。
- **F8 的其他部分**：「哪一層亮」用 `owns()`，跟按鍵路由、`closeTop` 同一個答案（D3）；toast 不觸發 dim、自己不被 dim；底下的 popup
  各自保留層色——這些是 v0.1.9 那輪修的，只有**怎麼 dim** 要照第 1、2 條換。
- **v0.1.9「多步驟的流程，每一步是自己的 popup」**：重新對照過。改 PIN 是 `pinCurrent` → options（New / Remove）→ `input`（new PIN）→
  `pinConfirm` 四個疊起來的框（`actions.go` `askPIN`）；設 PIN 是 `input` → `pinConfirm`；新增、複製、改名 profile、數字欄、路徑、bind
  key 都是一步一個框；activate 是一個 confirm。沒有在同一個框裡換內容的流程。
- **其他 rules**：K1–K11、M1–M9、L1–L5、F1–F5、X、T、S 在 v0.1.4–v0.1.10 各輪都對照過，v0.1.11 沒有改動，程式也沒有動過這些地方
  （上一輪之後的 commit 只有改連結）；既有偏離（鎖定畫布不套 core key、鎖定中 `Ctrl-C` 不離開、預覽任何鍵回來、三個 `[2]` 的 `?` 是
  每列說明、custom saver 底下不 dim）都寫在 `dev-remarks.md`「偏離 tdp」，理由沒變。

## tdp v0.1.12 定案（2026-09-28，回答四個 app 在 v0.1.11 盤點時的共同問題）

修的時候以這裡為準；本檔的條目與「待確認」照下面改讀。

- **F7 loading icon 一定要放**：popup 在 loading 時，標題後面一定放輪轉的 loading icon，**跟高度會不會變無關**。
  loading 指**整個 popup** 的內容還沒到；若只是**某一個項目**本身是持續進來的資料流，loading 的是那個項目，怎麼揭露由 app 決定。
- **F7 使用者操作造成的高度變化是「允許」不是「要求」**：app 可以維持原高；原則是揭露的資訊要正確。
- **D2 淡化絕不讓顏色變亮**：每個通道取原值與淡化值較小的那個；比 base 還暗的顏色（例：`#000000`）維持原色。
- **D2 / D6 truecolor**：家族要求 truecolor terminal，dim 一律輸出 24-bit，不必照色彩深度降階；README 的需求段跟 Nerd Font
  並列寫上「需要 truecolor terminal」（D6，家族預設）。
- **D3 loading icon 規格**：Nerd Font `nf-md-circle_slice_1`–`_8`（U+F0A9E–U+F0AA5）八格；一格 90ms；由時鐘決定哪一格
  （`frames[(now / 90ms) % 8]`），tick 只在有東西 loading 時續排；寬一格；顏色跟旁邊的字，放在 popup 標題後面時用該層層色（bold）。

- **本檔**：
  - 第 2 條「暗格不動」的疑問照 D2 定案：淡化照做，但**比原色亮就維持原色**，所以很暗的暗格（例：`#000000`）不會浮起來，
    locku 之前「暗格不變」的效果在這些顏色上自然保住。
  - 新增：dim helper 要照 D2 加「絕不變亮」這一步（搬 filu `dim.go` 時一起加，filu 也要補）。
  - 新增：README 兩份的需求段補「需要 truecolor terminal」。
  - 待確認「custom saver 的輸出串流要不要淡化」仍是 locku 自己的決定。

## 3. README 沒寫需要 truecolor terminal —— D6（v0.1.12）

- **現況**：兩份 README 的需求段只寫 Nerd Font。
- **規則**：D6：家族要求 truecolor terminal；D2：dim 一律輸出 24-bit。
- **怎麼改**：兩份 README 需求段跟 Nerd Font 並列，寫上需要 truecolor terminal（以使用者的語言寫：沒有的話顏色會失真）。

## 已定案（2026-09-28，使用者）

1. **custom saver 維持偏離**：不做串流淡化（理由不變：locku 手上沒有程式的畫面，串流淡化只會讓 PIN 框開之後重畫的格子變暗）。
   只把 `dev-remarks.md` 那條最後一句改成新的淡化做法。

## 待確認（已由上面「已定案」回答）

1. **custom saver 的 PIN prompt 底下，要不要改成「淡化之後才畫出來的部分」？** 新的 F8 是改寫 SGR，這件事 locku 其實可以對程式的
   輸出串流做（`internal/custom` 的 screen writer 本來就逐段轉送程式的輸出）：prompt 開著時把經過的 SGR 照 D2 淡化。但只有 prompt
   打開**之後**程式重畫的格子會變暗，已經在畫面上的格子要等程式自己重畫——cmatrix 這種一直在畫的很快就整片暗掉，靜態的畫面
   可能一直不變。建議維持偏離（理由仍是「手上沒有畫面」），只把那條最後一句改成新的做法；要試串流淡化的話，另外成一條。
