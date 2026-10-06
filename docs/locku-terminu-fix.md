# locku — terminu fix

locku 發版前要修的 bug，逐條待修。修好一條就刪掉一條。

盤點日期：2026-10-06。依據 `main` 的 `e4f986e`（已對齊 tdp v0.1.23，工作區乾淨）。**這一輪不是 tdp 改版**：是 input 盤點
（2026-09-29，terminu `.local/input-survey/locku.md`）翻出來、跟之後 tdp components 怎麼定無關的 bug；五個 app 修完就發版。
tdp 版本不變，連結不用改。這份清單由 terminu session 寫好留在工作樹，還沒 commit。


## 先看

- 每條先寫一個會紅的測試，再修；CHANGELOG `[Unreleased]` 的 Fixed 各記一條（合併或拆開由 locku 定）。
- 第 1 條是五個 app 共通的做法，五份清單的「做法」同一段文字：畫出來的樣子與行為要一樣，程式怎麼寫各 app 自己定。
- 最後「這一輪不修」列的要等 components 的 input 檔定案，不要先動。
- 修完刪掉這份清單。不 push、不打 tag、不發版：發版是下一步，版號由 user 在 terminu session 一個一個定。把這一輪寫進
  terminu `.local/family-fix/locku/README.md`。


## 1. 單行的值收進換行、Tab 與控制字元 —— 五個 app 共通

**現況**：
- 設定畫面的 `inputPopup`（profile 名稱、數字、config file path、bind-key、command、new／confirm／current PIN）：`update`
  （`inputpopup.go:103-126`）不過濾，貼上的換行原樣進值，值列被斷成兩列。new PIN 貼 `12\n34` 存成 5 個字元，而鎖定畫面的
  prompt 以 `unicode.IsPrint` 過濾、換行打不進去：這個 PIN 永遠解不開，只能 `locku pin reset`。`docs/function.md` §4.2
  寫 PIN 是「任意可列印字元」。
- 鎖定畫面的 PIN prompt：逐字 `IsPrint` 過濾，貼 `x\ny` 得 `xy`，偷偷少了一個字元。

**做法**（五個 app 同一段，2026-10-06 user 定案；之後寫進 tdp components 的 input 檔）：

- 範圍：單行的值 —— 一行文字、路徑、密碼／PIN、搜尋與篩選列。多行編輯框不在這條（只有 webu 的 editor，另有一條）。
  只收數字的欄位照舊只收 `0`–`9`。
- 收進值（打字或貼上）：
  - 換行（`\r\n` 算一個，單獨的 `\n`、`\r` 也各算一個）與 Tab 原樣留在值裡，不換成空白、不刪。
  - 其他控制字元（其餘的 C0、DEL、C1）丟掉。
- 畫：換行畫成 `\n`、Tab 畫成 `\t`，Red `#f38ba8`，佔 2 格，跟手打的 `\`、`n`（一般值的顏色）分得開。量寬、截斷、
  捲動都把它當成一個 2 格寬、不能切開的單位。
- 遮罩的值：照樣遮罩，一個換行或 Tab 也是一顆遮罩符號，不露出 `\n`；使用者靠錯誤列知道。
- 刪：`Backspace` 一次刪掉整個（它本來就是一個字元）。
- 送出：值會被拿去用的 input（送出、存檔、執行、交給別的程式），值裡有換行或 Tab 時 `Enter` 不送出，錯誤列說出哪一欄
  不能有換行或 Tab（英文；句式、大小寫照該 app 現有的錯誤訊息，例：`Name can't have line breaks or tabs`）。其他照 K3：
  多欄表單 focus 跳到第一個不合格的欄位、label 變 Red；有「第一次送出後每鍵重驗」的照舊。
  - 原本送出不會失敗、所以沒預留錯誤列的 input，現在會失敗了，照 F7 打開時就預留錯誤列。
- 搜尋與篩選列（值只拿來找東西，不存、不執行；`Enter` 選的是清單裡的項目）：只照上面畫，不擋。

**為什麼**：單行的值裡換行沒有意義。原樣畫出來會把框畫壞；偷偷換成空白或刪掉，又改了使用者的值而看不出來（user：
「應該轉成 `\n` 或 `\t` 這種明確顯示」）。只在畫面上轉、值裡留原字元，是為了跟手打的 `\n` 分得開，也不會把沒有意義的
字元送出去。

**locku 要改的地方**：
- 收字：`inputPopup.update` 與 `pinPrompt` 的收字都照這條（換行、Tab 留著，其他控制字元丟掉），取代 `IsPrint` 過濾；64 的
  上限照舊。
- 畫：文字框的值（`truncateHead`，`width.go:87`）畫 Red `\n`／`\t`；PIN 框照樣是 `●`。
- 擋：`commitInput()`（`actions.go:767-916`）裡設定畫面的每一個 input；鎖定畫面與 custom saver 的 prompt 也擋，錯誤列寫原因，
  **不比對 bcrypt、不算一次連錯**。
- `command` 現在沒有錯誤列（`docs/ui.md` §3.1：「`command` 送出不會被拒，不留」），照 F7 預留一列；`docs/ui.md` §3.1 與
  `docs/dev-remarks.md`「指令不做任何 sanitize」跟著改。
- custom saver 的 prompt 沒有 renderer，Bubble Tea 不開 bracketed paste：貼上的換行以 `Enter` 進來，跟按 `Enter` 分不出來。
  開得了 bracketed paste 就照這條；開不了就維持現狀，在 `docs/dev-remarks.md` 記一句。


## 2. custom saver 的連錯次數與冷卻，`Esc` 再開就歸零

**現況**：custom saver 每次開 prompt 都是新的 `NewLockPrompt`（`cmd/locku/main.go` 的 `runPrompt`；盤點時在 `:218`、
`:245`），連錯次數與冷卻倒數只存在同一次 prompt 裡：`Esc` 或逾時收起後，下一鍵開的 prompt 從零開始，冷卻可以用 `Esc`
繞過。文件寫的是不會：`docs/function.md:143`「計數只在進程內存活，Esc 回 saver 不重置，冷卻結束才歸零」、`docs/ux.md:170`
「Esc 仍可回 saver，再開 prompt 倒數繼續」。盤點是讀程式碼判斷的，沒有實際跑 custom 鎖。

**怎麼改**：連錯次數與冷卻的結束時間放在整個 custom 鎖定的進程裡，每次開 prompt 帶進去、收起時帶回來，跟內建 saver 一樣。
測試：連錯到冷卻、`Esc`、再開，倒數接著走；冷卻結束才歸零。


## 3. `docs/ui.md` 的舊寫法

- §3.1 input 列：寫 new 的 `name`「目前值當提議」、提議 saver 自己的名字；程式碼是預填（值直接在框裡、可接著編輯），不是
  §3 那種 dim 的提議。
- §1.1 tmux／screen 表 bind-key 列：「含空白或 `#` → ` · one key, e.g. l or C-l`」是錯誤寫在邊框尾綴的舊寫法；程式碼寫在
  錯誤列。


## 這一輪不修

沒有。


## 待確認

沒有。
