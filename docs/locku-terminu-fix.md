# locku — 還沒符合 tdp 的地方

照 tdp v0.9.0 與 terminu-go v0.4.0（還沒打 tag，先用 `go.work`）逐條待修（tdp E7）：違反了哪一條、在哪裡、現況、該怎麼改。
2026-10-20 升級時對過 v0.9.0 新增與改寫的清單條目，剩下的都要等 user；已經回報給 terminu（LK 編號）。全部修完就刪掉這個檔。

## 1. profiles 放在設定裡，tdp v0.9.0 可能把它算成資料（tdp E3；LK19）

- **違反**：tdp E3「東西放哪一類」：設定只放 app 自己啟動與運作用的選項；「使用者寫給 app 用的清單也是資料（例：sshu 的 hosts 與
  credentials、hostu 的 profiles）」。清單 E3-13。
- **在哪裡**：`internal/config` 的 `Config.Profiles`，寫在 `~/.config/locku/config.yaml` 的 `profiles:`。
- **現況**：profiles 是使用者命名、調好的幾組 saver 設定，跟 `profile:`（用哪一組）、`savers:`（新 profile 的預設值）一起在 config.yaml。
  locku 的 profile 算不算「使用者寫給 app 用的清單」，terminu 不確定，2026-10-20 回報成 LK19，等 user 定；之前照現在的位置讀寫。
- **該怎麼改**：user 定 A（算資料）時，搬到 `~/.locku/data/profiles.yaml`：設定畫面第一次啟動時照 E3「換了位置時」逐項搬、跳
  toast；兩邊都有時留著、開 error popup 寫兩個路徑；鎖定畫面與 `locku pin reset` 在搬之前照舊讀 config.yaml。定 B（還是設定）時
  不用改，刪掉這一條。
