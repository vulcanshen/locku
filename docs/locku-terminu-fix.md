# locku — terminu fix

locku 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.17/principle)（tdp v0.1.17）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `docs/` 裡描述該行為的段落。有意不修的，改寫成 `dev-remarks.md`「偏離 tdp」的一條並附理由。

> **v0.1.17（2026-09-29，這份清單寫完後才出）**：只補了 M5 三點，已照它調整本清單 —— menu 與 key reference **說明欄**裡提到的鍵
> 算句子，加方括號；**README** 內文的鍵用 Markdown code 標（`` `Enter` ``），不加方括號，鍵名與寫法照 M5；**別的工具自己的按鍵**
> （tmux 的 `prefix l`、`C-a x`）照那個工具的寫法。其餘條目照 v0.1.14–v0.1.16。

盤點日期：2026-09-29（對照 tdp v0.1.16）。以 `main` 的 `634a607` 為準（已對齊 v0.1.13，工作區乾淨）。這一輪**只對 v0.1.13 → v0.1.16
的改動**（`git -C terminu diff v0.1.13 v0.1.16 -- principle/` 與 terminu CHANGELOG 的 v0.1.14–v0.1.16 段）：

- v0.1.14：F1 / F8 / K11 的 toast、K10 / D5 的 PTY 出口鍵、術語「模式」（zoom 不是模式）、M5 鍵的寫法涵蓋 key reference 與 README、
  M6 的 key reference 變暗。
- v0.1.15：M5 鍵的寫法全部定案 —— 鍵帽上的名字、大駝峰、不自創縮寫；依位置寫：label 用括號標記、句子裡的鍵加方括號、hint 與
  footer 寫 `鍵:說明`、key reference 兩欄；D1–D5 的例子與鍵的顏色跟著改。
- v0.1.16：D5 `Alt-Esc` 一律先 confirm；M6 補「說明別的 surface 的段照亮」。

每一條照內容對程式碼核對過，位置寫檔案與函式，不寫行號；key reference 的顏色是在 scratch 複本裡開 TrueColor render 量的（沒有動
locku 的工作樹）。

要修的五條：hint 與 footer 改成 `鍵:說明`（第 1 條）、key reference 的鍵改 Blue 並用 `/` `–`（第 2 條）、key reference 把現在不能按的鍵
變暗（第 3 條，M6）、句子裡的鍵加方括號（第 4 條）、README 兩份與 docs 的鍵名（第 5 條）。第 1、2、4、5 條都是 M5（v0.1.15）。toast、
PTY 出口鍵、模式與 zoom 都已經符合或不適用，見最後。

指向 tdp 的五處連結（兩份 README、`docs/ux.md`、`docs/ui.md`、`docs/dev-remarks.md` 開頭）已經改釘 `tree/v0.1.17/principle`，在工作樹裡、
未 commit。


## 先看

- **清單先 commit，再動程式。** 這份清單與改釘的五處連結一起先 commit（v0.1.7、v0.1.12 那兩輪也是這樣）；待確認已在
  terminu session 定案。程式的 commit 跟清單分開，一條一個 commit，commit 只加自己改的路徑。
- **照編號做。** 第 1 條（hint、footer）與第 2 條（key reference 的鍵色）先做，第 3 條的測試才能直接量最後的顏色（亮的鍵是 Blue）；
  第 4、5 條是字串與文件，最後做。
- **每修一處補 model test，逐處 mutation**：把修正單獨改回舊行為，確認對應的測試會紅。斷言「舊色不在」，不只「有暗色」—— key reference
  的區標題本來就是 `dimColor`，只量「有暗色」會被它騙過（v0.1.9 那輪鎖定畫布的教訓）。測顏色先呼叫 `colours(t)`（`dim_test.go`）。
- **畫面字串改了，先找斷言舊字串的測試**：`rg -n "space menu|Bksp| · |Esc close|Press Esc" internal/ui/*_test.go e2e/`。目前沒有測試
  斷言舊的 hint、footer 或句子（`dump_test.go` 的 `space menu` 只是標籤；`TestHelpIsThePopupsOwn` 斷言的 `Space menu`、`Confirm`、
  `cancel`、`Choose one` 不受影響），所以新測試要把新寫法釘住；`TestViewFitsTheTerminal`（每一列等於終端機寬）守 footer 的寬度計算。
  改完 `LOCKU_DUMP=1 go test ./internal/ui -run TestDump -v` 把畫面印出來看一次。
- **同一個 commit 同步 README 兩份與 `docs/`（`ux.md`、`ui.md`、`dev-remarks.md`），CHANGELOG 記 `[Unreleased]`**；跟上一個發佈版比，
  已經有的條目就併進去，不另起一條。
- **修完拿 v0.1.17 全文再逐條對一次**，不只這五條與 CHANGELOG。
- **不 push、不發版。** 家族全部 app 與 tdp 都穩定之前不發 release。
- **把這一輪寫進 terminu repo 的 `.local/family-fix/locku/README.md`**（本機、不進版控）：開頭的輪次清單加「對照 v0.1.14–v0.1.16」一行，
  「修了什麼」、「已經符合」各加一節，「給下一個 app 的經驗」與「發布」補上這一輪。


## 1. hint 與 footer 改成 `鍵:說明` —— M5（v0.1.15）、D1、D2

**現況**：

- `internal/ui/popup.go` `hintLegend()`（每個 popup 的下框 hint）：每組是「鍵、一個空格、說明」，組與組之間兩個空格；鍵 `focusColor`
  （Blue `#89b4fa`）、說明 `dimColor`（Overlay0 `#6c7086`）。顏色已經對，缺冒號、間隔不對。
- `internal/ui/chrome.go` `keyLegend()`（footer）：同樣是「鍵、空格、說明」，組間三個空格，開頭一格；顏色同上；`plainW` 照這個間隔算寬，
  太窄時從右邊整組丟。
- 各處畫出來的字：

| 位置 | 產生的地方 | 現在 | 改成 |
|---|---|---|---|
| footer | `app.go` `View()` → `keyLegend()` | `space menu   ? help   tab/1-2 panels   q quit` | `Space:menu ?:help Tab/1–2:panels q:quit` |
| Space menu、global operation popup、options | `spacemenu.go` `view()` | `j/k move  Enter run  Esc close`；沒有列時 `Esc close` | `j/k:move Enter:run Esc:close`；`Esc:close` |
| confirm（Delete、activate、deactivate）與離開的 confirm | `confirm.go` `view()` | `Enter delete  Esc cancel`、`Enter quit anyway  Esc cancel`…… | `Enter:delete Esc:cancel`、`Enter:quit anyway Esc:cancel`…… |
| input（name、number、path、key、command） | `inputpopup.go` `view()` | `Enter save  Esc cancel`（`create`、`rename` 同）；有提議時 `Enter save  Tab edit it  Bksp clear  Esc cancel` | `Enter:save Esc:cancel`；`Enter:save Tab:edit it Backspace:clear Esc:cancel` |
| 設定畫面的 PIN 框（current、new、confirm） | `inputpopup.go` `view()`（masked） | `Enter next  Esc cancel` | `Enter:next Esc:cancel` |
| 鎖定畫面的 PIN prompt | `pinprompt.go` `view()` | `Enter unlock  Esc back`；冷卻中 `Esc back` | `Enter:unlock Esc:back`；`Esc:back` |
| key reference 與字典 | `helppopup.go` `view()` | `j/k scroll  Esc close`（捲得動才有 `j/k`） | `j/k:scroll Esc:close` |
| toast | `toast.go` `view()` | `Esc close` | `Esc:close` |

**規則**：M5（v0.1.15）—— hint、footer 寫 `鍵:說明`，冒號前後不空格，項目之間一個空格；鍵名用鍵帽上的名字、大駝峰、不自創縮寫
（`Space`、`Tab`、`Backspace`）；範圍用 `–`。D1：footer 固定為 `Space:menu ?:help Tab/1–N:panels q:quit`。D2：hint、footer 的鍵 Blue，
冒號與說明 Overlay0。D3：confirm `Enter:<動詞> Esc:cancel`。D4：menu `j/k:move Enter:run Esc:close`。

**怎麼改**：

- `hintLegend()`：每組畫成「鍵（`focusColor`）＋ `:說明`（`dimColor`，冒號跟說明同色）」，組間一個空格。框線上字的前後各留一格照舊
  （那是字與框線的距離，不是項目間隔）。
- `keyLegend()`：同樣的寫法；`plainW` 改成「鍵寬 + 1 + 說明寬」、組間 1。開頭那一格留白要不要留，由 app 決定。
- footer 的 pairs 改成 `Space`、`?`、`Tab/1–2`、`q`（`–` 是 U+2013 en dash）；input 的 `Bksp` 改 `Backspace`。
- 測試：`ansi.Strip` 後 footer 那一列（去掉頭尾空白）是 `Space:menu ?:help Tab/1–2:panels q:quit`；Space menu 的下框有
  `j/k:move Enter:run Esc:close`；有提議的 `path` 框有 `Backspace:clear`、沒有 `Bksp`；`colours(t)` 後 footer 的 `Space` 是 `focusColor`、
  `:menu` 是 `dimColor`（冒號不是 Blue）。mutation：組間改回兩格、拿掉冒號、`Bksp` 改回、冒號畫成鍵的顏色，各自要紅。`plainW` 算錯會讓
  footer 那一列不等於終端機寬，由 `TestViewFitsTheTerminal` 抓；但新的 footer 含開頭留白正好 40 欄，那個測試最窄的 40 × 12 不會丟組 ——
  要量「從右邊整組丟」得另外用更窄的寬度（例：30 欄只剩 `Space:menu ?:help`）。
- 同一個 commit 的文件：
  - 兩份 README「The settings screen」／「設定畫面」一節畫面示意的最後一列（footer）。
  - `docs/ui.md`：§1.1 畫面示意的 footer 列、§3.2 PIN prompt 示意下框的 `enter unlock · esc back`（→ `Enter:unlock Esc:back`）、§5 Chrome
    表的 footer 列、§4 色帶表 Blue 那一列補「hint、footer、key reference 裡的鍵」、Overlay0 那一列的「hint」改成「hint 與 footer 的冒號與說明」。
  - `docs/ux.md` §A.0 揭露對照表的「footer 常駐 `space menu`」「footer 常駐 `? help`」→ `Space:menu`、`?:help`。
  - demo gif 見「待確認」第 2 題。
- CHANGELOG `[Unreleased]` 新增一條 Changed，例：The footer and the hints on the popups write each key as `Key:what`, one space apart —
  `Space:menu ?:help Tab/1–2:panels q:quit` — and name keys as the keycap does: `Backspace`, not `Bksp`。第 2、4 條可以併進這一條。


## 2. key reference 的鍵：顏色與多鍵的寫法 —— M5（v0.1.15）、D2

**現況**：

- `internal/ui/helppopup.go` `helpPopup.layout()`：鍵欄 `handColor`（Subtext1 `#bac2de`），說明 `textColor`（Text `#cdd6f4`），區標題
  `dimColor`。實測（scratch、TrueColor）：`[1]` 的 `?` 裡每一列的鍵是 `186;194;222`、說明 `205;214;243`。
- 鍵欄的字：
  - `keyReference`（`everywhere` 段）：`Tab · 1-2`、`q · Ctrl-C`、`j · k`、`u · d`、`gg · G`；`Enter`、`Esc`、`Space`、`?` 已經對。
  - `menuHelp`：`j · k`、`u · d`、`gg · G`、`Space · Esc`；`globalMenuHelp`：`j · k`；`optionsHelp`：`j · k`、`u · d`、`gg · G`；
    `confirmHelp` 的 `Enter`、`Esc` 已經對。
  - `panelKeys()` 從 `actions()` 來的鍵（`Enter`、`a`、`p`、`D`、`r`、`X`、`n`、`S`、`R`、`P`）已經是鍵帽名與實際大小寫。

**規則**：M5（v0.1.15）—— key reference 兩欄，鍵不加括號、不加冒號；幾個鍵做同一件事用 `/`，範圍用 `–`。D2：key reference 的鍵
Blue `#89b4fa`、說明 Text `#cdd6f4`。

**怎麼改**：

- 鍵的樣式改 `focusColor`；說明維持 `textColor`。
- 鍵欄改成 `Tab/1–2`、`q/Ctrl-C`、`j/k`、`u/d`、`gg/G`、`Space/Esc`；說明不用改。`menuHelp`、`globalMenuHelp` 的 `[x]` 那一列照留
  （見「已經符合」）。
- 字典（preference、tmux、screen 的 `[2]` 上的 `?`，偏離）用同一個 `layout()`，但左欄是設定名稱（`PIN`、`activate`、`bind-key`……），
  不是鍵。建議左欄維持現在的顏色、不跟著變 Blue —— key reference 裡 Blue 的意思是「這是鍵」。做法由 app 決定（例：`helpPopup` 記一個
  「這是字典」，或 entry 自帶樣式）。
- 測試：`colours(t)` 後在 `[1]` 開 `?`：`Esc` 那一列有 `focusColor`、沒有 `handColor`，說明是 `textColor`；`ansi.Strip` 後有 `q/Ctrl-C`、
  `Tab/1–2`，沒有 `q · Ctrl-C`；Space menu 的 `?` 有 `Space/Esc`。照上面的建議做的話，字典的 `activate` 那一列沒有 `focusColor`。
  mutation：鍵色改回 `handColor`、`/` 改回 ` · `，各自要紅。
- 文件：`docs/` 沒有寫 key reference 鍵欄的顏色與分隔（§4 色帶的 Blue 在第 1 條補）；README 的按鍵表見第 5 條。CHANGELOG 併進第 1 條
  那一條。


## 3. `?` 的 key reference 沒把現在不能按的鍵變暗 —— M6（v0.1.14）

**現況**：

- `internal/ui/app.go` `panelKeys()`：從 `actions()` 讀這個 panel、這個游標的每一個動作，`disabled` 的也列（這一半是對的），但放進
  `helpEntry` 時沒有帶 `disabled`。`internal/ui/helppopup.go` 的 `helpEntry` 只有 `key`、`desc` 兩欄，`helpPopup.layout()` 對每一列都用
  同一組亮色。
- 實測（scratch 複本、TrueColor、`newTestApp`）：`[1]` 游標在啟用中的 `clock` 上，Space menu 的 `[a]ctivate`、`[X] Delete` 整列是
  `dimColor`（`108;112;134`），`?` 裡的 `a  Activate — …`、`X  Delete — …` 卻跟 `p`、`D`、`r` 一樣亮。profile 的 `[2]` 沒有草稿時，
  `S  Save — …`、`R  Reset — …` 也一樣亮。
- 會碰到的地方：`[1]` 的 profile 列（`a`：已經是啟用中的；`X`：最後一個或啟用中的），profile 與 saver 的 `[2]`（`S`、`R`：沒有草稿時）。
  tool 的 `[2]` 的 `[Enter] Activate`（路徑沒填時 disabled）不在範圍內：那裡的 `?` 是字典（偏離），不是 key reference。

**規則**：M6（v0.1.14 新增）—— `?` 的 key reference 跟 menu 同一套：對象存在、現在不能按的鍵照樣列出、**變暗**；對象不存在就不列。

**怎麼改**：

- `helpEntry` 加一個 `disabled`（或等價的做法），`panelKeys()` 把 `a.disabled` 帶進去；`helpPopup.layout()` 對 disabled 的列，鍵與說明
  都用 Space menu disabled 列的同一個 `dimColor`（`spaceMenu.view()` 裡的 `dim`），折行的續行也一樣。說明維持原句，不另寫原因。
- 其他 key reference 不用動：`everywhere`（`keyReference`）與各 popup 自己的（`menuHelp`、`globalMenuHelp`、`optionsHelp`、
  `confirmHelp`）沒有有條件的鍵；「對象不存在就不列」已經成立（見「已經符合」）。
- 測試（`app_test.go`，放在 `TestHelpIsThePopupsOwn` 旁邊），`colours(t)` 之後：
  ① `[1]` 在 `clock`（啟用中）開 `?`：`Activate`、`Delete` 兩列有 `dimColor`，沒有 `focusColor`（第 2 條之後亮的鍵色）、`handColor`、
  `textColor`；`Preview`、`Duplicate`、`Rename` 三列相反。② `j` 到 `clock2` 再開：`Activate`、`Delete` 是亮的。③ profile 的 `[2]` 沒有
  草稿時開 `?`：`Save`、`Reset` 暗；改一個顏色 channel（照 `TestColourDraftSaveReset` 的做法）之後再開：亮。找列用夠特定的字串
  （`line(m.help.view(), "Delete — this profile")`），先把畫面印出來看。mutation 兩處：`layout()` 不看 disabled、`panelKeys()` 不帶
  disabled，都要紅。
- 文件：`docs/ux.md` §A.2 裡「`[1]`，以及 profile / saver 的 `[2]`」那一點，「dimmed 的也列，它只是現在不能執行」改成「照樣列出、變暗，
  跟 Space menu 一樣」（標 2026-09-29、tdp v0.1.14 M6）；§B 元素專職化表 dim 那一列補「key reference 裡現在不能按的鍵」；`docs/ui.md`
  §3.1 浮層表 `?` help 那一列補一句。README 兩份沒有寫到明暗（Space menu 的變暗也沒寫），不用動；dev-remarks 沒有描述 key reference 的
  內容，不用動。
- CHANGELOG：併進 `[Unreleased]` 的第一條（`` `?` lists the keys, to read: … every row of its Space menu … ``），例如在 every row of its
  Space menu 後面接一句「one that cannot run now dimmed, as it is there」，不另起一條。


## 4. 句子裡的鍵沒加方括號 —— M5（v0.1.15）

**現況**：

| 位置 | 產生的地方 | 現在 | 改成 |
|---|---|---|---|
| 離開的 confirm（有未存的顏色時）第二行 | `app.go` `quit()` | `S on the profile saves them, R drops them` | `[S] on the profile saves them, [R] drops them` |
| `config file path` 框的提示句 | `actions.go` `editPath()` | `config file path — the file tmux's block goes into; Backspace then Enter to unset` | `…; [Backspace] then [Enter] to unset` |
| tmux、screen 字典的 `activate` 那一項 | `helppopup.go` `helpTool()` | `… off: it is not. Enter turns it, after a confirm` | `… [Enter] turns it, after a confirm` |
| splash 最下面一行 | `splash.go` `render()` | `Press Esc to close` | `Press [Esc] to close`（家族四個 app 的 splash 都是這一句） |

**規則**：M5（v0.1.15）—— 句子（空狀態、toast、錯誤訊息）裡的鍵一律加方括號：`Press [A] or [Space]`。confirm 的內容、輸入框的提示句、
字典的說明、splash 的提示都是句子。

**怎麼改**：

- 照表改字。
- 測試：`ansi.Strip` 後，離開的 confirm 有 `[S] on the profile`、`path` 框有 `[Backspace] then [Enter]`、tmux 的字典有 `[Enter] turns it`、
  splash 在提示出現之後（`splash_test.go` 的做法）有 `Press [Esc] to close`；mutation：各改回一處要紅。
- 文件：README 與 `docs/` 沒有引用這幾句，不用動。CHANGELOG 可以併進第 1 條那一條（a key named in a sentence is in brackets），或不記。
- 其餘句子不用改的見「已經符合」；tmux / screen 自己的鍵（`C-a x`、`prefix l`）見「待確認」第 1 題。


## 5. README 兩份與 docs 的鍵名 —— M5（v0.1.15）

**現況 → 改成**：

| 檔案、位置 | 現在 | 改成 |
|---|---|---|
| `README.md`「The lock screen」一節（不是按鍵表底下的同名小節） | `Ctrl+C, Ctrl+Z and Ctrl+\ are just keys` | `` `Ctrl-C`, `Ctrl-Z` and `Ctrl-\` are just keys `` |
| `README.md`「Limits」一節 | `(Alt+F1 … F7)` | `(Alt-F1–F7)` |
| `README-zh_TW.md`「鎖定畫面」一節 | `Ctrl+C、Ctrl+Z、Ctrl+\ 只是按鍵` | `` `Ctrl-C`、`Ctrl-Z`、`Ctrl-\` 只是按鍵 `` |
| `README-zh_TW.md`「限制」一節 | `（Alt+F1 … F7）` | `（Alt-F1–F7）` |
| 兩份「Key bindings」／「按鍵」的 Everywhere／到處都通表 | `` `q` · `Ctrl-C` `` | `` `q` / `Ctrl-C` ``（做同一件事，跟同表的 `` `j` / `k` `` 一樣用 `/`） |
| 同一張表 | `` `Tab` · `1` · `2` `` | `` `Tab` / `1`–`2` ``（跟 footer 的 `Tab/1–2` 一致） |
| `docs/ux.md`：§A.2 全域動作表「離開」那一列與表下那段、§5「離開的 confirm 是自己的浮層」那段、附錄「Core key」那一行 | 九個 `Ctrl+C` | `Ctrl-C` |
| `docs/function.md` §0.2 | `screen 的 Ctrl+a` | `Ctrl-A`（`Ctrl` 後面的字母大寫） |
| `docs/function.md` §0.3、§2.3 | `Alt+F1 到 F7` | `Alt-F1–F7` |
| `docs/function.md` §2.1、§12 驗收 | `Ctrl+C、Ctrl+Z、Ctrl+\` | `Ctrl-C`、`Ctrl-Z`、`Ctrl-\` |
| `docs/function.md` §2.2 | 訊號表的 `Ctrl+Z`、表下「Ctrl+C 會以 KeyMsg 進來」 | `Ctrl-Z`、`Ctrl-C` |
| `docs/function.md` §2.3 | `Cmd+W` | `Cmd-W` |
| `docs/dev-remarks.md`「建置與開發」 | `Ctrl+C 被吞` | `Ctrl-C` |

**規則**：M5（v0.1.15）—— 鍵名在畫面上所有地方與 README 都一樣：modifier 用 `-` 連接，`Ctrl` 後面的字母大寫；幾個鍵做同一件事用 `/`；
範圍用 `–`。

**怎麼改**：

- 照表改。`Alt-F1–F7` 與 `Cmd-W` 不是 locku 的鍵（Linux 主控台、終端機模擬器的），但 README 裡鍵的寫法只有一種，一起改。
  CHANGELOG.md 裡已經都是 `Ctrl-C`。
- README 的 footer 示意與 docs 裡描述 hint、footer 的地方，跟第 1 條同一個 commit，不在這裡。
- 純文件：不記 CHANGELOG、沒有 model test；驗收是 `rg -n "(Ctrl|Alt|Shift|Cmd)\+|Ctrl-[a-z]" README.md README-zh_TW.md docs/` 零筆。
- tmux 的 `prefix+d`、`prefix+c`（dev-remarks）與 README 裡的 `prefix l`、`C-a x` 見「待確認」第 1 題。


## 已經符合、不用修的（對照 v0.1.13 → v0.1.16 的改動）

- **F1、F8：toast 除了 `Esc` 不收任何鍵、不觸發 dim**：`internal/ui/app.go` `key()` 裡 toast 只出現在 `closeTop()` 的第一個 case
  （`m.toast.anim.owns()`）；其他鍵照沒有 toast 時的路由走 —— `Space`、`?`、`q`、`Tab`、字母、導覽鍵都穿過它（`toast.go` 的註解也這樣寫）。
  `View()` 的 `floats` 不含 toast，它在 dim 之後才疊上去：不觸發 dim，自己也不 dim。正在關的 toast 不吃 `Esc`（`owns()`，
  `TestEscPassesAClosingToast` 守著）。鎖定畫面沒有 toast。「其他鍵穿過」目前沒有直接的測試，要補可以順手（toast 開著時 `j` 移動、
  `Space` 開 menu），不是違反。
- **K11：模式裡回應 `Tab` 的 toast 還在時，第一個 `Esc` 先收它**：locku 沒有模式（v0.1.4、v0.1.10 已判），不適用；toast 先吃 `Esc`
  本來就是上一條的 `closeTop()` 順序。
- **K10、D5：家族的 PTY 出口鍵是 `Alt-Esc`（v0.1.14）；`Alt-Esc` 讓 focus 離開 PTY 或結束子程序時一律先 confirm（v0.1.16）**：不適用，
  locku 沒有把鍵交給子程序的 surface。
  - custom saver 跑在 locku 的 pty 上，但鍵永遠在 locku 手上（`internal/custom` 的套件說明）：鎖上的任何鍵叫出 PIN 框，預覽上的任何鍵
    結束預覽（`Terminal.Key()`、`Preview()`），都由 locku 讀，不送進程式。預覽「任何鍵回來、程式被收掉」是既有的偏離（預覽任何鍵就回來）；
    D5 擔心的「兩次 `Esc` 黏成 `Alt-Esc`」不會發生 —— 鍵從來不到程式那裡。
  - `internal/login` 的 `Verify()` 在 pty 上跑 `su`，密碼由 locku 寫進去，使用者不在那個 pty 裡打字；`locku pin reset` 是 CLI，用
    `term.ReadPassword()` 讀密碼。
  - 設定畫面沒有綁 `Alt-Esc`（它的 `msg.String()` 是 `alt+esc`、不是 `esc`，落到 `dispatch()` 什麼都不做）。
- **術語「模式」：版面的切換（zoom）不是模式**：locku 沒有 zoom，也沒有切版面的鍵。終端機窄於 60 欄時只畫 focus 的那個 panel
  （`app.go` `View()`、`chrome.go` 的 `narrowW`），是跟著寬度自動的，`Tab` / `1` / `2` 照常換 panel，`Esc` 不碰它。docs 裡的「無 PIN 模式」
  是設定狀態（沒設 PIN），不是 tdp 的模式；鎖定畫布本身是偏離。
- **M5 label**：`bracketHotkey()` 的括號標記照舊（`[a]ctivate`、`[X] Delete`、`[Enter] Edit`、`[q]uit`），panel 標題 `[1] locku`、`[2] …`。
- **M5 key reference 的 `[x]`**：`menuHelp`、`globalMenuHelp` 的 `{"[x]", "the letter in a row's brackets runs it"}` 說明的是「列裡括號的字母
  就是熱鍵」這個 label 記號本身，不是拿括號標鍵，不算違反。
- **M5 key reference 的其他部分**：兩欄、鍵不加冒號；區標題 `[1] locku`、`[2] …` 與說明裡的 `[1]`、`[2]` 是 panel 標題（label），也正好是
  按了會到那個 panel 的鍵；`Ctrl-C` 的字母已經大寫；說明已經是 `textColor`（Text）。
- **M5 hint 與 footer 的顏色**：鍵已經是 Blue（`focusColor`）、說明已經是 Overlay0（`dimColor`）；第 1 條只加冒號（跟說明同色）與改間隔。
- **M5 其他的句子**：toast 訊息（`PIN set`、`PIN removed`、`write failed: …`、`config.yaml ignored: …`）沒有 locku 的鍵；錯誤列
  （`name is taken`、`a whole number, 0 or more`、`an absolute or ~/ path`、`wrong PIN`、`try again in N s`）沒有鍵，`one key, e.g. l or C-l`
  與 `… or ^L` 是使用者要照 tmux / screen 語法填的值的例子，不是要按的鍵；鎖定畫布狀態列的 `no PIN · any key unlocks` 沒有點名鍵；locku 沒有
  空狀態（`[1]` 永遠有列，Space menu 永遠有 `Global operation`）。`locku pin reset` 的 `[y/N]` 是 CLI，不在 tdp 範圍。
- **M6：hint 與 footer 可以只列現在按得了的**：input 的 `Tab`、`Backspace` 只在有提議時出現，help 的 `j/k` 只在捲得動時出現，PIN 框冷卻中
  只剩 `Esc`；footer 的四組在 panel 上永遠按得了。
- **M6：對象不存在就不列**：custom profile 沒有顏色，`actions()` 不給 `S` / `R`，key reference 與 Space menu 都沒有；`[1]` 上 tool 與
  preference 的列只有 `Enter`。popup 自己的 key reference 沒有有條件的鍵（Space menu「變暗的列不能執行」寫在 `Enter` 那一列的說明裡）。
- **M6（v0.1.16）：說明別的 surface 的段照亮**：locku 的 key reference 沒有這種段。`everywhere` 是這個 panel 上也按得了的 core key 與導覽鍵，
  本來就照亮；字典是偏離，不是 key reference。


## 待確認

沒有。上一版兩題 user 2026-09-29 裁定：

1. **tmux / screen 自己的鍵照工具的寫法**（v0.1.17 寫進 M5）：`C-a x`、`prefix l`、`prefix :`、`bind-key` 的值都不動；dev-remarks 的
   `prefix+d`、`prefix+c` 改成 `prefix d`、`prefix c`，免得 `+` 被讀成 modifier。
2. **demo gif 不在這一輪重錄**：家族全部對齊、發版前一起用 `make gif` 錄。
