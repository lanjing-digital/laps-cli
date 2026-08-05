# 完整命令

## 策略版本

```sh
laps-cli scheduling-policy list [--status draft|published|archived]
laps-cli scheduling-policy get --id POLICY_ID
laps-cli scheduling-policy create --file policy.json
laps-cli scheduling-policy update --id POLICY_ID --file policy.json
laps-cli scheduling-policy delete --id POLICY_ID --yes
laps-cli scheduling-policy clone --id POLICY_ID
laps-cli scheduling-policy validate --id POLICY_ID
laps-cli scheduling-policy publish --id POLICY_ID
```

创建和更新也可使用 `--file -` 从标准输入读取 JSON。删除只允许草稿版本；发布会自动归档其他已发布版本。推荐流程：查询已发布版本 → 克隆 → 编辑草稿 → 校验 → 说明影响 → 用户确认后发布。

策略文件至少包含编码、名称和完整 `definition`。可以先读取现有策略后复制修改：

```json
{
  "code":"apparel-main",
  "name":"服装主排产策略 v3",
  "version":3,
  "definition":{
    "schemaVersion":"1.2",
    "optimization":{"mode":"portfolio","objectiveProfile":"delivery-first","timeoutMs":60000,"randomSeedMode":"input-hash"}
  }
}
```

`definition` 还必须保留现有版本中的订单优先级、资源候选、拆单、生产默认值、齐套、渠道和约束配置。`timeoutMs` 范围为 1000–60000；模式只能是 `heuristic`、`shadow-portfolio`、`cp-sat`、`ga`、`portfolio`、`hybrid-ga-cp`。

## 求解历史

```sh
laps-cli scheduling-policy runs list [--mode MODE] [--status STATUS] [--solver SOLVER] [--from ISO_TIME] [--to ISO_TIME] [--limit 50] [--page-token TOKEN]
laps-cli scheduling-policy runs get --run-id RUN_ID
```

历史返回模式、当前采用方案、推荐方案、状态、耗时、候选业务指标和约束结果。需要拥有审计查看权限；无权限时请让管理员授予权限后重新登录。
