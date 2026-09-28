# locku — terminu fix

locku 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.7/principle)（tdp v0.1.7）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與描述該行為的設計文件段落。有意不修的，改寫成
`dev-remarks.md`「偏離 tdp」的一條並附理由。

盤點日期：2026-09-28（對照 tdp v0.1.7）。v0.1.7 只改了 M2 一條。

---

## 1. Space menu 的 global 列上面還掛著 `global operation` 標題 —— M2

- **現況**：`internal/ui/app.go` `openMenu()` 用 `regions([]string{"item operation", "panel operation", "global operation"}, item, panel, global)`；
  `spacemenu.go` 的 `regions()` 對每一個留下的區都加標題，所以 `Global operation` 那一列上面有 `global operation` 標題。
- **規則**：tdp v0.1.7 M2：Space menu 的 global 那一列**不加區塊標題** —— `global operation` 標題底下只有一列
  `Global operation`，是同一句話講兩次（user 2026-09-28：「一個 global operation 的 section 只有一個 Global operation 的項目」
  太奇怪）。它跟上面的區塊之間照樣用分隔線隔開；item 與 panel 兩區在 panel 的 Space menu 上照舊一律加標題（即使只剩其中一區）。
- **怎麼改**：`regions()` 只組 item / panel 兩區（照舊一律加標題），`openMenu()` 在後面自己接分隔線（前面有東西時）與
  `Global operation` 那一列，不加標題。量 Space menu 形狀的測試改寫；`ux.md` §A.1、README 兩份若有畫出 Space menu 的例子
  一起改。
- 連結：README 兩份、`dev-remarks.md`、`ui.md`、`ux.md` 開頭的 tdp 連結已改釘 `v0.1.7`（未 commit）。

修完不發版：等家族全部 app 與 tdp 都穩定後一起發（CHANGELOG 記在 `[Unreleased]`）。
