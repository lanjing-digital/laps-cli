# 完整命令

## 补充预计到货

```sh
laps-cli procurement preview --file procurement.json
laps-cli procurement apply --file procurement.json --confirm
```

JSON 可用 `--file -` 从标准输入提供，或重复 `--set`，两种方式互斥。结构如下：

```json
{"fileName":"agent-arrivals.json","rows":[{"sourceRow":1,"analysisId":"真实分析ID","orderId":"真实订单ID","materialId":"真实物料ID","materialCode":"M001","purchaseQuantity":100,"expectedAt":"2026-10-20"}]}
```

每行 sourceRow、analysisId、orderId、materialId、materialCode 必填，purchaseQuantity、expectedAt 按原采购导入规则处理。先读取分析中的真实ID和缺料依据，preview 展示 create/update/cancel/ignore 与错误警告；用户确认后用同一输入 apply。预计到货不是实际收货或已齐套，保存后读取原分析核对。两步都需要基础资料写权限，apply沿原事务和审计，不自行写库存。

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
