# wi-sun-exporter
MB-RL7023-11/DSSの出力をPrometheusで引っこ抜くやつ


# usage

## 想定環境
* Debianのインストールされた秋月謎SoCボード
  * https://qiita.com/chibiegg/items/4b1b70a5ba09c4a52a12 に従いインストールされた環境を想定
  * [Wi-SUNモジュール安定動作対応](https://github.com/bakueikozo/buildroot_am3352_aki/issues/1#issuecomment-1496027634) で安定動作できる環境が整えてあること
* Go 1.20.3


## build

```bash
$ go build && sudo cp wi-sun-exporter /opt

# 何らかのエディタで systemd/wi-sun-exporter.service.example を編集し、パラメータを自分の環境に合わせる
$ nano systemd/wi-sun-exporter.service.example
$ sudo cp systemd/wi-sun-exporter.service.example /etc/systemd/system/wi-sun-exporter.service
$ sudo systemctl daemon-reload
$ sudo systemctl start wi-sun-exporter
```

## Prometheusの設定
`/etc/prometheus/prometheus.yml` あたりに設定を追加する。ターゲットホスト名は適宜調整すること。

```yaml
scrape_configs:
  - job_name: wi-sun
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:9000']
```
