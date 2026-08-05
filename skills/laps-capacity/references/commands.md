# 完整命令

## 产能计划与能力配置

```sh
laps-cli capacity resolved
laps-cli capacity list --resource categories|plans|profiles|capabilities|pools|concurrency-rules|exclusion-groups
laps-cli capacity get --resource RESOURCE --id ID
laps-cli capacity create --resource RESOURCE --file data.json
laps-cli capacity update --resource RESOURCE --id ID --file data.json
laps-cli capacity delete --resource RESOURCE --id ID --yes
laps-cli capacity profiles get --factory-id FACTORY_ID
laps-cli capacity profiles apply --factory-id FACTORY_ID --file profiles.json
laps-cli capacity validate --plan-id PLAN_ID
laps-cli capacity publish --plan-id PLAN_ID
```

`RESOURCE` 只能为 `categories`、`plans`、`profiles`、`capabilities`、`pools`、`concurrency-rules`、`exclusion-groups`。简单数据可用重复 `--set field=value` 代替文件；文件和 `--set` 不可混用。发布前必须先校验，并向用户说明生效的计划版本。

## 产能导入

```sh
laps-cli capacity import template --output capacity-template.xlsx
laps-cli capacity import preview --file capacity.xlsx --plan-code CODE --plan-name NAME --period-start YYYY-MM-DD --period-end YYYY-MM-DD [--version 1]
laps-cli capacity import apply --file capacity.xlsx --plan-code CODE --plan-name NAME --period-start YYYY-MM-DD --period-end YYYY-MM-DD [--version 1]
```

JSON 导入只需 `--file capacity.json`。Excel 预检和提交均要求计划基本信息。旧兼容方式：`capacity import --file capacity.json --preview=true|false`。

## 日期产能日历

```sh
laps-cli capacity calendar days --resource-id TEAM_ID --start-date YYYY-MM-DD --end-date YYYY-MM-DD
laps-cli capacity calendar range set --resource-id TEAM_ID --start-date YYYY-MM-DD --end-date YYYY-MM-DD --daily-output N [--reason TEXT]
laps-cli capacity calendar range reset --resource-id TEAM_ID --start-date YYYY-MM-DD --end-date YYYY-MM-DD
laps-cli capacity calendar category-days --resource-id TEAM_ID --category-id CATEGORY_ID --start-date YYYY-MM-DD --end-date YYYY-MM-DD
laps-cli capacity calendar category-range set --resource-id TEAM_ID --category-id CATEGORY_ID --start-date YYYY-MM-DD --end-date YYYY-MM-DD --daily-output N [--reason TEXT]
laps-cli capacity calendar category-range reset --resource-id TEAM_ID --category-id CATEGORY_ID --start-date YYYY-MM-DD --end-date YYYY-MM-DD
laps-cli capacity calendar history [--limit 200]
```

`set` 的日产能可为 0；`reset` 恢复默认产能。每次修改前说明资源、品类、日期范围、日产能和原因。

## 品类日期产能导入

```sh
laps-cli capacity calendar category-import template --output category-capacity.xlsx
laps-cli capacity calendar category-import preview --file data.json|data.xlsx|data.xls
laps-cli capacity calendar category-import apply --file data.json|data.xlsx|data.xls
```

标准输入仅支持 JSON：`--file -`。模板、预检和提交均保留中文/英文表头兼容；预检不写入数据。
