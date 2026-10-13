# 个人上下文与新增 CLI 能力（待发布源码）

本轮仅本地实现。公开 npm/Release、远端后端和 WorkBuddy 客户端未更新；已发布版本尚不具备新增命令。CLI 独立构建，不访问数据库。后端复用原 PostgreSQL 表、用户 OAuth、权限、审计及正式业务服务，无新增迁移。

## 命令覆盖

| 入口 | 能力与手册 |
| --- | --- |
| `personal memories/preferences/learning/sync` | [记忆、习惯与跨 Agent 同步](../skills/laps-personal-memory/references/commands.md) |
| `personal interests/workbench` | [关注、事实与分析](../skills/laps-personal-workbench/references/commands.md) |
| `conversations` | 本人对话 list/get；不提供跨用户历史或注入伪造对话 |
| `procurement` | [预计到货预览和导入](../skills/laps-material-readiness/references/commands.md) |
| `planning` | [生产计划、跟进与备货](../skills/laps-production-planning/references/commands.md) |
| `warehouse` | [仓库、库存、单据与同步](../skills/laps-warehouse-inventory/references/commands.md) |
| `execution` | [生产进度、报工与重排](../skills/laps-production-execution/references/commands.md) |

网页 Agent 的原 laps_cli 支持新增命令及此前遗漏的 scheduling-policy，并沿原确认机制分类。MCP 使用固定命名工具及同一 CLI 登录。没有新聊天运行器、任意 HTTP/shell/SQL 代理或另一个个人档案库。

## 外部习惯同步

第三方 Agent 在当前登录用户授权范围内整理结构化 JSON，字段与示例见个人记忆手册。不上传整个聊天档案或凭据，不接收用户ID、角色、租户、确认状态及网页观察事件。

```sh
laps-cli personal interests list
laps-cli personal preferences list
laps-cli personal sync preview --file external.json
laps-cli personal sync apply --file external.json --preview-token '<预览返回值>' --confirm
laps-cli personal profile get
```

预览凭证不写普通日志，复用后端现有 AGENT_CONFIG_TOKEN 签名；缺失时显式拒绝，不新增客户端密钥。凭证绑定本人、内容、记忆代次/版本、关注版本和权限，到期或状态变化后重新预览。明确写入返回 persisted=true；预览或无变化为false。后台学习提交和实际提炼完成分开报告。

同一主题的 agent/externalId 保持稳定。外部记忆以 recent 和来源记录保存并受保护；不能自动成为已确认事实。相同内容去重，人工修正后停止覆盖；已遗忘来源不能自动复活。关注是合并，不删除未提供项目；原关注权限失效时拒绝合并，避免误删不可见设置。

同批记忆与关注在原事务内写入，失败全部回滚，没有新的同步影子表。原审计触发器记录脱敏变更，操作摘要不保存个人正文或导入包。关闭学习仍可手工管理并使用已有记忆；正式业务事实使用前重新查询，写入仍需确认。

## 验证与发布

CLI 运行 go test ./...、npm test 与 npm pack --dry-run。后端相关测试使用 test/cli/jest.config.cjs，启用与 Nest 一致的装饰器编译；记忆集成测试经 AGENT_MEMORY_TEST_ADMIN_URL 新建随机本地隔离库并清理，不允许远端数据库。请求映射测试不能替代真实 PostgreSQL 或第三方客户端验收。

后续发布按原独立 CLI 流程统一版本、同步分发源、构建二进制、打包技能并更新目标后端。此文档不授予发布或远端变更权限。
