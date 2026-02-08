# Windows セットアップガイド

## 前提条件

### 1. GStreamer のインストール

GStreamer の Windows 向けランタイムと開発パッケージをインストールする。

1. winget でインストール:
   ```
   winget install --id=gstreamerproject.gstreamer -e
   ```
2. 環境変数 `PATH` に `C:\gstreamer\1.0\msvc_x86_64\bin` を追加（自動追加されない場合）
4. コマンドプロンプトで動作確認:
   ```
   gst-launch-1.0 --version
   ```

### 2. FT232 ドライバ (FTDI VCP) のインストール

RPi との USB シリアル通信に FTDI VCP (Virtual COM Port) ドライバが必要。

1. https://ftdichip.com/drivers/vcp-drivers/ から Windows 用ドライバをダウンロード
2. インストーラを実行
3. FT232 デバイスを接続し、デバイスマネージャで COM ポートが認識されることを確認

### 3. USB HDMI キャプチャデバイス

USB HDMI キャプチャデバイスを PC に接続する。Windows 標準の UVC ドライバで自動認識される。

## ビルド

macOS 上でクロスコンパイル:

```bash
make build-windows
```

`build/bin/` に `.exe` が生成される。

## 起動

1. USB HDMI キャプチャデバイスと FT232 を Windows PC に接続
2. `kvm-like.exe` を実行
3. 設定画面でキャプチャデバイスとシリアルポート (COMx) を選択
