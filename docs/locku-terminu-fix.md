# locku — terminu fix

locku 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.22/principle)（tdp v0.1.22）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `docs/dev-remarks.md` 裡描述該行為的段落。

盤點日期：2026-09-29。依據 `main` 的 `45b8015`（已對齊 v0.1.21，工作區乾淨）。**這一輪只對 v0.1.21 → v0.1.22 的改動**
（`git -C ~/Documents/sideproj/terminu diff v0.1.21 v0.1.22 -- principle/`）：

- D6：`TERMINU__ICON_WIDTH` —— 有 PTY 的 app 開子程序時，在子程序的環境設 `TERMINU__ICON_WIDTH=<自己用的格數>`；每個 app 取 icon
  寬度的順序是 `<APP>__ICON_WIDTH` → `TERMINU__ICON_WIDTH` → 探測（sshu 回報：在別的 app 的 PTY 裡，探測由外層的終端模擬器回答，
  它把 icon 當一格；user 選「家族共用一個變數，所有 app 互通」）。
- D6：疊 popup 時寬、高兩邊都比畫面大也一樣切，不可以把整個框原樣交出去；測試的參考改指 filu 的 `TestD6CompositeDispOversized`。

tdp 連結（README 兩份、docs、`.claude/rules`）已由 terminu session 從 v0.1.21 改成 v0.1.22，只改網址，跟這份清單一起留在工作樹，
還沒 commit。


## 先看

- 清單與改釘的連結先一起 commit，再動程式；commit 只加自己改的路徑。
- 每修一處補 model test、做 mutation；同一個 commit 同步 README 兩份與 dev-remarks；CHANGELOG 記 `[Unreleased]`。
- **修完拿 v0.1.22 全文再逐條對一次**，修完刪掉這份清單。不 push、不發版；把這一輪寫進 terminu `.local/family-fix/locku/README.md`。

## 1. icon 寬度沒有讀 `TERMINU__ICON_WIDTH` —— D6（v0.1.22）

**現況**（`internal/ui/iconwidth_unix.go` `DetectIconWidth()`，`runSettings()` 與 `runLock()` 各一次）：先讀 `LOCKU__ICON_WIDTH`，沒有就探測。locku 跑在別的 app 的 PTY 裡時（例：kbu 的 Alterm、filu 的 shell、sshu 的
格子），探測由外層的終端模擬器回答，icon 一律量成一格。

**規則**：D6（v0.1.22）—— 取 icon 寬度的順序是 `<APP>__ICON_WIDTH` → `TERMINU__ICON_WIDTH` → 探測；有 PTY 的 app 開子程序時，在子程序
的環境設 `TERMINU__ICON_WIDTH=<自己用的格數>`（自己用的格數 = 上面三步得到的那個值，所以巢狀幾層都傳得下去）。

**怎麼改**：`LOCKU__ICON_WIDTH` 沒設時讀 `TERMINU__ICON_WIDTH`（只收 `1`、`2`，其他值當沒設），都沒有才探測；有值時不探測（不送 CPR）。
測試：三種來源的優先順序各一例（兩個都設時 `LOCKU__ICON_WIDTH` 贏；只有 `TERMINU__ICON_WIDTH` 時用它、不探測；都沒有時探測）；
不合法的值被忽略。README 兩份寫 icon 寬度的地方與 dev-remarks 補一句。

## 2. 開 PTY 子程序時沒有設 `TERMINU__ICON_WIDTH` —— D6（v0.1.22）

**現況**：custom saver 在 `internal/custom/custom.go` 用 `pty.StartWithSize(cmd, …)` 開 PTY 跑使用者的程式，`cmd.Env` 沒有加 `TERMINU__ICON_WIDTH`。（`internal/login/login.go` 的 `pty.Start` 是把密碼交給 `su`，不是畫面，不用設。）

**規則**：D6（v0.1.22）—— 取 icon 寬度的順序是 `<APP>__ICON_WIDTH` → `TERMINU__ICON_WIDTH` → 探測；有 PTY 的 app 開子程序時，在子程序
的環境設 `TERMINU__ICON_WIDTH=<自己用的格數>`（自己用的格數 = 上面三步得到的那個值，所以巢狀幾層都傳得下去）。

**怎麼改**：每個開 PTY 的地方，子程序的環境加上 `TERMINU__ICON_WIDTH=<iconCells>`（已經有同名的就覆寫，不重複）。測試：子程序拿到的環境
裡有這個變數、值等於目前的 `iconCells`（1 與 2 各一例）。實機：在這個 app 的 PTY 裡跑另一個家族 app，框線不歪。


## 已經符合、不用修的（對照 v0.1.22 的改動）

- **D6 疊 popup 寬高都大也切**：`45b8015` 已經拿掉特例（見 locku 紀錄「對照 v0.1.21」）。


## 待確認

沒有。
