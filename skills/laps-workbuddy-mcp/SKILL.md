---
name: laps-workbuddy-mcp
description: 安装、配置和诊断 LAPS WorkBuddy 连接器，使 WorkBuddy 能以业务语言操作 APS 并展示正式甘特图。用户提到 WorkBuddy、MCP、连接器或排产图附件时使用。
---

# WorkBuddy 连接器

先阅读 [命令手册](references/commands.md)。其中包含安装、APS 地址设置、同账号登录、WorkBuddy 配置、信任和诊断的完整步骤；不要探测命令帮助。

连接器使用本机已安装的 LAPS 程序、当前系统账号的登录状态和已配置的 APS 地址。它不会要求用户提供令牌，也不会打开第二次登录。排产查询和试算会默认附上本地生成的正式 HTML 甘特图；不得用手写摘要页面代替。

连接器写入业务数据前必须先说明影响并取得确认。若连接不可用，按“地址未设置、需要登录、权限不足、版本过旧或 WorkBuddy 尚未信任连接器”的业务顺序引导处理，不转述技术异常。
