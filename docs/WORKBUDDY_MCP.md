# 在 WorkBuddy 中使用 LAPS

`laps-mcp` 是 LAPS 的 WorkBuddy 自定义连接器。它沿用当前电脑账户已经设置的 APS 系统地址和登录状态，因此不需要安装 Go，也不需要复制令牌。

## 安装与首次连接

先向企业管理员确认 APS 系统地址。地址必须包含 `https://` 或 `http://`，例如 `https://aps.example.com` 或 `http://192.168.1.20:3000`。不要猜测地址，也不要默认使用本机地址。

```sh
npx --yes github:lanjing-digital/laps-cli install --server https://aps.example.com
laps-cli auth login
laps-mcp status
```

上述安装仅需要 Node.js 18 或更高版本。CLI 和连接器会自动下载适合 macOS、Linux 或 Windows 的程序。

## 配置 WorkBuddy

先查看将写入的配置：

```sh
laps-mcp workbuddy config --print
```

将输出内容合并到 `~/.workbuddy/mcp.json`，或者在确认后让 LAPS 自动完成。自动写入前会备份原文件：

```sh
laps-mcp workbuddy config --install --yes
```

重新加载 WorkBuddy 后，进入“自定义连接器”，找到 LAPS 并点击 **Trust（信任）**。连接器的 command 必须是本机安装器生成的绝对路径；自动配置会写入该路径。

## 使用与更新

连接器按订单、物料与 BOM、齐套、排产、产能和基础资料分别提供工具。试算不会保存，涉及新增、修改、发布或删除时，WorkBuddy 会要求先确认影响范围。

更新时执行：

```sh
laps-cli update
```

更新会同时替换 CLI、连接器和 skills，保留 APS 系统地址和当前用户登录资料。

## 常见情况

- “请先设置 APS 系统地址”：执行 `laps-cli config set-server --url 你的系统地址`。
- “请先完成账号登录”：在与 WorkBuddy 相同的电脑账户中执行 `laps-cli auth login`。
- “当前账号没有权限”：联系管理员为该账号开通对应业务权限。
- 连接器未出现：确认 WorkBuddy 已重新加载，并在连接器管理页面点击 Trust。

正常业务使用时，连接器只会说明业务结果、影响范围和下一步，不显示技术错误、请求地址或登录资料。需要排查时再明确提出“技术诊断”。
