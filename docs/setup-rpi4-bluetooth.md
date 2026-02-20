# Bluetooth シリアル接続セットアップガイド

FT232 USB-Serial（有線 UART）の代わりに Bluetooth Serial Port Profile (SPP) を使い、
ホスト PC と RPi をケーブルレスで接続する手順です。

> **対応ハードウェア**: RPi4 / RPi5 の両方で動作確認済みです。

> **既存コードの変更は不要です。**
> Dejima KVM のシリアル実装はデバイスパスに依存しない汎用設計のため、
> RPi 側のデバイスを `/dev/ttyAMA0` → `/dev/rfcomm0` に変更するだけで動作します。

---

## 1. 前提条件

### 有線 UART 設定の解除

既に [有線 UART セットアップ](setup-rpi4.md) を実施済みの場合、
Bluetooth を無効化する設定が入っています。先に解除してください。

```bash
sudo nano /boot/firmware/config.txt
```

以下の行を**削除**:

```ini
# Dejima: PL011 UART を GPIO14/15 に割り当て、BT を無効化
dtoverlay=disable-bt
enable_uart=1
```

Bluetooth 関連サービスを再有効化:

```bash
sudo systemctl enable bluetooth
# hciuart がある場合のみ (Bookworm では存在しないことがあります):
sudo systemctl enable hciuart 2>/dev/null
sudo reboot
```

再起動後に確認:

```bash
systemctl is-active bluetooth    # active であること
hciconfig                        # hci0 が UP RUNNING であること
```

### uinput・必要パッケージ

[有線 UART セットアップガイド](setup-rpi4.md) の「uinput モジュール」と「必要パッケージのインストール」セクションの設定が完了していること。
まだの場合は先に実施してください。

---

## 2. RPi4 側: Bluetooth SPP サーバーのセットアップ

### Bluetooth の有効化

Bluetooth が rfkill でブロックされている場合があります。先に解除してください:

```bash
rfkill list
# hci0 の Soft blocked: yes の場合:
sudo rfkill unblock bluetooth
```

### Bluetooth の設定

```bash
sudo bluetoothctl
```

`bluetoothctl` のプロンプトで以下を実行:

```
power on
discoverable on
pairable on
agent on
default-agent
```

`exit` で `bluetoothctl` を終了します。

> `power on` が `Failed to set power on` で失敗する場合は、セクション 8 のトラブルシューティングを参照してください。

### デバイス名の変更 (推奨)

デフォルトのデバイス名 (`raspberrypi`) はペアリング時にわかりにくいため、変更を推奨します:

```bash
sudo nano /etc/machine-info
```

内容:

```
PRETTY_HOSTNAME=dejima-kvm
```

反映:

```bash
sudo systemctl restart bluetooth
```

確認:

```bash
sudo bluetoothctl
# show コマンドで Name: dejima-kvm を確認
exit
```

### SPP (Serial Port Profile) の有効化

Bluetooth SDP に Serial Port サービスを登録します。

```bash
sudo sdptool add SP
```

> `sdptool` が見つからない場合は `sudo apt install bluez` でインストールしてください。

#### sdptool が「Failed」になる場合

BlueZ 5 ではデフォルトで SDP が無効になっています。以下で有効にします:

```bash
sudo nano /etc/systemd/system/bluetooth.target.wants/bluetooth.service
```

もしくは:

```bash
sudo nano /lib/systemd/system/bluetooth.service
```

`ExecStart` の行に `--compat` フラグを追加:

```ini
ExecStart=/usr/lib/bluetooth/bluetoothd --compat
```

反映:

```bash
sudo systemctl daemon-reload
sudo systemctl restart bluetooth
sudo sdptool add SP    # 再実行
```

### rfcomm 待ち受けの確認 (手動テスト)

```bash
sudo rfcomm listen /dev/rfcomm0 1
```

このコマンドは接続待ちでブロックします。
ホスト PC からペアリング・接続した後、`/dev/rfcomm0` が作成されます (セクション 3 参照)。
手動テストが成功したら Ctrl+C で停止してください。

---

## 3. ホスト PC 側: Bluetooth ペアリング

### macOS

> **重要**: macOS は Bluetooth のプライバシー保護により、RPi 側のスキャンには応答しません。
> 必ず **macOS 側から** RPi を見つけてペアリングを開始してください。

1. RPi 側で `discoverable on` を実行（**タイムアウト 180 秒** なので素早く進めてください）
2. **システム設定 → Bluetooth** を開く
3. RPi4 (例: `dejima-kvm`) が表示されたら「接続」をクリック
4. macOS にペアリング番号が表示される
5. RPi 側の `bluetoothctl` にも確認プロンプトが表示される:
   ```
   [agent] Confirm passkey 123456 (yes/no):
   ```
6. **両方の番号が一致していることを確認**し、macOS で「ペアリング」をクリック、RPi で `yes` と入力
7. ペアリング完了後、シリアルポートが自動作成される

> RPi の `bluetoothctl` プロンプトが `[ホスト名]>` に変わっている場合、
> `back` と入力してメインメニューに戻ってから `yes` を入力してください。

> RPi 側にプロンプトが表示されない場合は、先に macOS 側の「ペアリング」ボタンを押してください。
> その操作をトリガーとして RPi 側に確認プロンプトが表示されます。

デバイスパスの確認:

```bash
ls /dev/tty.*
# 例: /dev/tty.dejima-kvm-SerialPort
```

> macOS の Bluetooth シリアルポートは、RPi 側の SPP が有効かつ
> ペアリング完了・接続確立後に初めて出現します。
> ペアリング前には一覧に表示されません。

Dejima KVM アプリの設定画面でこのポートを選択してください。

### Windows

1. **設定 → Bluetooth とデバイス** を開く
2. 「デバイスの追加」→「Bluetooth」で RPi4 を選択
3. ペアリングコードを確認・承認
4. ペアリング完了後、**設定 → Bluetooth とデバイス → その他の Bluetooth 設定** → 「COM ポート」タブで割り当てられた COM ポートを確認

> 「送信」用の COM ポートを使用してください（「受信」ではありません）。

Dejima KVM アプリの設定画面でこの COM ポートを選択してください。

---

## 4. RPi4 側: systemd サービスの設定

### rfcomm 待ち受けサービスの作成

Bluetooth 接続を自動で受け付けるサービスを作成します。

```bash
sudo nano /etc/systemd/system/dejima-rfcomm.service
```

内容:

```ini
[Unit]
Description=Dejima RFCOMM Listen
After=bluetooth.target
Requires=bluetooth.target

[Service]
Type=simple
ExecStartPre=/usr/bin/sdptool add SP
ExecStart=/usr/bin/rfcomm listen /dev/rfcomm0 1
Restart=always
RestartSec=3
User=root

[Install]
WantedBy=multi-user.target
```

### dejima-kvm-rpi.service の変更

デバイスを `/dev/rfcomm0` に変更し、rfcomm サービスへの依存を追加します。

```bash
sudo nano /etc/systemd/system/dejima-kvm-rpi.service
```

変更後の内容:

```ini
[Unit]
Description=Dejima KVM HID Daemon (RPi)
After=graphical.target dejima-rfcomm.service
Requires=dejima-rfcomm.service

[Service]
Type=simple
Environment=WAYLAND_DISPLAY=wayland-0
Environment=XDG_RUNTIME_DIR=/run/user/1000
Environment=DISPLAY=:0
ExecStart=/usr/local/bin/dejima-kvm-daemon-rpi -device /dev/rfcomm0 -baud 115200
Restart=always
RestartSec=3
User=root

[Install]
WantedBy=graphical.target
```

> 有線 UART 版との違いは `-device /dev/rfcomm0` と `After`/`Requires` の依存関係のみです。

### サービスの有効化・起動

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now dejima-rfcomm
sudo systemctl enable --now dejima-kvm-rpi
```

---

## 5. 複数ホストからの接続

Bluetooth ペアリングは複数台のホスト PC と同時に保持できます。
ただし、シリアル接続 (`/dev/rfcomm0`) は **同時に 1 台のみ** です。

- PC-A が接続中 → PC-B は接続できない
- PC-A のアプリでシリアルポートを切断（またはアプリ終了）→ PC-B が接続可能になる

ホスト PC を切り替える際は、先に接続中のアプリでポートを閉じてください。
ペアリングの削除や再設定は不要です。

---

## 6. 自動ペアリング (信頼済みデバイス)

毎回手動でペアリング承認が必要にならないよう、ホスト PC を信頼済みデバイスとして登録します。

RPi4 上で:

```bash
sudo bluetoothctl
```

```
# ペアリング済みデバイスの一覧
paired-devices
# 例: Device AA:BB:CC:DD:EE:FF MyHostPC

# 信頼済みとしてマーク
trust AA:BB:CC:DD:EE:FF
```

`exit` で終了。以降は自動的に接続が受け入れられます。

---

## 7. 動作確認

### RPi4 側

```bash
# rfcomm リンクの状態確認
rfcomm show
# 出力例: rfcomm0: AA:BB:CC:DD:EE:FF channel 1 connected [tty-attached]

# サービスの状態確認
sudo systemctl status dejima-rfcomm
sudo systemctl status dejima-kvm-rpi

# デーモンのログ確認
journalctl -u dejima-kvm-rpi -f
```

### ホスト PC 側

1. Dejima KVM アプリを起動
2. 設定画面で Bluetooth シリアルポートを選択
3. キーボード・マウス操作が転送されることを確認
4. クリップボード同期 (Cmd+V / Ctrl+V) が動作することを確認

### 診断モード

```bash
sudo /usr/local/bin/dejima-kvm-daemon-rpi -device /dev/rfcomm0 -diag
```

`/dev/rfcomm0` が OK と表示されれば正常です。

---

## 8. トラブルシューティング

### `power on` が失敗する

`bluetoothctl` で `power on` が `Failed to set power on` になる場合:

```bash
# 1. rfkill でブロックされていないか確認
rfkill list
# Soft blocked: yes の場合:
sudo rfkill unblock bluetooth

# 2. config.txt で Bluetooth が無効化されていないか確認
grep disable-bt /boot/firmware/config.txt
# dtoverlay=disable-bt があれば削除して再起動

# 3. unblock 後に再度 power on
sudo bluetoothctl
# power on
```

### ペアリングできない

```bash
# Bluetooth サービスが動作しているか確認
systemctl is-active bluetooth

# Bluetooth アダプタの状態確認
hciconfig
# hci0 が DOWN の場合:
sudo hciconfig hci0 up

# discoverable モードになっているか確認
sudo bluetoothctl
# show コマンドで Discoverable: yes を確認
```

### macOS から RPi が見つからない

RPi 側の Discoverable にはタイムアウト (デフォルト 180 秒) があります。

```bash
# RPi 側で discoverable を再設定
sudo bluetoothctl
discoverable off
discoverable on
exit
```

再設定後すぐに macOS の **システム設定 → Bluetooth** を開き直してください。
macOS はプライバシー保護のため RPi 側のスキャンには応答しないので、
必ず macOS 側から RPi を探す必要があります。

### /dev/rfcomm0 が見つからない

ホスト PC からの Bluetooth 接続が確立されていない状態です。

```bash
# rfcomm listen が動作しているか確認
sudo systemctl status dejima-rfcomm

# 手動で listen して確認
sudo rfcomm listen /dev/rfcomm0 1
```

ホスト PC 側から再接続してください。

### rfcomm listen が「Address already in use」になる

前回の rfcomm プロセスが残っている可能性があります。

```bash
# 残留プロセスを停止
sudo killall rfcomm

# bluetooth サービスを再起動
sudo systemctl restart bluetooth

# 再度 listen
sudo rfcomm listen /dev/rfcomm0 1
```

### sdptool add SP が失敗する

BlueZ の互換モードが有効になっていない可能性があります。
セクション 2 の「sdptool が Failed になる場合」を参照してください。

### 接続が頻繁に切れる

Bluetooth の電波干渉や電源管理が原因の可能性があります。

```bash
# RPi4 の WiFi/BT 共存設定 (干渉軽減)
# /boot/firmware/config.txt に追加:
# dtoverlay=disable-wifi   ← WiFi を使わない場合は無効にすると安定する

# Bluetooth の電源管理を無効化
sudo nano /etc/bluetooth/main.conf
# [Policy] セクションに以下を追加:
# AutoEnable=true
```

### 遅延が大きい

Bluetooth SPP の遅延は有線 UART より大きくなります（通常 10〜30ms 増加）。
マウスの応答性が気になる場合は、以下を試してください:

- RPi4 とホスト PC の距離を近づける（推奨 3m 以内）
- WiFi の 2.4GHz 帯と干渉していないか確認
- 有線 UART に戻す（`-device /dev/ttyAMA0`）

### macOS で SPP が認識されない (シリアルポートが作成されない)

macOS は Bluetooth の SDP (サービス発見) 結果を plist にキャッシュします。
RPi 側で SPP が SDP に登録される**前**にペアリングした場合、「SPP なし」の状態がキャッシュされ、
その後 SPP を登録して再ペアリングしてもシリアルポートが作成されないことがあります。

この問題は特に RPi5 で初回セットアップ時に発生しやすいです（RPi5 固有の問題ではなく macOS 側のキャッシュ汚染が原因）。

**解決手順**:

```bash
# 1. ペアリングを削除
blueutil --unpair <RPi の MAC アドレス>

# 2. Bluetooth plist キャッシュを完全削除
#    ※ 他のデバイス (マウス等) のペアリングもリセットされるため、有線マウスを用意してください
sudo rm -f /Library/Preferences/com.apple.Bluetooth.plist
rm -f ~/Library/Preferences/ByHost/com.apple.Bluetooth.*.plist
sudo rm -f /private/var/root/Library/Preferences/com.apple.bluetoothd.plist

# 3. Bluetooth デーモンを再起動
sudo pkill bluetoothd

# 4. デバイスファイルが消えたことを確認
ls /dev/tty.*<デバイス名>* /dev/cu.*<デバイス名>* 2>&1
# "No such file or directory" であること (消えていなければ Mac を再起動)
```

その後、RPi 側で SPP が登録されていることを確認してから (`sudo sdptool browse local | grep -A 10 "Serial Port"`)、
セクション 3 の手順に従ってクリーンな状態から再ペアリングしてください。

> `blueutil` が未インストールの場合: `brew install blueutil`

### 有線 UART に戻したい

1. このドキュメントで追加したサービスを無効化:
   ```bash
   sudo systemctl stop dejima-rfcomm
   sudo systemctl disable dejima-rfcomm
   sudo rm /etc/systemd/system/dejima-rfcomm.service
   ```

2. [有線 UART セットアップガイド](setup-rpi4.md) に従って再設定してください。

---

## 9. 有線 UART との比較

| 項目 | 有線 UART (FT232) | Bluetooth SPP |
|------|-------------------|---------------|
| ケーブル | FT232 + GPIO 配線が必要 | 不要 |
| セットアップ | config.txt 編集 + BT 無効化 | BT ペアリング + rfcomm |
| 遅延 | 最小限 | +10〜30ms 程度 |
| 安定性 | 非常に高い | 電波環境に依存 |
| 到達距離 | ケーブル長依存 | 〜10m (Class 2) |
| 同時利用 | BT 無効化が必要 | BT で通信 |
