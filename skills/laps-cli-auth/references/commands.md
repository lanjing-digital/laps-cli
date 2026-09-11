# 完整命令

先设置企业 APS 地址，再以当前系统账号登录。地址优先级：命令 `--base-url`、环境变量 `SCHEDULING_API_BASE_URL`、安装时保存的地址。

```sh
laps-cli config set-server --url https://aps.example.com
laps-cli auth login [--base-url https://aps.example.com] [--no-browser]
laps-cli auth status [--base-url https://aps.example.com] [--local]
laps-cli auth logout [--base-url https://aps.example.com]
```

- `login` 默认打开浏览器复用当前系统的 APS 登录会话；`--no-browser` 仅在需要手动打开授权地址时使用。
- 地址可使用 HTTPS 域名或受信任网络中的 `http://IP:端口`。
- `status` 只检查状态，不刷新或改写凭据；默认验证远端账号，访问凭据到期时明确返回失败。`--local` 只读取已保存且仍可续期的会话，供 WorkBuddy 轮询使用，返回 `authenticated: true` 与 `refreshRequired`；远端撤销仍由业务请求验证。
- WorkBuddy 使用 `auth login --no-browser`，配置 `authWaitForExit: true` 保持授权回调进程，由 WorkBuddy 打开授权页。`unAuth` 配置映射 `auth logout`，没有新增一级命令。
- 登录失效：提示“请先完成账号登录后重试”。地址未设置：提示“请设置 APS 系统地址”。

## 安装与版本

```sh
npx --yes @lanjing-digital/laps-cli@latest install --non-interactive --server https://aps.example.com
laps-cli version --json
laps-cli update --check --json
laps-cli update [--source auto|npm|github] [--force] [--json]
```

安装不等待输入也不自动登录，无需 Go。`update --check` 只查询版本，不安装；普通命令读取 24 小时缓存并后台检查版本，JSON 的 `_notice` 和标准错误流包含提醒。自动化可设置 `LAPS_CLI_NO_UPDATE_NOTIFIER=1` 或 `LAPS_CLI_NO_SKILLS_NOTIFIER=1` 分别关闭提醒，显式 `update --check` 仍可用。CI 环境默认关闭自动提醒。

WorkBuddy 托管运行时通过 `npm install -g` 安装同一 npm 包，再执行 `laps-cli install --managed --non-interactive --no-skills --default-server https://lanjingshuzi.cn:3000` 预下载对应二进制。连接器目录自带业务 Skills，无需另装到全局技能目录。`--default-server` 只在尚未配置地址时生效；修改地址用 `config set-server`。
