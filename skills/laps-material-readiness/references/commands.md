# 完整命令

```sh
laps-cli readiness status
laps-cli readiness schema
laps-cli readiness latest [--full=true|false]
laps-cli readiness get --analysis-id ANALYSIS_ID
laps-cli readiness analyze [--order-id ORDER_ID ...] [--source auto|builtin|external] [--persist=true|false]
laps-cli readiness analyze --file request.json
laps-cli readiness external import --file external-result.json
```

分析请求 JSON 可使用：

```json
{"orderIds":["order_1","order_2"],"source":"auto","persist":true}
```

外部结果文件必须符合 `readiness schema` 返回的业务字段结构。分析保存后可用分析 ID 或最新结果读取；只读试算使用 `"persist": false`。导入外部结果前先确认来源、有效时间和将影响的订单范围。
