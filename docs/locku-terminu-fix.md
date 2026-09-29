# locku — terminu fix

locku 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.21/principle)（tdp v0.1.21）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `docs/dev-remarks.md` 裡描述該行為的段落。

盤點日期：2026-09-29。依據 `main` 的 `4946664`（已對齊 v0.1.20，工作區乾淨）。**這一輪只對 v0.1.20 → v0.1.21 的改動**
（`git -C ~/Documents/sideproj/terminu diff v0.1.20 v0.1.21 -- principle/`）：

- D5：選取文字的模式照 vim 移動 —— `h/j/k/l`、`w/b/e`、`0/$`、`gg/G`、`u/d`（user 要求寫回 tdp，filu 轉達）。
- D6：環境變數命名 `<APP>__<NAME>`（app 名後兩個底線、變數名全大寫單底線分隔），共用名 `<APP>__CONFIG` / `__STATE` / `__DATA`
  / `__CACHE`（都指向**目錄**）/ `__ICON_WIDTH`；給別的程式讀的變數例外；**改名不留舊名**（user 2026-09-29 裁定）。
- D6：疊 popup 時 popup 比畫面寬或高（調整終端機大小的那一格）：起點取 0、超出的部分切掉，不可以 panic（kbu、locku 照搬時抓到，
  filu 的參考實作有這個 bug）。

tdp 連結（README 兩份、docs、`.claude/rules`）已由 terminu session 從 v0.1.20 改成 v0.1.21，只改網址，跟這份清單一起留在工作樹，
還沒 commit。


## 先看

- 清單與改釘的連結先一起 commit，再動程式；commit 只加自己改的路徑。
- 每修一處補 model test、做 mutation；同一個 commit 同步 README 兩份與 dev-remarks；CHANGELOG 記 `[Unreleased]`。
- **環境變數改名是破壞性改動**：CHANGELOG `[Unreleased]` 要寫一條「改名，舊名不再讀」並列出新舊對照；CHANGELOG 裡已發版的舊段落不改。
  `.checkpoints/` 是本機筆記，不用改。改完 `grep -rn '<舊名>'` 除了 CHANGELOG 舊段落應該是零。
- **修完拿 v0.1.21 全文再逐條對一次**，修完刪掉這份清單。不 push、不發版；把這一輪寫進 terminu `.local/family-fix/locku/README.md`。


## 1. 環境變數沒照家族命名 —— D6（v0.1.21）

| 現在 | 意思 | 改成 |
|---|---|---|
| `LOCKU_CONFIG`（`internal/config/config.go`） | 設定目錄 | `LOCKU__CONFIG` |
| `LOCKU_DATA`（`config.go`） | 資料目錄（PIN 重設） | `LOCKU__DATA` |
| `LOCKU_ICON_WIDTH`（`internal/ui/iconwidth_unix.go`） | icon 寬度覆寫 | `LOCKU__ICON_WIDTH` |
| `LOCKU_DUMP`（`internal/ui/dump_test.go`） | 測試用 | `LOCKU__DUMP` |

**規則**：D6（v0.1.21）—— `<大寫 app 名>__<變數名>`，變數名全大寫、單字之間一個底線；app 自己讀的變數（含測試用、傳給自己子程序的）
都照這個寫。共用名：`<APP>__CONFIG`（設定目錄）、`<APP>__STATE`（狀態目錄）、`<APP>__DATA`（資料目錄）、`<APP>__CACHE`（快取目錄）、
`<APP>__ICON_WIDTH`。給別的程式讀的變數例外。**改名不留舊名**（user 裁定）。

**怎麼改**：照上表改名。一起改的地方：**`uninstall.sh`**（它讀 `LOCKU_CONFIG` 找要刪的目錄）、`cmd/locku/pin_test.go` 與 `internal/ui` 各測試檔、
`.local/demos/demo.tape`、README 兩份、`docs/dev-remarks.md`、`docs/ui.md`、`docs/function.md`。改完 `grep -rn 'LOCKU_[A-Z]' .` 除了
CHANGELOG 舊段落是零。


## 2. `compositeDisp()` 比畫面高時只畫中間，跟 D6 的做法不同 —— D6（v0.1.21）

**現況**（`internal/ui/width.go`）：locku 照搬時已經修掉 panic（見 locku 紀錄「對照 v0.1.20」），做法是比畫面高的框照 `overlay.Composite` 只畫中間。

**規則**：D6（v0.1.21）—— 疊 popup 時 popup 可能比畫面寬或高（調整終端機大小的那一格還是舊尺寸）：起點取 0、超出畫面的部分切掉，
**不可以 panic**；測試的邊界要含這種情況。

**怎麼改**：起點取 0、超出畫面的部分切掉（跟 kbu、之後的 filu 一樣），測試照 kbu 的 `TestD6_CompositeDisp`。鎖定畫面的 PIN 框與 custom
saver 的 PIN 框在很小的終端機上會用到這條，改完各在 20×5 這類尺寸印一次看框的上緣。


## 已經符合、不用修的（對照 v0.1.21 的改動）

- **D5 選取模式的移動**：locku 沒有選取文字的模式，不適用。


## 待確認

沒有。
