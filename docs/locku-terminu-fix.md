# locku — terminu fix

locku 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/main/principle)（tdp v0.1.0）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `docs/ux.md` 裡描述該行為的段落。有意不修的，改寫成
`dev-remarks.md`「偏離 tdp」的一條並附理由。

盤點日期：2026-09-26。行號以當天的 `main` 為準。

---

## 1. `Space` 會關掉 confirm 與 options —— K5、F6

- **現況**：`internal/ui/app.go` 的 `key()`，confirm 開著時 `case " "` 關掉它（等於取消）；options
  開著時 `if k == " "` 也關掉它。
- **規則**：`Space` 只開關 Space menu；其他 popup（由 `Enter` 打開的 confirm、options）上按 `Space`
  不作用。
- **怎麼改**：拿掉這兩處的 `" "` 分支，只留 Space menu（`m.menu`）上的那一個。`ux.md` §5「`Space`
  關掉非輸入浮層」的說法一起改。

## 2. `Ctrl-C` 直接結束，不走離開流程 —— K9

- **現況**：`app.go` `key()` 第一行 `if k == "ctrl+c" { return m, tea.Quit }`，有未存的顏色草稿也直接走。
- **規則**：`Ctrl-C` 與 `q` 做同一件事 —— 進入離開流程（有未存草稿先 confirm）；離開流程進行中
  再按一次 `Ctrl-C` 才立刻離開。
- **怎麼改**：`Ctrl-C` 改成呼叫跟 `q` 同一個離開函式；quit confirm 開著時的 `Ctrl-C` 直接 `tea.Quit`。
  help 裡的 `Ctrl+C  force quit` 改成 `quit (twice: at once)` 之類的說法。鎖定畫布不受影響（偏離 tdp，
  見 dev-remarks）。

## 3. `q` 只在沒有浮層時有效 —— K1、K9

- **現況**：`q` 在 `key()` 的「No float: the globals」段才處理；Space menu、options、confirm、help
  開著時按 `q` 沒有作用。
- **規則**：`q` 是 core key，除了輸入態以外在每個 surface 都是「離開 app」。
- **怎麼改**：`q` 的處理提到浮層路由之前（輸入態之後）。

## 4. `?` 疊在 popup 上時顯示整個 app 的 help —— K6

- **現況**：`helpEntries()` 只看 panel focus；在 Space menu、options、confirm、input 上按 `?`，顯示的
  是 `helpKeys`（整個 app 的按鍵）。
- **規則**：focus 在 popup 上時，`?` 只顯示**這個 popup** 的 help：這個框裡能按什麼、做什麼。
- **怎麼改**：`helpEntries()` 先看最上層的 popup，各給一份自己的 help（menu：`j/k`、`Enter`、熱鍵、
  `Esc`；confirm：`Enter <動詞>`、`Esc`；options：`j/k`、`u/d`、`Enter`、`Esc`；input：`Enter`、`Esc`、
  `Tab` 接受提議、`Backspace`）。

## 5. `?` menu 不能執行、也沒有 global operation 區 —— M4

- **現況**：`helppopup.go` 是唯讀 viewport，內容是 `helpKeys`。
- **規則**：focus 在 panel 上時，`?` 打開的是 `?` menu：
  1. `global operation` 區，可以直接執行（`j/k`、`Enter`、熱鍵），離開 app 必須在這裡；
  2. `key reference` 區，唯讀，至少列出 core key。
- **怎麼改**：`?` menu 改成「上半可執行、下半唯讀」。locku 的 global operation 目前只有 `[q]uit`。
  `helpKeys` 裡按 panel 分組的熱鍵已經都在 Space menu 裡，`key reference` 可以只留 core key 與導覽鍵。
- **要決定**：preference、tmux、screen 的 `[2]` 上，`?` 現在顯示「每一列是什麼意思」的字典
  （`helpPreference`、`helpTool`，2026-09-25 使用者定案）。tdp M4 沒有這一區。兩個選擇：
  (a) 字典當成 `key reference` 之後的第三區，並在 dev-remarks「偏離 tdp」寫明理由；
  (b) 字典搬到別處（例如每一列的 Space menu 說明）。

## 6. Space menu 沒有 global operation 區 —— M2

- **現況**：`app.go` `openMenu()` 只組 `item operation` 與 `panel operation` 兩區。
- **規則**：Space menu 第三區 `global operation`，列出全部全域動作，與 `?` menu 的 global operation
  同一份清單、同一順序。
- **怎麼改**：`openMenu()` 最後接上 `global operation` 區（目前是 `[q]uit`）；只剩一區時不加標題的
  規則照舊。建議把全域動作定義成一份清單，`?` menu 與 Space menu 共用。

## 7. 不能執行的列改寫說明、按了還會 toast —— M6

- **現況**：`internal/ui/actions.go` 對 disabled 的列把 `hint` 換成原因（`cannot delete: last one`、
  `cannot delete: active`、`already active`、`nothing to save`、`nothing changed`、
  `set the config file path first`）；`dispatch()` 遇到 disabled 的列用 toast 顯示那個原因。
- **規則**：列照樣出現、變暗；說明欄維持原本那句，不另外寫原因；`Enter` 與熱鍵都不作用。
- **怎麼改**：disabled 時不改 `hint`；`dispatch()` 遇到 disabled 直接 `return m, nil`。
  `spacemenu.go` 的 disabled 註解（「still answers when pressed」）與 `ux.md` 的對應段落一起改。

## 8. splash 開著時 `Ctrl-C` 直接結束 app —— S3

- **現況**：`internal/ui/app.go` `key()` 先判斷 `ctrl+c` 再判斷 `m.splash.isActive()`，所以 `V` 叫出
  splash 後按 `Ctrl-C`，app 直接結束。
- **規則**：splash 開著時任何鍵都只關掉它，第一次 `Ctrl-C` 也只關 splash。
- **怎麼改**：`splash.isActive()` 的判斷移到 `ctrl+c` 之前（kbu、filu、sshu、webu 都是 splash 先攔）。

## 9. 程式碼註解仍引用 VTP 的 § 編號 —— 文件對齊

- **現況**：`internal/ui` 的註解用 VTP 的 § 編號與「u-family」（不影響行為）。只換引用 VTP 的；
  `lockscreen.go`、`tmux.go`、`config.go` 等處的 `§1.2`、`§4.4`、`§6.2` 指的是 `function.md`，不動。
- **怎麼改**：照 [terminu `vtp/README.md` 的對照表](https://github.com/vulcanshen/terminu/blob/main/vtp/README.md) 換成 tdp 編號：

| 檔案:行 | 現在 | 換成 |
|---|---|---|
| `chrome.go:11` | `kbu §8.4` | `tdp D2` |
| `chrome.go:143` | `§A.1 / §A.2` | `tdp M1` |
| `confirm.go:19` | `§6.1` | `tdp F1` |
| `splash.go:19` | `u-family mark` | `terminu family mark` |
| `spacemenu.go:10` | `§4.2` | `tdp M3` |
| `spacemenu.go:18` | `VTP §A.1.1` | `tdp M2` |
| `spacemenu.go:26` | `the §A.1 contextual entry point` | `the Space menu (tdp K5, M2)` |
| `pinprompt.go:26` | `VTP §2.4` | `tdp D2` |
| `app.go:138` | `§4.3` | `tdp K4` |
| `app.go:153` | `§4.5` | `tdp K8` |
| `app.go:277` | `VTP §A.1.1` | `tdp M2` |
| `actions.go:23` | `§4.2` | `tdp M3` |
| `actions.go:30` | `VTP §A.1.1` | `tdp M2` |
| `width.go:11` | `§1.2` | `tdp L2` |
| `inputpopup.go:35` | `§4.5` | `tdp K8` |
| `inputpopup.go:97` | `§4.3` | `tdp K1` |
| `popup.go:12` | `VTP §2.2 / §6.3` | `tdp D2` |
| `popup.go:14` | `§B` | `tdp P4` |
| `popup.go:31` | `§6.2 … inside the 100-200ms band` | `tdp F2 … the family default (tdp D3)` |
| `popup.go:71` | `§6.2` | `tdp F2` |
| `popup.go:151` | `§4.5` | `tdp K8` |
| `popup.go:208`、`:233` | `§4.4` | `tdp M5` |
| `popup.go:237` | `§A.0` | `tdp P2` |
| `toast.go:26` | `§6.5` | `tdp F3` |
| `theme.go:6` | `VTP §B` | `tdp P4` |
| `theme.go:12` | `§4.4` | `tdp M5` |
| `theme.go:26` | `§2.4` | `tdp D2` |
| `helppopup.go:8` | `the §A.2 non-contextual entry point` | `the ? entry point (tdp K6, M4)` |
| `helppopup.go:44` | `§A.0.K` | `tdp K1` |
