# udco2s-exporter
IODATA UD-CO2Sの出力をPrometheusで引っこ抜くやつ


# usage

## 想定環境
* Ubuntu 22.04
* Docker 20.10.23
* Go 1.20

## udevルールの投入

USBシリアルデバイス名を固定する+誰でも読み書きできるようなパーミッションを適用するためのudevルールを投入する

```bash
$ sudo cp udev/99-udco2s.rules /etc/udev/rules.d/
$ sudo udevadm control --reload-rules && sudo udevadm trigger
$ ls -la /dev/ttyUDCO2S
lrwxrwxrwx 1 root root 7  2月 11 23:28 /dev/ttyUDCO2S -> ttyACM0
```

`/dev/ttyUDCO2S` が生えてればOK

## Dockerを使う場合

```bash
$ docker pull misodengaku/udco2s-exporter:latest

# 直接動かす場合
$ docker run -e TTY=/dev/ttyUDCO2S --device=/dev/ttyUDCO2S -p 127.0.0.1:9999:9999 misodengaku/udco2s-exporter

# systemd経由でコンテナを自動起動させる場合
$ sudo cp systemd/udco2s-exporter-docker.service /etc/systemd/system
$ sudo systemctl daemon-reload
$ sudo systemctl enable udco2s-exporter-docker
$ sudo systemctl start udco2s-exporter-docker
```

## ホスト上で直接動かす場合

```bash
$ go build && cp udco2s-exporter /opt

# 何らかのエディタで systemd/udco2s-exporter.service.example を編集し、パラメータを自分の環境に合わせる
$ nano systemd/udco2s-exporter.service.example
$ sudo cp systemd/udco2s-exporter.service /etc/systemd/system
$ sudo systemctl daemon-reload
$ sudo systemctl start udco2s-exporter
```

## Prometheusの設定
`/etc/prometheus/prometheus.yml` あたりに設定を追加する。ターゲットホスト名は適宜調整すること。

```yaml
scrape_configs:
  - job_name: udco2s
    scrape_interval: 10s
    static_configs:
      - targets: ['localhost:9999']
```
