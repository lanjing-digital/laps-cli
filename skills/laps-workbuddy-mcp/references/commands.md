# 安装、配置与诊断

用户需要 Node.js 18+，不需要 Go 或编译环境。安装时必须提供企业 APS 域名或 IP：

```sh
npx --yes --force github:lanjing-digital/laps-cli install --server https://aps.example.com
laps-cli config set-server --url https://aps.example.com
laps-cli auth login
laps-mcp workbuddy config --print
laps-mcp workbuddy config --install --yes
```

`--install --yes` 会先备份 `~/.workbuddy/mcp.json`，再合并 `laps` 连接器，不覆盖已有连接器。随后在 WorkBuddy 的自定义连接器页面找到 LAPS 并选择 Trust。

更新：

```sh
laps-cli update
laps-cli update --source github
laps-cli update --source npm
```

连接器必须与 `laps-cli` 使用同一个操作系统账号和同一份 APS 登录状态。确认环境时调用 `laps_connection`；业务操作使用固定的 `laps_orders`、`laps_material_master`、`laps_material_readiness`、`laps_scheduling`、`laps_capacity`、`laps_master_data`、`laps_scheduling_policy` 工具。

排产试算不传展示格式或输出位置时，会返回正式 HTML 甘特图资源。连接器不能渲染图时，先让用户确认已 Trust 连接器和已更新版本；必要时输出官方 SVG 文件，不制作替代摘要页面。
