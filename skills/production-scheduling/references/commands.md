# 完整命令

通用：`--base-url URL` 可临时覆盖系统地址；默认 JSON。排产图使用 `--format html --output /绝对路径/plan.html`，可选 `timeline`、`svg`。

## 查询

```sh
laps-cli orders list --status pending|scheduled|all [--query 文本] [--limit 100] [--page-token TOKEN]
laps-cli teams list
laps-cli schedules list [--team-id ID] [--order-id ID] [--limit 100] [--page-token TOKEN] [--format html] [--output FILE]
laps-cli schedules get --id SCHEDULE_ID
```

## 自动试排和确认提交

```sh
laps-cli auto-schedule preview \
  [--order-id ORDER_ID ...] \
  [--plan-id PLAN_ID | --ref-date YYYY-MM-DD] \
  [--resource-id TEAM_ID ...] \
  [--capacity-mode inherit|sam_efficiency|guaranteed_daily_output|category_daily_output] \
  [--prefer-same-product-resource inherit|true|false] \
  [--replan-unstarted-orders inherit|true|false] \
  [--readiness-enabled inherit|true|false] \
  [--readiness-mode inherit|ignore|warn|block] \
  [--readiness-source inherit|auto|builtin|external] \
  [--readiness-max-age-minutes N] \
  [--solver-mode inherit|heuristic|shadow-portfolio|cp-sat|ga|portfolio|hybrid-ga-cp] \
  [--include-candidate-plans=true|false] \
  [--format html] [--output FILE]

laps-cli auto-schedule apply --preview-token TOKEN \
  --candidate-solver heuristic|cp-sat|ga|hybrid-ga-cp \
  [--format html] [--output FILE]
```

- 未传订单 ID 时，系统按当前待排范围试算。未传计划 ID 或参考日期时，使用当天参考日期。
- `inherit` 完全继承已发布策略；试算默认返回所有可用候选方案。
- 试算结果的 `previewSession` 包含确认所需的 `token`、有效期、可选方案和推荐方案。该凭证仅在本次确认链路内部使用，不展示给业务用户。
- 先解释候选差异并让用户选择，再提交。凭证 30 分钟有效，只能成功使用一次；数据变化或超时必须重新试算。
- 历史兼容方式 `auto-schedule apply` 可携带旧试算参数重新计算，但不得在正式业务流程使用。

## 直接调整、锁定、移动

```sh
laps-cli schedules update --id SCHEDULE_ID --set allocated-qty=800 --set team-id=TEAM_ID
laps-cli schedules delete --id SCHEDULE_ID --yes
laps-cli schedules lock --id SCHEDULE_ID --locked true|false
laps-cli schedules apply --file changes.json
laps-cli move order preview --order-id ORDER_ID --to-team-id TEAM_ID [--format html] [--output FILE]
laps-cli move order apply --order-id ORDER_ID --to-team-id TEAM_ID
laps-cli move schedule preview --schedule-id SCHEDULE_ID --to-team-id TEAM_ID [--format html] [--output FILE]
laps-cli move schedule apply --schedule-id SCHEDULE_ID --to-team-id TEAM_ID
```

`changes.json` 使用 GUI 同等批量格式：

```json
{"adds":[],"updates":[{"recordId":"sched_1","expectedUpdatedAt":"2026-08-05T00:00:00.000Z","fields":{"分配数":800}}],"deletes":[],"replan":false}
```

锁定、移动、批量调整和删除均先取得用户确认。自动重排批量提交使用 `"replan": true`，并保留每条记录的 `expectedUpdatedAt`。
