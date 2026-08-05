# 完整命令

## 工厂与班组

```sh
laps-cli resources list [--include-inactive=true|false]
laps-cli resources get --factory-id FACTORY_ID
laps-cli resources apply --file factory-tree.json
laps-cli resources apply --factory-id FACTORY_ID --file factory-tree.json
laps-cli resources delete-factory --factory-id FACTORY_ID --yes
laps-cli resources delete-team --team-id TEAM_ID --yes
laps-cli resources batch-settings --factory-id FACTORY_ID [--factory-id FACTORY_ID ...] [--enabled true|false] [--ownership-type owned|outsourced] [--is-headquarters true|false]
laps-cli resources batch-settings --file batch-settings.json
```

工厂树、虚拟产线和复杂班组配置使用 `resources apply --file`。批量设置文件示例：

```json
{"factoryIds":["factory_1","factory_2"],"enabled":true,"ownershipType":"owned","isHeadquarters":false}
```

文件与批量设置标志不能混用；至少提供一个工厂 ID 和一个要调整的设置。

## 效率、日历和休假日期

```sh
laps-cli efficiencies list|get|create|update|delete [--id ID] [--file JSON|-] [--set field=value] [--yes]
laps-cli calendars list|get|create|update|delete [--id ID] [--file JSON|-] [--set field=value] [--yes]
laps-cli holidays list|get|create|update|delete [--id ID] [--file JSON|-] [--set field=value] [--yes]
laps-cli calendars bind --calendar-id CALENDAR_ID --team-id TEAM_ID
```

效率快捷字段：`team-id`、`team-text`、`product-type`、`efficiency`。日历快捷字段：`calendar-code`、`calendar-name`、`factory`、`production-line`。休假快捷字段：`calendar-id`、`holiday-date`、`weekday`、`label`。

创建和更新必须在 `--file` 与一个或多个 `--set` 间二选一；删除必须有精确 ID 与 `--yes`。
