# locku — terminu fix

locku 尚未符合 [terminu design principle](https://github.com/vulcanshen/terminu/tree/v0.1.0/principle)（tdp v0.1.0）的地方，逐條待修。
修好一條就刪掉一條，並同步 README（兩份）與 `docs/ux.md` 裡描述該行為的段落。有意不修的，改寫成
`dev-remarks.md`「偏離 tdp」的一條並附理由。

盤點日期：2026-09-26（補盤，v0.1.2 之後）。行號以當天的 `main`（`fc1e7fc`）為準。

---

## 1. 從 Space menu 開出的 confirm，取消後回不到 menu —— F4、T1

- **現況**：`internal/ui/app.go` `key()` 在 Space menu 上選一列時，先 `m.menu.close()` 再 `m.dispatch(key)`
  （約 233 行）。所以 Delete profile、Activate / Deactivate integration 這類會開 confirm 的動作
  （`actions.go` 的 `confirm.ask`），在 confirm 上按 `Esc` 取消後，回到的是 panel，不是 Space menu。
  `?` menu 的 global 區（約 199 行，`m.helpMenu.close()`）是同一個寫法。
- **規則**：從 popup A 開出 popup B 時，A 預設留在底下，取消 B 回到 A（F4）。短的 confirm 屬於
  「保留 source」的一類（T1）；只有完成之後 source 已失去意義（長 session、切換 context）才清掉。
- **怎麼改**：選了會開 confirm / input / options 的列時，Space menu 留著、新的 popup 疊在上面（layer + 1）；
  取消回到 menu，**完成**（confirm 接受、input 送出）才把整個 stack 清掉（tdp D3 的家族做法）。
  直接完成的列（不開下一個 popup 的）照舊：執行後 menu 關閉。`?` menu 的 global 區一併處理。
  `ux.md` 描述「選了就關 menu」的段落一起改。
- **細節**（2026-09-26 補）：
  1. 「完成」也包括 options 選定一個值；PIN 那串（current PIN → New / Remove → new PIN → confirm）
     整串做完才清 stack，中途 `Esc` 取消整串、回到 menu。
  2. `key()` 目前先判斷 `?` menu、後判斷 confirm。`?` menu 留在底下時，疊在上面的 Quit confirm
     會被它先吃鍵，所以 confirm、input、options 的判斷要移到 `?` menu 之前。
  3. 預覽照舊關掉 menu：預覽把整個畫面換掉，是 T1 的「切換 context」，回來時 menu 不該還浮著。

## 2. 正在關的 toast 還會吃 `Esc` —— F3

- **現況**：`internal/ui/app.go` `closeTop()` 用 `m.toast.isActive()` 判斷 toast（約 345 行），正在跑關閉
  動畫的 toast 也算。toast 關到一半再按 `Esc`，關閉動畫重來，那個 `Esc` 也傳不到下面那層 popup。
  其他 popup 用的是 `anim.owns()`（開啟中或已開，不含關閉中）。
- **規則**：已經在跑關閉動畫的 popup 不再理會 `Esc`，也不再接收其他按鍵（F3）。
- **怎麼改**：`closeTop()` 的 toast 判斷改成 `m.toast.anim.owns()`，與其他 popup 一致。
