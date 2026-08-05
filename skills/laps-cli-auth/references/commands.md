# 完整命令

先设置企业 APS 地址，再以当前系统账号登录。地址优先级：命令 `--base-url`、环境变量 `SCHEDULING_API_BASE_URL`、安装时保存的地址。

```sh
laps-cli config set-server --url https://aps.example.com
laps-cli auth login [--base-url https://aps.example.com] [--no-browser]
laps-cli auth status [--base-url https://aps.example.com]
laps-cli auth logout [--base-url https://aps.example.com]
```

- `login` 默认打开浏览器复用当前系统的 APS 登录会话；`--no-browser` 仅在需要手动打开授权地址时使用。
- 地址可使用 HTTPS 域名或受信任网络中的 `http://IP:端口`。
- `status` 成功后可继续执行业务操作。若账号权限刚调整，重新执行 `login` 以取得新权限。
- 登录失效：提示“请先完成账号登录后重试”。地址未设置：提示“请设置 APS 系统地址”。
