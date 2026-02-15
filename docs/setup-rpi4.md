# Raspberry Pi 4 セットアップガイド

Dejima デーモン (`dejima-kvm-daemon-rpi`) を RPi4 にインストールし、
ホスト PC からキーボード・マウスを操作できるようにするまでの手順です。

---

## 1. 前提条件

### OS

**Raspberry Pi OS Bookworm (64-bit)** を推奨します。Wayland (labwc) がデフォルトで有効です。

```bash
cat /etc/os-release   # VERSION_CODENAME=bookworm を確認
uname -m              # aarch64 を確認
```

> Bullseye 以前 (X11) でも動作しますが、xdotool / xclip が必要になります。

### ハードウェア接続

ホスト PC 側の FT232 USB-Serial アダプタを RPi4 の GPIO ヘッダに接続します。

#### FT232 電圧設定

**必ず 3.3V に設定してください。** RPi4 の GPIO は 3.3V ロジックです。
5V を入力すると GPIO が破損します。
FT232 モジュールのジャンパまたはスイッチを **3.3V 側** にしてから接続してください。

#### FT232 → RPi4 配線

```
FT232 モジュール          RPi4 (40ピンヘッダ)
┌──────────┐            ┌─────────────────────────┐
│  TX  ──────────────── │ Pin 10 (RXD / GPIO15)   │
│  RX  ──────────────── │ Pin  8 (TXD / GPIO14)   │
│  GND ──────────────── │ Pin  6 (GND)            │
│  VCC │  接続しない     │                         │
└──────────┘            └─────────────────────────┘
```

- **TX↔RX はクロス接続** （FT232 TX → RPi4 RX、FT232 RX → RPi4 TX）
- **VCC は接続しない** （RPi4 は別途 USB-C 等で電源供給）

#### RPi4 40ピンヘッダの物理位置

```
   USB 端子側
  ┌─────────┐
  │ [1] [2] │  1=3.3V   2=5V
  │ [3] [4] │
  │ [5] [6] │  6=GND  ← GND ここ
  │ [7] [8] │  8=TXD  ← FT232 の RX に接続
  │ [9][10] │ 10=RXD  ← FT232 の TX に接続
  │  ...    │
  └─────────┘
```

### UART の有効化

RPi4 はデフォルトで GPIO14/15 に mini UART (`ttyS0`) を割り当てており、
高性能な PL011 UART (`ttyAMA0`) は Bluetooth が使用しています。
dejima-kvm-daemon-rpi は `/dev/ttyAMA0` を使うため、以下の設定で PL011 を GPIO14/15 に切り替え、
Bluetooth を無効にします。

#### config.txt の編集

```bash
sudo nano /boot/firmware/config.txt
```

ファイル末尾に以下を追加:

```ini
# Dejima: PL011 UART を GPIO14/15 に割り当て、BT を無効化
dtoverlay=disable-bt
enable_uart=1
```

> `dtoverlay=disable-bt` は Bluetooth を無効にし、PL011 を GPIO14/15 に戻します。
> `enable_uart=1` は UART を明示的に有効にします。

#### Bluetooth 関連サービスの無効化

```bash
sudo systemctl disable hciuart
sudo systemctl disable bluetooth
```

#### シリアルコンソールの無効化

Raspberry Pi OS はデフォルトでシリアルポートをログインコンソールとして使用しています。
dejima-kvm-daemon-rpi がシリアルポートを専有するため、これを無効にします。

```bash
# シリアルコンソールを無効化
sudo raspi-config nonint do_serial_cons 1   # 1 = disable

# または手動で
sudo systemctl disable serial-getty@ttyAMA0.service
sudo systemctl stop serial-getty@ttyAMA0.service
```

`/boot/firmware/cmdline.txt` からもコンソール指定を削除:

```bash
# 編集前に確認
cat /boot/firmware/cmdline.txt
```

`console=serial0,115200` や `console=ttyAMA0,115200` があれば削除してください。

```bash
sudo sed -i 's/console=serial0,[0-9]* //g' /boot/firmware/cmdline.txt
sudo sed -i 's/console=ttyAMA0,[0-9]* //g' /boot/firmware/cmdline.txt
```

#### 再起動と確認

```bash
sudo reboot
```

再起動後:

```bash
ls -la /dev/ttyAMA0                        # 存在すれば OK
systemctl is-active serial-getty@ttyAMA0    # inactive であること
systemctl is-active bluetooth              # inactive であること
```

### uinput モジュール

dejima-kvm-daemon-rpi は `/dev/uinput` を通じて仮想キーボード・マウスを作成します。

```bash
# カーネルモジュールを起動時に自動ロード
echo "uinput" | sudo tee /etc/modules-load.d/uinput.conf
sudo modprobe uinput

# 確認
ls -la /dev/uinput     # 存在すれば OK
```

### 必要パッケージのインストール

```bash
sudo apt update
sudo apt install -y wtype wl-clipboard
```

| パッケージ | 用途 |
|-----------|------|
| `wtype` | Type モード: 仮想キーストロークでテキスト入力 |
| `wl-clipboard` | Paste モード: クリップボード経由でテキスト入力 (`wl-copy`, `wl-paste`) |

> X11 環境の場合は代わりに `sudo apt install -y xdotool xclip` を実行してください。

---

## 2. デーモンのインストール

### ホスト PC 側でクロスコンパイル

```bash
# プロジェクトルートで実行 (Mac / Linux)
make build-rpi
```

`build/bin/dejima-kvm-daemon-rpi` (linux/arm64) が生成されます。

### RPi4 への転送

```bash
RPI=pi@<rpi-ip>

# バイナリ
scp build/bin/dejima-kvm-daemon-rpi ${RPI}:/tmp/
ssh ${RPI} 'sudo mv /tmp/dejima-kvm-daemon-rpi /usr/local/bin/ && sudo chmod +x /usr/local/bin/dejima-kvm-daemon-rpi'

# systemd サービスファイル
scp rpi-daemon/dejima-kvm-rpi.service ${RPI}:/tmp/
ssh ${RPI} 'sudo mv /tmp/dejima-kvm-rpi.service /etc/systemd/system/'
```

### 動作確認 (手動実行)

サービスとして登録する前に、手動で起動して問題がないか確認します。

```bash
# RPi4 上で実行
sudo /usr/local/bin/dejima-kvm-daemon-rpi -diag
```

出力例:

```
=== dejima-kvm-daemon-rpi diagnostics ===
Version:  20260208-153000
Go:       go1.23 linux/arm64

--- Display environment ---
  WAYLAND_DISPLAY              wayland-0
  XDG_RUNTIME_DIR              /run/user/1000
  ...

--- Required tools ---
  wtype          [wayland]  OK  (/usr/bin/wtype)
  wl-paste       [wayland]  OK  (/usr/bin/wl-paste)
  wl-copy        [wayland]  OK  (/usr/bin/wl-copy)
  ...

--- UART ---
  /dev/ttyAMA0       OK  (mode: crw-rw----)

--- uinput ---
  /dev/uinput        OK  (mode: crw-------)
```

確認ポイント:
- `wtype`, `wl-copy`, `wl-paste` が **OK** であること
- `/dev/ttyAMA0` が存在すること
- `/dev/uinput` が存在すること

問題がなければ Ctrl+C で停止します。

### サービスとして登録・起動

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now dejima-kvm-rpi
```

```bash
# 状態確認
sudo systemctl status dejima-kvm-rpi

# ログ確認
journalctl -u dejima-kvm-rpi -f
```

---

## 3. dejima-kvm-rpi.service のカスタマイズ

`/etc/systemd/system/dejima-kvm-rpi.service` の内容:

```ini
[Unit]
Description=Dejima KVM HID Daemon (RPi)
After=graphical.target

[Service]
Type=simple
Environment=WAYLAND_DISPLAY=wayland-0
Environment=XDG_RUNTIME_DIR=/run/user/1000
Environment=DISPLAY=:0
ExecStart=/usr/local/bin/dejima-kvm-daemon-rpi -device /dev/ttyAMA0 -baud 115200
Restart=always
RestartSec=3
User=root

[Install]
WantedBy=graphical.target
```

### 変更が必要な場合

| 項目 | デフォルト | いつ変更するか |
|------|-----------|---------------|
| `-device` | `/dev/ttyAMA0` | 別の UART デバイスを使う場合 |
| `-baud` | `115200` | ホスト側の BaudRate 設定と合わせる |
| `XDG_RUNTIME_DIR` | `/run/user/1000` | デスクトップユーザーの UID が 1000 以外の場合 |
| `WAYLAND_DISPLAY` | `wayland-0` | Wayland ソケット名が異なる場合 |

変更後:

```bash
sudo systemctl daemon-reload
sudo systemctl restart dejima-kvm-rpi
```

---

## 4. アップデート手順

バイナリを差し替えてサービスを再起動するだけです。

```bash
# ホスト PC 側
make build-rpi
scp build/bin/dejima-kvm-daemon-rpi ${RPI}:/tmp/

# RPi4 側
ssh ${RPI} 'sudo systemctl stop dejima-kvm-rpi && sudo mv /tmp/dejima-kvm-daemon-rpi /usr/local/bin/ && sudo systemctl start dejima-kvm-rpi'
```

---

## 5. トラブルシューティング

### wtype が NOT FOUND

```bash
sudo apt install wtype
```

Bookworm の apt にない場合は [wtype のリポジトリ](https://github.com/atx/wtype) からビルドしてください。

### /dev/ttyAMA0 がない

`/boot/firmware/config.txt` に `dtoverlay=disable-bt` と `enable_uart=1` が入っているか確認し、再起動してください。

### /dev/uinput がない

```bash
sudo modprobe uinput
```

永続化: `echo "uinput" | sudo tee /etc/modules-load.d/uinput.conf`

### Type モードで文字が入力されない

- ターミナルアプリにフォーカスがあるか確認
- `wtype` がインストール済みか確認 (`which wtype`)
- ログに権限エラーがないか確認: `journalctl -u dejima-kvm-rpi --no-pager -n 20`

### Paste モードで文字が入力されない

- ブラウザ等の GUI アプリにフォーカスがあるか確認
- `wl-copy` がインストール済みか確認 (`which wl-copy`)
- ターミナルでは Ctrl+V がペーストとして機能しないため、Type モードを使用してください

### daemon が即座に終了する

`graphical.target` が完了する前に起動している可能性があります。
ログを確認してください:

```bash
journalctl -u dejima-kvm-rpi --no-pager -n 50
```

---

## 6. アンインストール手順

Dejima KVM のシリアル UART 設定をすべて元に戻し、デーモンを削除する手順です。

### デーモンの停止・削除

```bash
# サービスの停止と無効化
sudo systemctl stop dejima-kvm-rpi
sudo systemctl disable dejima-kvm-rpi

# サービスファイルの削除
sudo rm /etc/systemd/system/dejima-kvm-rpi.service
sudo systemctl daemon-reload

# バイナリの削除
sudo rm /usr/local/bin/dejima-kvm-daemon-rpi
```

### UART 設定の復元

#### config.txt から Dejima の設定を削除

```bash
sudo nano /boot/firmware/config.txt
```

ファイル末尾に追加した以下の行を削除してください:

```ini
# Dejima: PL011 UART を GPIO14/15 に割り当て、BT を無効化
dtoverlay=disable-bt
enable_uart=1
```

#### Bluetooth の再有効化

```bash
sudo systemctl enable hciuart
sudo systemctl enable bluetooth
```

#### シリアルコンソールの再有効化 (必要な場合)

セットアップ時にシリアルコンソールを無効にした場合、必要に応じて再有効化できます。

```bash
sudo raspi-config nonint do_serial_cons 0   # 0 = enable

# または手動で
sudo systemctl enable serial-getty@ttyAMA0.service
```

### uinput 自動ロードの解除 (任意)

他のアプリケーションが uinput を使用していなければ削除できます。

```bash
sudo rm /etc/modules-load.d/uinput.conf
```

### インストールしたパッケージの削除 (任意)

`wtype` や `wl-clipboard` が他の用途で不要であれば削除できます。

```bash
sudo apt remove -y wtype wl-clipboard
```

### 再起動

すべての変更を反映するために再起動してください。

```bash
sudo reboot
```

再起動後、Bluetooth が復活していることを確認:

```bash
systemctl is-active bluetooth    # active であること
hciconfig                        # hci0 が表示されること
```
