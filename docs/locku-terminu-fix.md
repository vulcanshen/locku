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
