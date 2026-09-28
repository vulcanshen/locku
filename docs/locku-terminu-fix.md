# locku — terminu fix

locku 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.9/principle)（tdp v0.1.9）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與描述該行為的設計文件段落。有意不修的，改寫成
`dev-remarks.md`「偏離 tdp」的一條並附理由。

盤點日期：2026-09-28（對照 tdp v0.1.8）。v0.1.8 只改了 popup 的規則：F1（六類）、F7（尺寸與位置，新）、F8（層疊時 dim，新），
連帶改了 T2（popup 蓋在上面時例外）、K3（錯誤寫在 input 預留的錯誤列）、D4（key reference 寬度改照 F7）。
tdp 在 locku 管的只有**設定畫面**與 **PIN prompt**；鎖定畫布與預覽是既有的偏離（`dev-remarks.md`「偏離 tdp」），
但 PIN prompt 底下畫什麼屬於 F8，所以第 4 條會碰到鎖定畫布。

---

## 先看

- **這是橫跨每一個 popup 的改動**，不要一個 popup 一個 popup 各修一次。做成四個共用的東西：
  1. 一個寬度函式（`min(terminal 寬 − 2, 120)`），取代 `popup.go` 的 `popupInnerW(screenW, want)`；
  2. 一條「高度打開時定好」的規則（locku 大多已經是，見「已經符合」）；
  3. 一個「最上層以外全部 dim」的繪製器，設定畫面的 `View()` 與鎖定畫布各用一次；
  4. input popup 裡一列預留的錯誤列，一般輸入框、PIN 輸入框、鎖定畫布的 PIN prompt 共用。
- **「放在最上層」= 按鍵路由 + `closeTop` + 繪製**（tdp D3）。F8 多了第四個要一致的地方：「哪一層是亮的」要跟這三處是同一個答案，
  用 `owns()` 判斷（D3），不用 `isActive()`。
- **改完把畫面印出來看**：暫時寫一個測試 `t.Log(ansi.Strip(m.View()))`（dim 要看顏色就不 strip），看完就刪。
  寬度、疊層、哪層亮只有這樣看得到。至少用 80 × 40（L1）與 200 欄各看一次。
- **逐處 mutation**：每一處修正單獨改回舊行為，跑對應的測試，確認會紅。寬度的 mutation 小心被上限夾住（filu 的經驗）——
  用 200 欄量 120 的上限、用 80 欄量 `− 2`。
- **守舊尺寸的測試改寫，不是刪掉**：量 ` · taken`、` · wrong` 標題尾綴的測試（`app_test.go` 154、185、800、899 行）改成量錯誤列。
- **畫出或描述 popup 的文件一起改**：`ui.md` §2.3、§3.1、§3.2（含 PIN prompt 四個狀態的框圖）、`ux.md` §2.1 的輸入表、
  §2.2 的 PIN 三連問、§2.3 鎖定畫布的 PIN prompt 表、§5 浮層行為、§A.2 `?` 那段的「框寬依最長的說明（tdp D4）」。
  README 兩份沒有畫 popup，只有面板的圖，不用改。帶日期的歷史紀錄（例如 `function.md` 決定 33 的「confirm 是 message class」）不改。
- **不發版**：等家族全部 app 與 tdp 都穩定後一起發（CHANGELOG 記在 `[Unreleased]`）。

## popup 盤點

設定畫面 9 個、鎖定時 1 個（PIN prompt，一般 saver 與 custom saver 兩條畫法），共 10 個。位置全部垂直、水平置中
（`overlay.Center`），toast 在底部往上 2 列；custom saver 的 PIN prompt 由 `internal/custom/custom.go` `paintBox` 自己置中。

| popup | 檔案 | F1 類 | 寬度（現在） | 高度（現在） |
|---|---|---|---|---|
| Space menu（`menu`） | `spacemenu.go` `view()` | menu | 依內容：`max(標題+6, label+說明+4, 區標題, hint+1)` | `min(列數, 畫面高 − 6)`，打開時定 |
| global operation（`globalMenu`） | 同上 | menu | 同上 | 同上（目前一列 `[q]uit`） |
| options（`options`） | 同上 | menu | 同上 | 色通道 0–255 固定 10 列一窗；其他 `min(列數, 畫面高 − 6)` |
| input（`input`，一般） | `inputpopup.go` `view()` | input | `max(40, 值+8, 提議+8, 欄名+3)`，**打字時跟著變** | 3 列，固定；錯誤寫在上框標題尾綴、邊框轉紅，沒有錯誤列 |
| PIN input（`input`，`masked`） | 同上 | input | 固定 48（`pinPromptW`，外框含邊） | 1 列；錯誤在標題尾綴 |
| confirm（`confirm`） | `confirm.go` `view()` | confirm | 依最長一行：`max(標題+6, 行+4)` | 行數，固定 |
| 離開的 confirm（`quitAsk`） | 同上 | confirm | 同上 | 同上 |
| help（`help`） | `helppopup.go` `layout()` | note | 依最長的說明（舊 D4） | `min(折行後列數, 畫面高 − 6)`，框內捲動 |
| 離開 confirm 的 help（`quitHelp`） | 同上 | note | 同上 | 同上 |
| toast（`toast`） | `toast.go` `view()` | toast | 訊息 + 4 | 1 列 |
| PIN prompt（鎖定畫布） | `pinprompt.go` `view()`、`lockscreen.go` `View()` | input | 固定 48 | 1 列；`· wrong`、`· try again in N s` 在標題，那時框內一列空白 |
| PIN prompt（custom saver） | 同上，經 `custom.go` `Overlay` 畫 | input | 固定 48 | 同上 |

所有寬度最後都過 `popup.go` `popupInnerW(screenW, want) = max(10, min(want, screenW − 6))`：外框最寬是畫面寬 − 4，沒有 120 的上限。

底下怎麼畫：設定畫面 `app.go` `View()`（425 行起）先畫面板與 footer，再照固定順序（menu → globalMenu → options → input → confirm →
help → quitAsk → quitHelp）用 `overlay.Composite` 一層一層疊上去，最後疊 toast。**沒有任何一層 dim**：底下的 popup 與面板都照原色畫。
鎖定畫布 `lockscreen.go` `View()`（452 行）：PIN prompt 開著時 `boardRows` / `plainRows` 把亮格與 accent 改畫 `backdropColor`（Surface2），
暗格不變；狀態列（`canvas.go` `statusRow`）照原色。custom saver：底下是程式自己的畫面，locku 只把框畫上去，沒有 dim。

---

## 1. popup 寬度不是 `min(terminal 寬 − 2, 120)` —— F7、D4

- **現況**：每個 popup 依內容算寬，再由 `popup.go` `popupInnerW(screenW, want)`（202 行）夾在 `screenW − 6`（外框 ≤ 畫面寬 − 4）；
  見上表。一般 input 的寬度含 `dispW(m.value)+8`（`inputpopup.go` 141 行），**打字時框會變寬**，`Backspace` 拒絕提議時又可能變窄。
  PIN input 與 PIN prompt 固定 48（`pinprompt.go` 36 行 `pinPromptW`）。help 的寬依最長的說明（`helppopup.go` `layout()`，註解寫 tdp D4）。
- **規則**：F7：寬度 `min(terminal 寬 − 2, 120)`，左右各留一欄，水平置中。toast 寬度照同一條。D4：key reference 不再依最長的說明，
  說明太長在框裡換行或截尾。
- **怎麼改**：
  - `popupInnerW` 改成只看畫面寬：`min(screenW − 2, 120) − 2`（內寬），拿掉 `want` 參數；每個 `view()` 不再算內容寬。
    畫面比 L1 還窄時（`TestViewFitsTheTerminal` 量到 40 欄）照同一條算，不要再有 `max(10, …)` 以外的特例。
  - 一般 input：值比框長時照舊 `truncateHead`（尾端在畫面上）；`value`、`placeholder` 不再影響寬度。
  - PIN input 與 PIN prompt：`pinPromptW` 拿掉，改用同一個寬度；`pinRow` 本來就從框的中央往兩側長，寬度變了照樣置中。
    這推翻 `ui.md` §3.1 / §3.2「48 欄」的決定（2026-09-24）——見待確認 1。custom saver 的 `paintBox` 依框的實際寬度置中、
    `rect` 記最大的範圍，框變寬不用改 `custom.go`。
  - help：`layout()` 的 `want` 計算拿掉，說明欄寬 = 內寬 − key 欄；折行照舊。註解裡的「tdp D4」改成 F7。
  - toast：同一個寬度，訊息靠左、其餘留白。
  - 測試：每個 popup 在 80、100、200 欄各開一次，量框的第一列寬度 = 78、98、120；一般 input 打 30 個字前後寬度不變。
    mutation：寬度改回依內容、拿掉 120 上限、`− 2` 改回 `− 6`，各自要紅。
  - 文件：`ui.md` §3.1 PIN input 列、§3.2 PIN prompt 列與框圖的「48 欄」、`ux.md` §A.2 的 `?` 那段「框寬依最長的說明，上限是螢幕（tdp D4）」。

## 2. input popup 沒有預留錯誤列，錯誤寫在標題尾綴 —— F7、K3

- **現況**：
  - 一般 input（`inputpopup.go` `view()`）：框內三列（欄名、空白、值），錯誤是 `suffix` 接在上框標題後面（` · empty`、` · taken`、
    ` · invalid`、` · absolute or ~/ path`、` · one key, e.g. l or C-l`、` · <err>`；`actions.go` 735、739、798、835、855、857、873、886 行），
    邊框轉紅。
  - PIN input：框內一列點點；目前 PIN 錯是 `freeze(" · wrong")`（`actions.go` 867 行）：標題尾綴、清空、吞鍵 1 秒、下框 hint 拿掉。
    `confirm PIN` 跟 `new PIN` 不一致時跳 **toast** `PIN mismatch`，再重開 `new PIN`（`actions.go` 882 行）。
  - 鎖定畫布的 PIN prompt（`pinprompt.go` `view()`）：`· wrong`、`· try again in N s` 寫在標題，那時框內一列清成空白。
- **規則**：F7：input popup 打開時高度就含一列錯誤列，沒有錯誤時空白；送出失敗時錯誤寫在這一列，框的高度不變。
  K3：不合格就不送出，把錯誤（哪個欄位、為什麼）寫在那一列。
- **怎麼改**：
  - 一般 input：`rows` 多一列錯誤列（放在值那一列下面），`suffix` 改成錯誤列的內容（Red），上框標題只寫型別。錯誤寫成一句話
    （`name taken`、`not a number`、`absolute or ~/ path` …），單一欄位的框「哪個欄位」就是框本身。何時清掉照舊（下一個鍵）。
  - PIN input：同一列錯誤列；`freeze` 的 `wrong` 寫進錯誤列。`PIN mismatch` 不再跳 toast，寫進重開的 `new PIN` 框的錯誤列
    （這是 `confirm PIN` 送出失敗，K3 要求寫在 input 裡）；ux.md §5 的 toast 清單拿掉 `PIN mismatch`。
  - 鎖定畫布的 PIN prompt：同一列錯誤列放 `wrong` 與 `try again in N s`，點點那一列在錯誤時照舊清空；`· closing` 不是錯誤，留在標題。
    邊框轉紅是 D2 的警示色，可以留著。三種 PIN 框共用 `pinRow` 之外，錯誤列也共用同一個畫法（`ui.md` 說三個 PIN 框長得一樣）。
  - 測試：每種 input 有錯與沒錯時 `View()` 裡框的列數相同；錯誤文字在框內、不在上框。改寫 `app_test.go` 154、185、800、899 行量
    `suffix` 的斷言；`lockscreen_test.go` 156 行量 `try again in 30 s` 的照樣過，另加「在框內那一列」。mutation：錯誤寫回標題、錯誤列只在有錯時才加。
  - 文件：`ui.md` §3.1 input 列（`number · invalid`、`name · taken` 寫在邊框）、§3.2 四個狀態的框圖、`ux.md` §2.1 的輸入表
    （name、config file path 等「邊框 ` · …` 框留著」）、§2.2 Change PIN 列、§2.3 PIN prompt 表的 Enter 與連錯兩列、`ui.md` §4 色帶的 Red 用途。

## 3. 設定畫面：popup 開著時底下不 dim —— F8

- **現況**：`app.go` `View()`（454–470 行）照固定順序把每個 `isActive()` 的 popup 用 `overlay.Composite` 疊上去，面板、footer、
  底下的 popup 全部照原色；草稿未存的 Yellow `unsaved` 膠囊等顏色也照原色。
- **規則**：F8：有 popup 開著時，最上層以外的一切——底下的 popup 與整個 base 畫面——都用 dim 色畫，警示色也 dim；toast 不觸發 dim；
  popup 邊框依層數的顏色照舊保留。
- **怎麼改**：
  - 在 `View()` 先決定「最上層」：照繪製順序，最後一個 `owns()` 的 popup（toast 不算，跟 `popupDepth()` 不數 toast 一致）。
    沒有這一層就照舊不 dim。
  - base（面板 + footer）整塊交給一個 dim 繪製器：每一列去掉樣式、以 `dimColor` 重畫，寬度不變（`TestViewFitsTheTerminal` 照樣要過）。
  - 最上層以下的 popup 用同一個繪製器畫框內的列；邊框依 `popupLayerColor(layer)` 照舊（見待確認 4）。做法是 `drawPopupBox` 多一個
    `dimmed` 參數，或每個 popup 的 `view()` 收一個 `dimmed`，不要在 `View()` 裡對整個框字串做 strip（會連邊框一起洗掉）。
  - 正在關閉的 popup：它已經不 `owns()`，底下那層在它開始關的那一刻就亮；關閉中的框本身以 dim 畫完動畫。跟按鍵已經交給底下那層一致。
  - toast 永遠用自己的顏色疊在最上面，不 dim，也不讓底下 dim。
  - 測試：menu 開著時 base 的某段文字（例：`[1] locku`）不含原色、含 `dimColor`；menu 上再開 confirm 時 menu 的列變 dim、confirm 不 dim；
    `Esc` 關 confirm 後 menu 回到亮的；只有 toast 時 base 不 dim。mutation：拿掉 base 的 dim、拿掉下層 popup 的 dim、最上層改用 `isActive()`、
    toast 也算一層，各自要紅。
  - 文件：`ux.md` §5 浮層行為加一段 dim；`ui.md` §3 開頭與 §4 色帶（dim 的用途多一條「popup 底下的一切」）。

## 4. 鎖定畫布：PIN prompt 底下的狀態列不 dim —— F8

- **現況**：PIN prompt 開著時，`lockscreen.go` `View()` 把 `dimmed` 傳給 `boardRows` / `plainRows`（`canvas.go`），亮格與 accent 改畫
  `backdropColor`（Surface2），暗格不變（2026-09-24 使用者定案：背景變色但不停）。這已經是 F8 的意思。沒 dim 的是最後一列
  `statusRow`（`canvas.go` 429 行）：`config error: …` 與 custom saver 的結束原因是 Red，`user@host · locked since` 是 `dimColor`。
  （`no PIN · any key unlocks` 的 Yellow 不會碰到：沒有 PIN 就不會有 PIN prompt。）
- **規則**：F8：最上層以外的一切都 dim，警示色也一起 dim（T2 的例外也寫明了）。
- **怎麼改**：`statusRow` 收 `dimmed`，為真時整列用畫布的 dim 色畫（用哪個色見待確認 2）。暗格照舊不變：暗格畫的是 profile 的底色，
  換成 dim 色反而比原本亮。
  - 測試：有 `problem` 的鎖開 PIN prompt，最後一列不含 `warnColor`；關掉後又含。mutation：拿掉 `statusRow` 的 dim。
  - 文件：`ui.md` §2.3「PIN popup 開著時亮格改畫 Surface2、暗格不變」補上狀態列；`ux.md` §5 最後一段「鎖定畫布」同樣補上。
  - custom saver 的 PIN prompt 底下是程式的畫面，見待確認 3。

## 5. 程式註解與設計文件還用舊的 popup 分類名 —— F1

- **現況**：`confirm.go` 264 行註解「confirmPopup is the message class (tdp F1)」、`toast.go` 337 行「the message class with an auto-dismiss」；
  `ui.md` §3.1 表的「類型」欄：`?` help 是 `viewport`、confirm 與 toast 是 `message`。
- **規則**：F1：六類 menu / confirm / input / note / toast / terminal。
- **怎麼改**：註解改成 confirm、toast 類；`ui.md` §3.1 表：help → note、confirm → confirm、toast → toast，並補上漏列的 global operation popup
  （menu）。`function.md` 決定 33 是帶日期的歷史，不改。只是用語，沒有行為變動，不用測試。

---

## tdp v0.1.9 定案（2026-09-28，回答本檔與其他 app 共同的待確認）

v0.1.9 只補了 v0.1.8 popup 規則的細節。本檔的條目與「待確認」照下面改讀；修的時候以這裡為準。

- **F8 邊框**：底下那幾層 popup 的**邊框也一起 dim，但保留層色** —— 畫成它自己層色（D2）的 dim 版本，不是統一的 dim 色。
  內容照 F8 用 dim 色。只有最上層是亮的。
- **F1 input 附候選清單**：邊打字邊篩選的清單仍算 input：可列印的鍵一律是字元（`j`、`k` 也是），只有方向鍵在候選之間移動，
  `Enter` 送出選中的那一筆。不必拆成兩階段。
- **F1 finder**：`Tab` 在打字與結果清單之間切換 focus；`Esc` 關掉整個 finder（階段不是一層）。
- **F1 多步驟**：流程的每一步是自己的 popup，疊起來（F4 保留 source），不在同一個框裡換內容；每一步有自己打開時定好的高度。
- **F7 錯誤列**：只有**送出可能失敗**的 input 預留錯誤列；送出不會失敗的（例：多行編輯器）不必。
- **F7 terminal 類例外**：PTY popup 寬高用滿可用範圍（terminal 寬 − 2 × 高 − 2），不受 120 欄上限。

- **本檔**：待確認第 4 題（邊框）照上面定案。PIN 框 48 欄、鎖定畫布的 dim 色、custom saver 底下無法 dim 三題仍待 user 在 locku 決定。

## 6. 改 PIN 的每一步不是自己的 popup —— F1（v0.1.9 多步驟）、F4

（locku session 對照 v0.1.9 時補的，2026-09-28。）

- **現況**：`current PIN` 送出後 `m.input.close()` 再開 options（New / Remove），兩步沒有疊起來；`new PIN` 送出後
  `askPIN("confirm PIN", …)` 用同一個 `input` 換內容（`actions.go` 877 行），不是新的一層。`Esc` 取消整串、回到 menu（`ux.md` §5）。
  不一致時跳 toast `PIN mismatch`，再重開 `new PIN`。
- **規則**：v0.1.9 F1：多步驟流程的每一步是自己的 popup，疊起來（F4 保留 source），不在同一個框裡換內容；每一步有自己打開時定好的高度。
  F4 / K4：`Esc` 只關最上層，回到上一步。K3：送出失敗，錯誤寫在那個 input 的錯誤列，框留著。
- **怎麼改**：
  - 同一時間可以有好幾個 input 開著：`current PIN` → options（New / Remove）→ `new PIN` → `confirm PIN` 一層疊一層，底下的留著。
    Set PIN（還沒有 PIN）是 `new PIN` → `confirm PIN` 兩層。整串完成（PIN set / removed）才整疊清掉，連同底下的 menu（F4、T1）。
  - `Esc` 一次退一步：在 `confirm PIN` 按 `Esc` 回到 `new PIN`（值照留或清空，照 input 的慣例），在 options 按 `Esc` 回到 `current PIN`。
  - 不一致：錯誤寫在 `confirm PIN` 的錯誤列、值清空、框留著；要重打新的 PIN 就 `Esc` 回 `new PIN`。不再跳 toast。
  - 路由、`closeTop`、繪製、`popupDepth`、`owns()`、F8 的「哪一層亮」都要看得到這一疊。
  - 測試：每一步開著時底下那一步還 `owns()`；`Esc` 從 `confirm PIN` 回到 `new PIN`、從 options 回到 `current PIN`；不一致時錯誤在
    `confirm PIN` 的框內、沒有 toast；完成時整疊清掉。mutation：換內容而不是疊、`Esc` 關整串、不一致跳 toast。
  - 文件：`ux.md` §2.2 PIN 三連問、§5 浮層行為的「PIN 那串中途 `Esc` 取消整串」、toast 清單拿掉 `PIN mismatch`；`ui.md` 的層數。

## 已定案（2026-09-28，使用者）

1. **PIN 框寬度照 F7**：PIN 最多 64 字，每個點後面一格空白、再一格游標，整列 129 欄，超過 120 的上限（使用者：「用這個總寬和 120 來判斷」），
   所以 PIN 框就是 `min(W − 2, 120)`，跟其他 popup 一樣，不寫偏離。放不下時照舊截掉開頭，游標那端在畫面上。
2. **鎖定畫布的 dim 色統一成 Overlay0**（`#6c7086`，設定畫面的 dim 色）：畫布的亮格與 accent、第 4 條的狀態列都用它，
   取代 2026-09-24 的 Surface2（`backdropColor`）。
3. **custom saver 的 PIN prompt 底下不 dim**：寫成 `dev-remarks.md`「偏離 tdp」的一條（F8：底下是別的程式的畫面，locku 沒有副本）。
4. 邊框：照 v0.1.9 定案（見上）。
5. **第 6 條照 tdp**：`Esc` 一次退一步；不一致時錯誤留在 `confirm PIN` 的框裡，不跳 toast。

## 待確認（已由上面「已定案」回答）

1. **PIN 框的 48 欄。** `ui.md` §3.1 / §3.2 記著使用者 2026-09-24 定的「48 欄、解鎖與設定一樣」。照 F7 改成 `min(W − 2, 120)` 以後，
   三個設定畫面的 PIN 框與鎖定畫布的 PIN prompt 仍然一樣寬（「一樣」這半句還成立），只是不再是 48。第 1 條預設照 F7 改；
   若使用者要留 48，得寫成 `dev-remarks.md`「偏離 tdp」的一條（F7）。
2. **鎖定畫布的 dim 色是 Surface2 還是 Overlay0。** 畫布在 PIN prompt 底下退成 Surface2（`#585b70`，`backdropColor`）是 2026-09-24 的定案；
   設定畫面的 dim 色（D2「暗字、hint」）是 Overlay0（`#6c7086`）。兩者不衝突：畫布本來就在做 F8 要的事，F8 也沒寫死色碼。
   要決定的只有：畫布（含第 4 條的狀態列）維持 Surface2，還是跟設定畫面統一成 Overlay0。預設維持 Surface2，狀態列也用它。
3. **custom saver 的 PIN prompt 底下沒辦法 dim。** 框疊在程式還在動的畫面上：程式的輸出直接送到終端機（`internal/custom/custom.go`
   `screen.Overlay` / `paint` 只畫框），locku 沒有那張畫面的副本，要 dim 就得在 locku 裡放一個終端機模擬器重畫程式的畫面。
   使用者 2026-09-25 也定過「框之外什麼都不畫」（`ui.md` §2.3、§3.2）。建議寫成 `dev-remarks.md`「偏離 tdp」的一條（F8：底下是別的程式的
   畫面），不修；要 user 點頭。
4. **底下那幾層 popup 的邊框要不要一起 dim。** F8 說「最上層以外全部 dim」，又說「popup 邊框依層數的顏色（D2）照舊保留」。第 3 條照字面讀：
   下層 popup 框內的內容 dim、邊框維持它那一層的顏色。若 tdp 的意思是「整個下層框都 dim，保留的只是最上層依層數取色」，第 3 條的
   `dimmed` 也要套到邊框。這是家族共同的讀法，最好在 terminu 那邊定一次。

## 已經符合、不用修的（對照 v0.1.8）

- **F1 分類**：10 個 popup 各自只屬於六類之一，沒有同時混兩類的。Space menu、global operation、options 是 menu（`j/k` 走、`Enter`
  或熱鍵執行）；confirm、quitAsk 是 confirm；input（一般與 PIN）與 PIN prompt 是 input；help、quitHelp 是 note（唯讀、`j/k/u/d` 捲動、
  沒有可執行的列，K6 起就是這樣）；toast 是 toast（`Esc` 或時間到收掉，其他鍵照常到 panel）。PIN 框的 wrong（1 秒）與 lockout
  （倒數）只是 input 在錯誤裡吞鍵，不是換類。splash 與預覽不是 popup（S 章；預覽是既有偏離）。沒有 terminal 類：custom saver 跑在 pty 上，
  但按鍵永遠在 locku 手上（K10 那輪已判過）。
- **F7 高度打開時定好**：沒有 popup 在開著時跟著內容伸縮。menu / options 的列在 `setItems` 時定下，開著時不再換（`app.go` 344、359 行、
  `actions.go` 545、718 行都在 `open()` 之前）；confirm 的行、help 的列（`entries` 在 `open` 時給定）、input 的三列 / 一列、toast 的一列、
  PIN prompt 的一列都不變。上限是畫面高 − 6（`capRows`、`visible()`），超過就在框裡捲動（help 的 `top`、menu 的視窗）。
  視窗大小改變時重算：F7 說的是「不跟著內容伸縮」，terminal 自己變了不在此列。第 2 條加錯誤列時要守住這一點（錯誤列一開始就在）。
  開關時的長高是 F2 的動畫（`animRows`），不算伸縮。
- **F7 位置**：全部 `overlay.Center` 垂直、水平置中；toast 在底部（`overlay.Bottom`，往上 2 列）；鎖定畫布的 PIN prompt 置中，custom 的由
  `paintBox` 置中。
- **F8 邊框依層數取色**：`popupLayerColor(layer)`，layer 由 `popupDepth()` 數開著的 popup（不數 toast）。第 3 條不動它。
- **F8 toast 不觸發 dim**：目前沒有 dim，自然不觸發；第 3 條照 `popupDepth()` 不數 toast 的同一個判斷。
- **T2**：設定畫面沒有串流內容，失焦也不變暗（D2「失焦只換邊框」，面板只換邊框）；鎖定畫布的 saver 在 PIN prompt 底下照常 tick、
  照常揭露、一起 dim，正是 T2 新加的例外。
- **K3 其他部分**：設定畫面的輸入框都是單一欄位，`Enter` 一律送出這一欄，不合格就不送出、框留著；PIN 三連問是三個接連的單一輸入框，
  不是 input group（v0.1.4、v0.1.6 那輪已判過）。這次只差錯誤寫在哪（第 2 條）。
- **D4 其他部分**：menu 列的樣式、熱鍵括號、`j/k move · Enter run · Esc close` hint 都沒變；只有寬度（第 1 條）。

連結：README 兩份、`dev-remarks.md`、`ui.md`、`ux.md` 開頭的 tdp 連結已改釘 `v0.1.9`；`ui.md` §3.1、`ux.md` §A.1 的「tdp v0.1.7 M2」是帶日期的紀錄，不改。
