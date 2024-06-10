# Fast.com Speedtest

## System Service

`/etc/systemd/system/fast-speedtest.service`
```
[Unit]
Description=Fast.com speedtest
After=network.target

[Service]
ExecStart=/usr/local/bin/fast-speedtest_Linux_x86_64
Restart=always
Environment="PARALLEL_CONNECTIONS=2"

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable fast-speedtest
sudo systemctl start fast-speedtest
systemctl status fast-speedtest.service
```