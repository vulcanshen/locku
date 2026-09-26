# locku

<p align="center"><img src="docs/icon.svg" width="128" alt="locku icon" /></p>

[![GitHub Release](https://img.shields.io/github/v/release/vulcanshen/locku)](https://github.com/vulcanshen/locku/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vulcanshen/locku)](https://go.dev/)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue)](LICENSE)

**語言**：[English](README.md) · 繁體中文

**終端機的螢幕保護程式加 PIN 鎖。** locku 接管整個終端機：一面 LED 點陣板寫著時鐘、Chrome 的離線小恐龍永遠跑下去，或是你自己的程式。任何鍵只會叫出 PIN 輸入框，PIN 對了終端機才回來。作為 tmux 的 `lock-command`、screen 的 `LOCKPRG` 接上去，或在裸 tty 直接執行，所以 prefix 之後一個鍵、或閒置幾分鐘，就把你離開的那個畫面鎖起來。不設 PIN 就是純螢幕保護：任何鍵結束。

> _不確定的時候，就按_ **`Space`**。

## Demo

![demo](docs/demo.gif)

`locku lock` 把終端機變成時鐘點陣板；按一個鍵叫出 PIN 框，PIN 錯了框變紅，對了終端機回來。接著是設定畫面：`[2]` 裡一個 profile 的設定、用 `p` 預覽 dino 與你自己的程式（cmatrix）、`Space` 列出這一列能做的事、Integration 底下的 tmux。

## 你會看到什麼

三種 saver，每種可以生任意多個有名字的 profile，其中一個是啟用中的：

- **clock**——整個終端機是一面 LED 點陣板：每個像素是一個 Nerd Font 方塊，暗格是 profile 的 `bg`、亮格是它的 `fg`。時間用全直角的 3 × 7 像素字型（或 3 × 5）畫，像七段顯示器，`HH MM` 或 `HH MM SS`，24 時制，不畫冒號；日期在下面，四種格式或關掉。`column` 直排把 `HH` / `MM` / `SS` 疊起來、字大好幾倍，日期在左邊。三種 size；塞不下就先去年、去秒、去日期，再降 size。只重畫有變的像素，用 shuffle 的方式揭露。
- **dino**——Chrome 的離線小恐龍遊戲當螢幕保護：地面與障礙向左捲，草原上是仙人掌、沙漠裡是金字塔，暴龍自己跳過去，無限循環、不會死；不記分、不畫時間。一隻或兩隻、大或小，畫布自己取塞得下的最大倍率。
- **custom**——你自己的程式當畫面：例如 `cmatrix -b`，任何會畫畫面的東西，經 `sh -c` 跑。locku 管鎖、PIN 與整合；程式跑在 locku 開的 pty 上，輸出原樣直通，按鍵永遠到不了它。PIN 框直接疊在還在動的畫面上，解鎖時程式跟鎖一起結束。

clock 各 size 需要的終端機（欄 × 列，`3x7` / `3x5`）：

| 內容 | small | medium | large |
|---|---|---|---|
| `HH MM` 一行 | 38 × 10 / 8 | 62 × 17 / 13 | 96 × 24 / 18 |
| `HH MM SS` 一行 | 58 × 10 / 8 | 94 × 17 / 13 | 148 × 24 / 18 |
| `HH` / `MM` 直排 | 18 × 18 / 14 | 30 × 32 / 24 | 44 × 47 / 35 |
| `HH` / `MM` / `SS` 直排 | 18 × 26 / 20 | 30 × 47 / 35 | 44 × 70 / 52 |

## 安裝

> locku **只支援 macOS / Linux**（WSL 可）。沒有 Windows 版：鎖站在 tty、pty、`su` 與 tmux / screen 上。

**Homebrew**（macOS / Linux）：

```bash
brew install vulcanshen/tap/locku
```

**安裝腳本**（把最新 release 的 binary 放到 `~/.local/bin`，root 則是 `/usr/local/bin`）：

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/locku/main/install.sh | sh
```

從原始碼建置見 [`docs/dev-remarks.md`](docs/dev-remarks.md)。

**Nerd Font 必裝**：點陣板的每個像素就是 nf-fa-square，設定畫面也用 Nerd Font 字符畫。沒有的話整面板子是一堆方框。

### 移除

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/locku/main/uninstall.sh | sh
```

移除 binary，然後問你——不擅自決定——要不要刪設定目錄。先在設定畫面把 `activate` 關掉，tmux.conf、screenrc 與 shell rc 裡 locku 的區塊就沒了；移除腳本不碰那些檔。

## 快速開始

```bash
locku            # 設定畫面：設 PIN、選 saver、接上 tmux / screen
locku lock       # 現在就鎖住這個終端機
locku pin reset  # 忘記 PIN 時，用登入密碼換一組新的
locku version    # 版本；locku help 印用法
```

1. `locku`，到 **Settings › preference**，PIN 那列按 `Enter`，輸入一組。這步可省略：不設 PIN，locku 就是螢幕保護，任何鍵結束。
2. **Integration › tmux**（或 **screen**）：填 `config file path`（會提議 `~/.tmux.conf`），把 `activate` 打開、確認。locku 的區塊已在檔案裡，跑著的 tmux server 也立刻收到。
3. 在 tmux 裡 `prefix :` 打 `locku`，整台 server 的 client 一起鎖。閒置 `lock-after-time` 秒（預設 300）的 session 自己鎖。填一個 `bind-key`，例如 `l`，`prefix l` 也鎖。
4. 任何鍵叫出 PIN 框；PIN 對了終端機回來。

在裸 tty（例如一條 ssh）跑 `locku lock`，那個終端機一樣被鎖。它的 `-S` / `-t` 是 tmux 整合寫進 lock-command 的，你不用打。screen 用 `SCREEN-LOCK` 這個名字、不帶參數執行鎖，locku 認得這個名字。

## 鎖定畫面

- 任何鍵開 PIN 框，那個鍵不算輸入。`Enter` 送出、`Esc` 回 saver、`Backspace` 刪一字。
- 錯誤 PIN 邊框變紅 1 秒並吞掉所有輸入。連錯 `wrong_pin_attempts` 次進入 `wrong_pin_attempt_cooldown` 秒倒數（預設關）；`pin_prompt_timeout` 秒沒按鍵框自動收起（預設 30）。
- 狀態列 `user@host · locked since HH:MM`（`show_status: false` 可關），沒設 PIN 時標明 `no PIN · any key unlocks`。
- Ctrl+C、Ctrl+Z、Ctrl+\ 只是按鍵；SIGINT / SIGTERM / SIGHUP 一律忽略；panic 後鎖定畫面重新升起。進程只在三種情況結束：PIN 正確、無 PIN 時任意鍵、終端機消失。
- 沒設 PIN、或 config 讀不到，一律 fail open：saver 照常顯示、狀態列說明原因、任何鍵結束。

**不是安全邊界。** locku 擋的是誤觸與路人，擋不住你自己帳號的另一條 session。攔不到的：ssh 的 `~.`（在 client 端處理，byte 不會過線）、Linux VT 切換（vlock 的領域，要 root）、另一條 SSH 的 `tmux attach -d` / `kill -9`、終端機模擬器自身的快捷鍵。

## 忘記 PIN

從你自己的任何一個 shell：

```bash
locku pin reset
```

先問 `[y/N]`，再要你的**登入密碼**——帳號是 locku 唯一的邊界——然後產生一組新的八位數 PIN、覆蓋舊的、顯示一次。鎖定中的畫面下一鍵就認得新 PIN。之後到設定畫面換成自己要的。每次 reset，成功或被拒，都記在 `~/.locku/data` 底下，不含 PIN。

## 設定畫面

```
╔[1] locku═════════════╗╭[2] clock  unsaved─────────────────────────────╮
║ Profiles               ║│ Property          Value                          │
║ ● clock                ║│ name              clock                          │
║   clock2               ║│ saver             clock                          │
║   dino                 ║│ layout            row                            │
║ Savers                 ║│ size              medium                         │
║   clock                ║│ time              HH MM                          │
║   dino                 ║│ date              off                            │
║ Integration            ║│ bg                ■ #313244  →  ■ #ff3244        │
║   tmux                 ║│   R               ───────────● 255               │
║   screen               ║│   G               ──●───────── 50                │
║ Settings               ║│   B               ───●──────── 68                │
║   preference           ║│ fg                ■ #f2b753                      │
╚════════════════════════╝╰──────────────────────────────────────────────────╯
 space menu   ? help   tab/1-2 panels   q quit
```

兩個面板：**`[1]`** 側欄，**`[2]`** 游標那列的內容，Property / Value 兩欄的表。`Tab`、`1`、`2` 在兩邊移動；`Enter` 進 `[2]` 或編輯一列；`Esc` 關浮層；`Space` 列出當前能做的事；`?` 是全部的鍵，在 preference、tmux、screen 的 `[2]` 上則是每一列的說明。

- **Profiles**——你設定好、有名字的 saver。`●` 是啟用中的、鎖定畫面顯示的那個；`a` 把游標那個設為啟用、`p` 預覽、`D` duplicate、`r` rename、`X` delete。它的 `[2]` 是它的設定：clock 的 `layout`、`size`、`font`、`time`、`date`；dino 的 `runner`、`scene`；custom 的 `command`；以及 `bg` / `fg` 各三個 RGB slider，改的是草稿，`S` 才寫檔（`R` 丟掉；有未存草稿時 `q` 先問）。其他每一列一改就寫檔。
- **Savers**——三種種類：clock、dino、custom。每個 `[2]` 是說明加**預設值**，之後用這種 saver 新增的 profile 就從這裡開始；`n` 生一個、`p` 用預設值預覽。改預設值不動既有的 profile。
- **Integration**——tmux 與 screen，見下。
- **Settings › preference**——PIN（設定；已設時 `Enter` 先驗目前的，再選 `New PIN` 或 `Remove PIN`）、啟用的 `profile`、`show_status`、`pin_prompt_timeout`、`wrong_pin_attempts`、`wrong_pin_attempt_cooldown`。

任何 `[2]` 上 `P` 就地預覽鎖定畫面：profile 或 saver 的 `[2]` 是那一個，其他是啟用中的 profile。任意鍵回來，不驗 PIN。custom profile 的預覽把終端機整個交給程式，直到按鍵。

## tmux 與 screen

Integration › tmux 與 Integration › screen 各有 `activate`（on / off）、`config file path`（要寫的檔；會提議 `~/.tmux.conf` 與 `~/.screenrc`），分隔線下面是工具自己的 key：

| | tmux | screen |
|---|---|---|
| 閒置幾秒工具自己鎖（0 不鎖） | `lock-after-time` | `idle` |
| prefix 之後按哪個鍵就鎖，照工具自己的寫法；空就不綁 | `bind-key`（`l`、`C-l`） | `bind`（`l`、`^L`）；`C-a x` 內建就鎖 |
| 鎖什麼 | `lock`：`lock-server`（整台 server 的 client）或 `lock-session`（只鎖這個 session） | ——（screen 沒有 server 可選範圍） |

`activate` 打開，confirm 後把 locku 的區塊寫進檔案；開著的時候任何一列一改就重寫：跑著的 tmux server 整塊立刻收到，跑著的 screen 立刻收到 `idle` 與 `bind`。關掉就拿掉，檔案與跑著的都拿掉。只碰兩個標記之間的區塊，每行尾巴 `# locku`；檔案其他部分是你的。路徑沒填，`activate` 按不下去：不猜。

tmux 拿到的：

```
# >>> locku >>>
set -gF lock-command "/opt/homebrew/bin/locku lock -S '#{socket_path}'"  # locku
set -g lock-after-time 300                                                  # locku: 0 never
set -s "command-alias[90]" "locku=lock-server"                              # locku: prefix : locku locks every client
set-hook -g "client-attached[90]" "if -F \"#{@locked}\" lock-client"        # locku: attaching while locked locks the client
set-hook -g "client-session-changed[90]" "if -F \"#{@locked}\" lock-client" # locku: so does switching sessions
bind-key l lock-server                                                      # locku: prefix l locks every client
# <<< locku <<<
```

`prefix :` 打 `locku` 就鎖，`bind-key` 填 `l` 時 `prefix l` 也鎖。鎖著的時候誰來都被鎖：鎖定中 attach 進來、或切換 session，一樣落在保護程式上，直到 PIN 解開。`lock` 選 `lock-session` 時同樣的行改指向 `lock-session`，多一個 hook 給每個 session 自己的 lock-command，其他 session 照常用。

screen 拿到的：

```
# >>> locku >>>
idle 300 lockscreen   # locku: 0 never
bind l lockscreen     # locku: C-a l locks, as C-a x does
# <<< locku <<<
```

而且因為 screen 的鎖定程式只從啟動它的 shell 環境讀、從不讀 screenrc，shell rc（`~/.zshrc`、`~/.bashrc`，或 fish 的 `config.fish`）也拿到：

```
# >>> locku >>>
export LOCKPRG=/usr/local/bin/locku   # locku: screen's LOCKPRG
# <<< locku <<<
```

新開的 shell 就有。已在跑的 screen session 要 detach 後從新 shell 重新 attach 才有；在那之前那個 session 鎖到的是 screen 內建的鎖，設定畫面會說明。區塊是同一份文字，locku 寫或你手寫都一樣。

## 按鍵

### 到處都通

| 鍵 | |
|---|---|
| `Tab` · `1` · `2` | 下一個面板 / 直達面板 |
| `Enter` | `[1]` 上：這列的欄位進 `[2]`；`[2]` 上：編輯、選、切換、挑 |
| `Esc` | 關最上層的浮層 |
| `Space` | 這裡能做什麼：item 與 panel |
| `?` | help：全部的鍵；preference、tmux、screen 的 `[2]` 上是每一列的說明 |
| `P` | `[2]` 上：預覽鎖定畫面；任意鍵回來 |
| `q` | 離開；有未存的顏色先問 |
| `j` / `k` · `u` / `d` · `gg` / `G` | 下 / 上 · 半頁 · 首 / 尾 |

### `[1]` 側欄

| 鍵 | 在 | |
|---|---|---|
| `n` | saver | 用這種 saver 生一個新 profile，取名 |
| `p` | saver / profile | 預覽預設值 / 這個 profile |
| `a` | profile | 設為啟用：鎖定畫面從此顯示這個 |
| `D` · `r` · `X` | profile | duplicate · rename · delete（啟用中的與最後一個不能刪） |

### `[2]` 明細

| 鍵 | 在 | |
|---|---|---|
| `Enter` | 任何一列 | rename、選、切換、挑顏色通道、設或換 PIN、改路徑或鍵、開關 `activate` |
| `S` · `R` | profile 或 saver | 存顏色草稿 · 丟掉 |

### 鎖定畫面

| 鍵 | |
|---|---|
| 任何鍵 | 開 PIN 框（那個鍵不算輸入） |
| `Enter` · `Esc` · `Backspace` | 送出 · 回 saver · 刪一字 |

## 你的資料放在哪

| | 什麼 | 哪裡 |
|---|---|---|
| 設定 | `config.yaml`——PIN 的 hash、profile、各 saver 的預設值、整合設定 | `~/.config/locku`（有設 `$XDG_CONFIG_HOME` 就是 `$XDG_CONFIG_HOME/locku`；`$LOCKU_CONFIG` 直接指定） |
| 資料 | `pin-resets.log`——每次 `locku pin reset`，不含 PIN | `~/.locku/data`（`$LOCKU_DATA`） |

無 history、無 cache、無 session。`config.yaml` 原子寫入、mode 0600，可以手改：

```yaml
auth: pin                 # 唯一的驗證；pam 保留
pin_hash: "$2a$10$..."    # bcrypt；空或缺欄位 = 無 PIN，任何鍵解鎖
profile: clock            # 啟用中的 profile
profiles:
  - name: clock
    saver: clock          # clock / dino / custom；建立後不改
    layout: row           # row / column
    size: medium          # small / medium / large：一個像素佔 1 / 2 / 3 格見方
    font: 3x7             # 3x7 / 3x5
    time: "HH MM"         # HH MM / HH MM SS
    date: off             # off / YYYY-MM-DD / YYYY-MMM-DD / MM-DD / MMM-DD
    bg: "#313244"
    fg: "#f2b753"
  - name: dino
    saver: dino
    runner: big           # big / small / big-big / small-small / small-big / big-small
    scene: grassland      # grassland / desert
    bg: "#313244"
    fg: "#f2b753"
  - name: matrix
    saver: custom
    command: "cmatrix -b" # sh -c 跑；沒有自己的顏色
savers:                   # 每種 saver 的預設值：新 profile 從這裡開始
  clock: { saver: clock, layout: row, size: large, font: 3x5, time: "HH MM SS", date: YYYY-MM-DD, bg: "#313244", fg: "#f2b753" }
  dino: { saver: dino, runner: big, scene: grassland, bg: "#313244", fg: "#f2b753" }
  custom: { saver: custom, command: "" }
show_status: true               # user@host · locked since 那一列
pin_prompt_timeout: 30          # 幾秒沒按鍵框收起；0 永不收起
wrong_pin_attempts: 0           # 連錯幾次進冷卻；0 關
wrong_pin_attempt_cooldown: 30  # 冷卻秒數
tmux:
  conf: "~/.tmux.conf"          # config file path；空 = activate 打不開
  lock-after-time: 300
  bind-key: ""
  lock: lock-server             # lock-server / lock-session
screen:
  conf: "~/.screenrc"
  idle: 300
  bind: ""
```

## 限制

刻意不做的：
- **Windows**——鎖站在 tty、pty、`su` 與 tmux / screen 上，原生移植是另一個產品
- **系統密碼**——驗證只有 locku 自己的 PIN；`auth: pam` 只是保留位
- **鎖 Linux 的虛擬主控台**（Alt+F1 … F7）——vlock 的領域
- **鎖定畫面上的「忘記 PIN」入口**——回來的路是 `locku pin reset`，在 shell 裡、用登入密碼

## 相關連結

- [CHANGELOG.md](CHANGELOG.md)——每個版本改了什麼
- [`docs/dev-remarks.md`](docs/dev-remarks.md)——開發者備忘錄：怎麼運作、為什麼、設計文件導讀、建置與測試

## terminu family

locku 遵循 [terminu design principle](https://github.com/vulcanshen/terminu)：跟家族其他成員一樣的按鍵、一樣的 menu——[kbu](https://github.com/vulcanshen/kbu)（Kubernetes）、[filu](https://github.com/vulcanshen/filu)（檔案）、[sshu](https://github.com/vulcanshen/sshu)（ssh）、[webu](https://github.com/vulcanshen/webu)（網頁）。

## 授權

[GPL-3.0](LICENSE)
