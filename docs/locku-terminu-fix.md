# locku — terminu fix

locku 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.4/principle)（tdp v0.1.4）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）、`docs/ux.md` 與 `docs/dev-remarks.md` 裡描述該行為的段落。有意不修的，改寫成
`dev-remarks.md`「偏離 tdp」的一條並附理由。

盤點日期：2026-09-27（對照 tdp v0.1.4）。locku 已照 tdp v0.1.0 修完（v0.1.2、v0.1.3），也跟上了 v0.1.1；這裡只列
v0.1.2–v0.1.4 帶來的差距。行號以當天的 `main` 為準，會很快過期，改的時候照函式與內容找。


## 先看：locku、webu、sshu 修完的經驗（2026-09-27 更新）

locku 自己（v0.1.2、v0.1.3）、webu（v0.4.0）與 sshu（對照到 v0.1.4，尚未發版）修的時候發現這些，這裡的各條也適用：

- **每修一條補一個 model test，並逐處做 mutation**：把每一個修正單獨改回舊行為，確認對應的測試會紅。新測試用到新欄位時，
  「拿舊程式碼整個跑一次」會編譯不過，逐處 mutation 才量得到（sshu）。mutation 沒被抓到時，先看是不是用被改掉的函式去量結果、
  或改在走不到的分支上。
- **守舊規則的測試要改寫，不是刪掉**：改名並反轉斷言，測試名稱寫出新規則。本檔第 1、3 條會讓幾個既有測試變紅（各條列出）。
- **離開的 confirm 用自己的 popup，不借共用的 confirm**（sshu 的 `quitAsk`，tdp D3）：`Ctrl-C` 在任何地方都叫得出離開流程，
  它可能疊在另一個 confirm、輸入框或 help 上；借同一個會蓋掉使用者正在回答的問題。見第 4 條。
- **「最上面」只能有一個答案，「放在最上層」要同時改三處：按鍵路由、`closeTop`、繪製順序**（sshu，tdp D3）。少改一處就是一種
  bug：路由錯了 `Enter` 會確認底下看不見的問題，`closeTop` 錯了 `Esc` 關錯層，繪製錯了畫面上看不到。繪製順序只有讀渲染出來的
  畫面才測得到。
- **F3 別只看 toast**：整個 `closeTop` 與按鍵路由一起看，每一個 popup 都用 `owns()`（開啟中或已開），不用含關閉中的
  `isActive()`（locku 自己 v0.1.3 修過 toast；sshu 照抄清單只修 toast，其實每個 popup 都錯）。新加的 popup 也一樣。
- **global operation 用一份清單**：`globalActions()` 是 global operation popup 的唯一來源；Space menu 的 global 區只接一列，
  不再各自展開（sshu）。
- **key reference 從 Space menu 讀**：panel 的 key reference 由它的 Space menu 列產生（只收按得出來的鍵：單一字元或 `Enter`），
  再接 core key，兩邊不會不一致（sshu）。
- **K3 改成一律送出時，驗證要把「有沒有填」一起問**：以前「沒填完就不會送出」替驗證擋掉了必填檢查（sshu 的 `missing()`）。
  locku 的輸入框都是單一欄位、`Enter` 已經一律送出（見文末「對照過、符合的」），這條目前用不到，新增輸入時記得。
- **模式的按鍵清單只用方向鍵移動**（tdp K11）：模式的鍵跟導覽鍵重疊時，`j`/`k` 是「執行那一列」。locku 目前沒有模式。
- **動手前先把清單互相對一遍，「現況」要拿程式碼核對過才算數**（webu、sshu）：sshu 的清單照抄別的 app，現況寫錯了一半。
- **換註解編號照內容，不照行號**（webu、sshu）：每一處在該檔的註解裡找「現在」那串，數量對得上才換。
- **改完一條就回頭檢查別的 surface 有沒有說錯的提示**（webu）：README 的按鍵表、`ux.md`、`?` 的說明文字。
- **F4 的做法沿用 locku 自己的 `runRow` / `done()`**：dispatch 後若有框握著鍵盤就留住 menu，框完成時整疊清掉。新的 global
  operation popup 也走這條（見第 3 條）。
- **不要順手發版**：家族全部 app 與 tdp 都穩定下來之前，app 不發版（user 2026-09-27）。修完只 commit，CHANGELOG 先記在
  `[Unreleased]`。
- 完整紀錄：terminu repo 的 `.local/family-fix/locku/`、`.local/family-fix/webu/`、`.local/family-fix/sshu/`（本機）；sshu 的
  worked example 在 sshu repo 的 `terminu-fix` 分支。

---

## 1. panel 上的 `?` 是可執行的 `?` menu —— K6、M4、F1

- **現況**：`internal/ui/app.go` `openHelp()`（:425）在 `[1]` 與 profile / saver 的 `[2]` 上組出 `helpMenu`：`global operation`
  標題 + `globalActions()` 的列（`[q]uit`，可執行）、分隔線、`key reference` 標題 + `keyReference` 的唯讀列（`menuItem.ref`）。
  `key()` 的 `helpMenu` 分支（:215）讓它 `j`/`k` 選列、`Enter` 或熱鍵經 `runRow(true, …)` 執行；`runRow` 的 `fromHelp`、`done()`
  清 `helpMenu`、`closeTop` 的 `helpMenu`、`popupDepth`、`View` 的繪製、`AnimTickMsg` 都為它留了位置。`internal/ui/spacemenu.go`
  有 `newHelpMenu()` 與 `menuItem.ref`（只給 `?` menu 用）。
  popup 上的 `?`（`menuHelp`、`optionsHelp`、`confirmHelp`）已經是唯讀的 `helpPopup`，符合；preference、tmux、screen 的 `[2]`
  是欄位字典（偏離，見 dev-remarks，已照 v0.1.4 改寫）。
- **規則**：`?` 打開**最前端那個 surface 的 key reference**：唯讀、可以捲動，沒有游標、不能執行，不是 menu。能做的事在
  `Space` 與 global operation popup，能讀的鍵在 `?`；一個 popup 只屬於一類（F1）。
- **怎麼改**：
  - panel 上的 `?` 改成 `m.help.open(m.layer(), <這個 panel 的 key reference>)`（內容見第 2 條），跟 popup 上的 help 同一個
    `helpPopup`。`?` 再按一次、`Esc` 關掉，`j`/`k` 捲動（`helpPopup.update` 已有）。
  - 拿掉 `helpMenu` 整條：`AppModel.helpMenu`、`newHelpMenu()`、`key()` 裡「`?` 關 `?` menu」與 `helpMenu` 分支、`runRow` 的
    `fromHelp` 參數、`done()` 迴圈裡的 `&m.helpMenu`、`closeTop`、`popupDepth`、`View`、`WindowSizeMsg`、`AnimTickMsg` 裡的
    `helpMenu`；`menuItem.ref` 與 `spaceMenu.view()` 的 `ref` 分支沒人用了一起拿掉。
  - `keyReference` 裡 `?` 那一列（`this menu; on a popup, that popup's keys`）改成 `the keys here; on a popup, that popup's keys`
    之類，不再說「menu」。
  - 既有測試守著舊規則，改寫（改名、反轉斷言）：`TestHelpIsThePanelsGlossaryOnItsDetail`（`[1]` 與 profile 的 `[2]` 上斷言
    `global operation`、`[q]uit`）、`TestHelpIsThePopupsOwn` 後半（`helpMenu.isInteractive()`、`Enter` 在 `?` menu 的 Quit 上離開）、
    `TestBoxOverItsMenu` 最後一段（quit confirm 疊在 `?` menu 上，改成疊在 global operation popup 上，見第 3 條）、
    `TestSplashTakesCtrlC`（:1270 看 `helpMenu`）。新測試：panel 上的 `?` 打開後 `Enter`、`q` 以外的熱鍵都不執行任何東西。
  - 文件：README 兩份的「到處都通」表 `?` 列（`the keys, and quit` / `按鍵與離開`）與 :122 那段的「`?` 列出按鍵」；`ux.md` §A.2
    「`[1]`，以及 profile / saver 的 `[2]`：`?` menu，兩區……」那一條、§A.0 表格的 `?` 列（「可疊在任何浮層上」仍對，但要說它唯讀）。
    §A.0 的 `Space` 列還寫著「在 menu / viewport / message 浮層上 = 關掉它」，是 v0.1.2（locku）修 K5 之前的描述，一起改。

## 2. key reference 沒有列出 panel 自己的鍵 —— K6、M4、D4

- **現況**：`internal/ui/helppopup.go` 的 `keyReference`（:361）只有 core key 與導覽鍵（`Tab · 1-2`、`Enter`、`Esc`、`Space`、`?`、
  `q · Ctrl-C`、`j · k`、`u · d`、`gg · G`），註解寫明「Every other key is a row of a Space menu」。`[1]` 的 `p`、`a`、`D`、`r`、`X`、
  `n` 與 `[2]` 的 `P`、`S`、`R`、`n` 都不在 `?` 裡。寬度由 `helpPopup.layout()` 的 `popupInnerW(m.screenW, keyW+64)` 決定（固定 64 欄給說明）。
- **規則**：panel 上的 key reference **至少列出 core key 與這個 panel 能按的鍵**（M4）；其餘列不列由 app 決定。
  D4（建議）：key reference 的寬度依最長的說明計算。
- **怎麼改**：組 panel 的 key reference 時，先從 `m.actions()` 讀出這個 panel、這個 cursor 能按的鍵（只收按得出來的：單一字元或
  `enter`；disabled 的照列，它只是現在不能執行），每一列 `鍵 → label / hint`，再接 `keyReference` 的 core key 與導覽鍵（sshu 的做法：
  key reference 從 Space menu 讀，兩邊不會不一致）。`P` 在 `[1]` 上不作用（`p` 才是游標那列），不列。寬度照 D4 改成依最長的說明、
  上限仍是螢幕；字典的長說明目前靠換行，照舊。測試：`[1]` 的 profile 列上 `?` 看得到 `[D]`、`[X]`，`[2]` 上看得到 `P`、`S`。
  README 兩份「到處都通」表的 `?` 說明一起改。

## 3. Space menu 的 global 區直接列出 `[q]uit`，沒有 global operation popup —— M2、M4、K9、M3

- **現況**：`internal/ui/app.go` `openMenu()`（:294）把 `globalActions()` 的每一列（目前只有 `[q]uit`）直接放進 Space menu 的
  `global operation` 區；`dispatch()`（`internal/ui/actions.go` :215）同時搜尋 `actions()` 與 `globalActions()`。沒有 global
  operation popup。`regions()`（`spacemenu.go` :39）只剩一區時不加標題（v0.1.1 起 panel 上的 Space menu 一律加標題；目前每個
  panel 都有 item 列，實際上不會只剩一區，但函式與 `ux.md` §A.1 的「只剩一個時不加標題」還是舊規則）。
- **規則**：Space menu 的最後一區 `global operation` **固定一列** `Global operation`，全域動作只有一個時也一樣。`Enter` 打開
  **global operation popup**：一種 menu，疊在 Space menu 上，列出 app 全部全域動作，`j/k` 選、`Enter` 或熱鍵執行；`Esc` 回到
  Space menu（F4）；執行了會關掉整疊的動作照 T1。離開必須在這裡（K9）。panel 上的 Space menu 一律加區塊標題。
  **owner 2026-09-27 裁定：locku 也照這條走 `Space` → `Global operation` → `[q]uit`，即使離開是它唯一的全域動作。**
- **怎麼改**：
  - Space menu 的 global 區換成一列 `Global operation`（沒有熱鍵、不加括號；不要標 `[?]`，`?` 在 Space menu 上是它自己的 help，
    webu 踩過），說明例如 `everything that is about the whole app`。`Enter` 打開新的 global operation popup。
  - global operation popup：再一個 `spaceMenu` 實例（自己的 animator target 與 glyph），列來自 `globalActions()`，標題例如
    `Global operation`。放進 `AppModel`、`WindowSizeMsg`、`AnimTickMsg`、`popupDepth`；**按鍵路由**排在 confirm / options 之後、
    Space menu 之前；**`closeTop`** 排在 options 之後、`menu` 之前；**繪製**在 menu 之後、input / confirm 之前（三處一起改，D3）。
  - 它上面按 `Space` 不作用（K5：`Space` 只關 Space menu，而且只在 Space menu 是最上層時）；`?` 打開它自己的 help（像
    `menuHelp`：`j · k`、`Enter` 執行、方括號裡的字母、`Esc` 回到 Space menu）。
  - 執行它的一列沿用 `runRow` / `done()`：`[q]uit` 沒有未存顏色就離開；有的話 quit confirm 疊在它上面，`Esc` 回到它，接受就離開。
    其他會開框的全域動作（將來若有）開出的框留住整疊，完成時 `done()` 連 global operation popup 一起清掉。
  - `regions()` 一律加標題（拿掉「只剩一區時不加標題」），`openMenu()` 的註解同步。
  - `dispatch()` 仍可搜尋 `globalActions()`（`q` 在 `key()` 早就被攔下，熱鍵路徑不變）；但 Space menu 裡按 `q` 仍是離開（K9），
    跟「global operation popup 裡的 `[q]uit`」是同一個 `quit()`。
  - 既有測試改寫：`TestMenuCoversEveryHotkey` 的「the global %q is not a row」與區數計算（global 區現在是一列 `Global operation`，
    `[q]uit` 要在 global operation popup 的列裡找）、`TestBoxOverItsMenu` 最後一段。新測試：Space → `G`（到最後一列）→ `Enter` 開出
    global operation popup、底下的 Space menu 還在；`Esc` 回到 Space menu；`Space` 在它上面不作用；`?` 是它自己的鍵；`Enter` 在
    `[q]uit` 上離開（有未存顏色時先疊 confirm）。
  - 文件：README 兩份的「到處都通」表 `Space` 列（`the row, the panel, and quitting` / `這一列、這個面板，以及離開`）與 :122 那段
    （`down to quitting` / `包括離開`）；`ux.md` §A.1 第一段（「`global operation` 在每個 Space menu 的最後，跟 `?` menu 同一份清單」
    「只剩一個時不加標題」）、§A.2 的「離開」列加上「在 Space menu 的 `Global operation` 裡」。

## 4. 離開的 confirm 借用共用的 confirm，不一定在最上層 —— K9、F4、F6、D3

- **現況**：`internal/ui/app.go` `quit()`（:502）有未存顏色時用**共用的** `m.confirm.ask(…confirmQuit…)`。它跟其他 popup 的
  上下關係沒有排過，有顏色草稿時實際會出現三種情況（2026-09-27 照程式碼讀出，尚未寫測試重現）：
  - 在 Delete / Activate / Deactivate 的 confirm 上按 `q` 或 `Ctrl-C`：`asksToQuit()` 看的是 `confirm.action`，不是離開，於是
    `confirm.ask` 把使用者正在回答的問題**換掉**；`Esc` 取消離開後，原本的問題也不見了。
  - 在輸入框（rename、new、數字、路徑、PIN……）裡按 `Ctrl-C`：quit confirm 畫在輸入框上面，但 `key()` 的輸入分支（:166）排在
    confirm 之前，**`Enter` 送出的是底下的輸入框**，打的字也進輸入框；看得到的 confirm 拿不到鍵。
  - 在 help（popup 的 `?`、欄位字典）上按 `q`：`q` 在 help 分支之前（:177），quit confirm 開了，但 help 仍然先拿鍵、也畫在最後，
    confirm **被蓋在 help 底下**。
- **規則**：K9 的離開流程在任何 surface 都能進入；進入後使用者看到的那個問題要拿得到 `Enter`（F6 接受、K3），底下正在回答的
  問題原樣留著（F4）。D3（建議）：離開的 confirm 用自己的 popup，疊在整疊最上面；「放在最上層」同時是按鍵路由、`closeTop`、繪製順序。
- **怎麼改**：照 D3 另開一個 quit confirm（例如 `quitAsk confirmPopup`，自己的 animator target），`quit()` 開它、`asksToQuit()`
  看它；`confirmQuit` 從共用 confirm 拿掉。它在按鍵路由上排在 splash、`Ctrl-C` 之後、`Esc` 與其他 popup 之前（`Enter` 離開、
  `?` 開它自己的 help、其他鍵不作用），`closeTop` 第一個關它（toast 之後），繪製在所有 popup 之後（toast 之前）；它自己的 `?` help
  要疊在它上面（D3 的 help 在最上層）。`Ctrl-C` 再按一次立刻離開、`q` 不再疊第二個，照舊。
  測試（每條先 mutation 確認會紅）：有草稿時在 Delete confirm 上按 `q`，`Esc` 後 Delete confirm 還在；在 rename 框裡按 `Ctrl-C` 再按
  `Enter` 是離開、名字沒被改；在 popup 的 `?` help 上按 `q`，quit confirm 畫在最上面、`Enter` 離開。
  若 owner 決定不照 D3 另開 popup（D3 只是建議），至少要讓共用 confirm 在這三種情況下排到最上層，並不再覆蓋底下的問題。

---

## 描述舊規則的 Go 註解

程式改完時一起改；照內容找，不照行號。

| 檔案 | 註解內容 | 改成 |
|---|---|---|
| `internal/ui/app.go` | `helpMenu is the ? menu on a panel (tdp M4): the global operations, run from it, over the key reference.` | 拿掉（欄位拿掉）；新的 global operation popup 欄位註明 tdp M4 |
| `internal/ui/app.go` | `key()` 裡 `? closes the ? menu when that is the top…`、`The ? menu: the global operations, run from here, over the key reference (tdp M4).` | 隨程式拿掉或改寫成 global operation popup |
| `internal/ui/app.go` | `openMenu` 的 `…and no titles when only one is left (tdp M2).` | 標題一律都在；global 區固定一列 |
| `internal/ui/app.go` | `openHelp` 的 `…in place of the ? menu there… On any other panel it is the ? menu (tdp M4).` | panel 上是 key reference（K6、M4），三個 `[2]` 是字典（偏離） |
| `internal/ui/app.go` | `done()` / `runRow` 提到 the menus（Space menu 與 `?` menu） | Space menu 與 global operation popup |
| `internal/ui/spacemenu.go` | `header` 的 `"global operation", "key reference"`、`ref is a row of the ? menu's key reference (tdp M4)`、`regions` 的 `titles only when more than one is left`、`spaceMenu` 的 `a third is the ? menu (tdp M4)` | 隨第 1、3 條改 |
| `internal/ui/actions.go` | `globalActions` 的 `the same rows in the same order in the ? menu, which runs them, and at the foot of every Space menu (tdp M2, M4)` | global operation popup 的唯一來源（M4）；Space menu 只有一列 `Global operation`（M2） |
| `internal/ui/helppopup.go` | `helpPopup is ? where ? is not the ? menu… On any other panel ? is the ? menu`、`keyReference is the ? menu's second region (tdp M4)… Every other key is a row of a Space menu.` | `?` 在每個 surface 都是唯讀的 key reference（K6）；panel 的鍵加 core key（第 2 條） |

---

## 對照過、符合的（2026-09-27）

- **K3**：設定畫面的輸入框都是單一欄位，`Enter` 一律送出這一欄；不合格就不送出、框留著、框標題的尾綴說出原因（` · empty`、
  ` · taken`、` · invalid`、` · absolute or ~/ path`、` · one key…`、` · wrong`）。`Tab` 只在單一欄位上接受灰字提議（K2 允許）。
  PIN 三連問（current → New / Remove → new → confirm）是一串各自單一欄位的輸入框，不是一個多欄位的 input group；兩次不一致時
  toast 說 `PIN mismatch`、回到 new PIN。鎖定畫面的 PIN prompt 同理。
- **K4、F4**：`closeTop` 一次只關最上層，menu 開出的框取消後回到 menu（第 3 條新增的 popup 要照同一個順序插進去）。
- **F3**：`closeTop` 與按鍵路由對每一個 popup 都用 `owns()`。
- **K10**：沒有把按鍵交給子程序的 PTY。custom saver 的程式跑在 pty 上，但按鍵永遠在 locku 手上；設定畫面的 custom 預覽是
  「任何鍵回來」（偏離，見 dev-remarks）。
- **K11**：沒有模式。

## 已定案

- **三個 `[2]` 的字典不接 key reference**（owner 2026-09-27：「以 UX 角度來說，那個使用者最需要看到的是現在的呈現方式」）。
  維持偏離，dev-remarks 已照 v0.1.4 改寫。
- **Space menu 的 `Global operation` 列沒有熱鍵**（owner 2026-09-27）：locku 照 `Space` → `Global operation` → `[q]uit` 走。
