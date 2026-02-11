# Mac での使い方

## 1. 事前準備

### GStreamer のインストール（映像キャプチャに必要）

```bash
brew install gstreamer gst-plugins-base gst-plugins-good
```

### ハードウェア接続

- USB HDMIキャプチャデバイスをMacに接続
- FT232 USB-Serialアダプタを Macに接続し、下記の通りRPi4に配線

#### FT232 電圧設定

**必ず 3.3V に設定すること。** RPi4のGPIOは3.3Vロジックです。5Vを入力するとGPIOが破損します。
FT232モジュールのジャンパまたはスイッチを **3.3V側** にしてから接続してください。

#### FT232 → RPi4 配線

```
FT232モジュール          RPi4 (40ピンヘッダ)
┌──────────┐            ┌─────────────────────────┐
│  TX  ──────────────── │ Pin 10 (RXD / GPIO15)   │
│  RX  ──────────────── │ Pin  8 (TXD / GPIO14)   │
│  GND ──────────────── │ Pin  6 (GND)            │
│  VCC │  接続しない     │                         │
└──────────┘            └─────────────────────────┘
```

- **TX↔RX はクロス接続** （FT232 TX → RPi4 RX、FT232 RX → RPi4 TX）
- **VCC は接続しない** （RPi4は別途USB-C等で電源供給）

#### RPi4 40ピンヘッダの物理位置

```
   USB端子側
  ┌─────────┐
  │ [1] [2] │  1=3.3V   2=5V
  │ [3] [4] │
  │ [5] [6] │  6=GND  ← GND ここ
  │ [7] [8] │  8=TXD  ← FT232の RX に接続
  │ [9][10] │ 10=RXD  ← FT232の TX に接続
  │  ...    │
  └─────────┘
```

#### RPi4 UART設定

RPi4はデフォルトでGPIO14/15にmini UART (ttyS0) を割り当てており、
PL011 (ttyAMA0) はBluetoothに使われています。
dejima-kvm-daemon-rpi は `/dev/ttyAMA0` を使うため、PL011をGPIO14/15に切り替えます。

```bash
# /boot/firmware/config.txt の末尾に追加
dtoverlay=disable-bt
enable_uart=1
```

```bash
# Bluetoothサービスを停止して再起動
sudo systemctl disable hciuart
sudo reboot
```

再起動後の確認:

```bash
ls -la /dev/ttyAMA0    # 存在すればOK
```

## 2. 起動

開発モード:

```bash
cd /Users/syagox/dev/ai/kvm_like
wails dev
```

プロダクションビルド:

```bash
wails build
# → build/bin/dejima-kvm.app が生成される
```

## 3. 操作の流れ

```
┌─────────────────────────────────────────┐
│ [Dejima]     [Start Video]   [Settings] │  ← Toolbar
├─────────────────────────────────────────┤
│                                         │
│         映像表示エリア                    │  ← クリックで入力キャプチャ開始
│      (クリックして入力開始)               │     Escで解除
│                                         │
├─────────────────────────────────────────┤
│ ● Video: Off   ● Serial: Disconnected  │  ← StatusBar
└─────────────────────────────────────────┘
```

### Step 1: 映像表示

1. USBキャプチャデバイスを接続
2. **Start Video** ボタンをクリック
3. RPi4のHDMI出力がウィンドウに表示される

### Step 2: シリアル接続

1. **Settings** ボタンをクリック
2. **Refresh** でポート一覧を取得、または **Auto-detect** でFT232を自動検出
3. ポートを選択して **Connect**（例: `/dev/tty.usbserial-XXXX`）

### Step 3: 入力操作

1. 映像エリアを **クリック** → pointer lockが有効になり入力キャプチャ開始
2. キーボード・マウス操作がそのままRPi4に送信される
3. **Esc** キーでpointer lockを解除（操作をMacに戻す）

## 4. RPi4 側のセットアップ

RPi4にデーモンをデプロイ:

```bash
# Mac側でクロスコンパイル
make build-rpi

# RPi4に転送
scp build/bin/dejima-kvm-daemon-rpi pi@<rpi-ip>:/usr/local/bin/
scp rpi-daemon/dejima-kvm-rpi.service pi@<rpi-ip>:/etc/systemd/system/

# RPi4側で有効化
ssh pi@<rpi-ip>
sudo systemctl enable --now dejima-kvm-rpi
```

## 5. デバイス番号の確認

USBキャプチャデバイスが複数ある場合、Settings で **Device Index** を変更できます。デバイス一覧の確認:

```bash
# GStreamerで認識されるデバイスを表示
gst-device-monitor-1.0 Video/Source
```

## まとめ

| 要素 | 内容 |
|------|------|
| 映像入力 | USB HDMI キャプチャ → GStreamer → MJPEG表示 |
| キー/マウス出力 | Wails → FT232 serial → RPi4 UART → uinput |
| 入力キャプチャ切替 | クリックで開始、Escで解除 |
| 必須ハード | USB HDMIキャプチャ、FT232、RPi4 |
