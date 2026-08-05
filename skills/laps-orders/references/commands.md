# 完整命令

## 查询与导出

```sh
laps-cli orders list [--status pending|scheduled|all] [--query TEXT] [--limit 100] [--page-token TOKEN]
laps-cli orders get --id ORDER_ID
laps-cli orders export --output orders.csv [--format csv|json]
```

## 新建、修改、删除

```sh
laps-cli orders create --set sequence-no=FG-26001 --set style-no=SUIT-01 --set style-desc="西装" --set order-qty=800 --set delivery-date=2026-08-20
laps-cli orders create --file order.json
laps-cli orders update --id ORDER_ID --set customer-name="客户 A" --set order-qty=900
laps-cli orders update --id ORDER_ID --file order.json
laps-cli orders delete --id ORDER_ID --yes
```

支持字段：`sequence-no`、`style-no`、`style-desc`、`style-type`、`category-id`、`customer-name`、`order-qty`、`delivery-date`、`sam-value`、`factory`、`order-status`。`--set` 的键使用 kebab-case；数字、布尔值和 JSON 数组可直接写值。`--file JSON|-` 与业务字段 `--set` 不能同时使用。

## 批量导入

```sh
laps-cli orders import template --output orders-template.xlsx
laps-cli orders import preview --file orders.xlsx [--mode create|upsert]
laps-cli orders import apply --file orders.xlsx [--mode create|upsert]
```

JSON 文件为 `{"orders":[...],"mode":"create"}`；Excel 通过 `--mode` 传递。默认 `create`，以 `sequenceNo` 新建；`upsert` 按 `sequenceNo` 覆盖。文件内重复工单号或数据库有多个匹配项时预检失败。必须先预检并向用户说明创建/更新数量后再提交。
