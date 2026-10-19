# locku — 還沒符合 tdp 的地方

照 tdp v0.8.0 與 terminu-go v0.1.13 逐條待修（tdp E7）：違反了哪一條、在哪裡、現況、該怎麼改。2026-10-18 重構後照
清單（`checklist/items-zh_TW.md`）對過一遍，locku 自己改得掉的都改了；剩下的三條都要等 lib，已經回報給 terminu（LK 編號）。
全部修完就刪掉這個檔。

## 1. 改 PIN 的三步，上一步沒有留在底下（tdp F1、F4、T1；LK7）

- **違反**：tdp F1「多步驟的流程，每一步是自己的 popup」，例子是兩個疊起來的 popup、F4 保留 source；清單 F1-17、T1-1。
- **在哪裡**：設定畫面 preference › PIN 的 `Enter`：`internal/settings/edit.go` 的 `editPIN` → `newPIN` → `confirmPIN`（`TODO(lib LK7)`）。
- **現況**：每一步是 lib 的 `PasswordInput`，`Check` 通過時 lib 先關掉這一步（`closeTop`），再呼叫 `Done`；下一步開在 `[2]` 上。在
  `confirm PIN` 按 `Esc` 是整個放棄，不是回到 `new PIN`。
- **該怎麼改**：lib 讓 input 在確定之後留著、把下一步疊上去（例如 `Next` 或 `Stay`），整串做完由 app 一起關（`ClosePopup`）。lib 有了就把
  三步改成疊起來的，`Esc` 一次退一步。

## 2. 寫檔失敗時，error popup 沒有疊在值來的那個框上（components/layout/popup「錯誤列」的「動作本身失敗」；LK10）

- **違反**：「動作本身失敗……直接開一個 error popup 蓋在當前的 popup 上；關掉後回到原本的 popup，值都還在」。清單 popup-69。
- **在哪裡**：`internal/settings/settings.go` 的 `save`（`TODO(lib LK10)`）。值都是在 input 的 `Done` 裡寫檔的。
- **現況**：寫檔失敗開的是 error popup，訊息完整（2026-10-18 起，之前是 toast），但那時 lib 已經關掉 input，關掉 error popup 後回到的是
  `[2]`，打的值沒了，要重開框重打。
- **該怎麼改**：lib 讓 `Done` 回報失敗，input 留著、error popup 疊在它上面（跟 1 同一個根源）。

## 3. `[2]` 的捲動位置照列數算，不是照欄位（components/layout/panel「一列一個值」→ dialog/form；LK11）

- **違反**：一列一個值的 panel「照 dialog/form 的規則」：捲動位置以欄位為單位，radio 的幾個選項算同一欄（components/layout/popup
  「內容放不下時」）。清單沒有拆到這一條（popup-56 的適用只寫表單）。
- **在哪裡**：`[2]` 是 lib 的 `Values`；它的 `Scroll()` 數的是每一個 cursor 停得到的地方（radio 的每個選項各算一個）。
- **現況**：59 × 12 的終端機上，runner 的 `[2]` 停在 character 的 `cat` 時寫 `5 of 14`；照欄位算應該是 `4 of 6`。
- **該怎麼改**：lib 的 `Values` 照欄位數；locku 不用改。
