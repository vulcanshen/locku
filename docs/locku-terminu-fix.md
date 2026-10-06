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


## 3. `docs/ui.md` 的舊寫法

- §3.1 input 列：寫 new 的 `name`「目前值當提議」、提議 saver 自己的名字；程式碼是預填（值直接在框裡、可接著編輯），不是
  §3 那種 dim 的提議。
- §1.1 tmux／screen 表 bind-key 列：「含空白或 `#` → ` · one key, e.g. l or C-l`」是錯誤寫在邊框尾綴的舊寫法；程式碼寫在
  錯誤列。


## 這一輪不修

沒有。


## 待確認

沒有。
