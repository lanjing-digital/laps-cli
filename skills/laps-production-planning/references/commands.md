# 生产计划命令
默认JSON，读写均复用当前登录身份。以下 FILE 均为JSON，支持 --file - 或重复 --set field=value，不能混用；所有写操作显式 --confirm。
```sh
laps-cli planning status
laps-cli planning plans list [--from YYYY-MM-DD] [--to YYYY-MM-DD]
laps-cli planning plans get --id PLAN_ID
laps-cli planning plans save --file plan.json --confirm
laps-cli planning plans analyze --id PLAN_ID --file version.json --confirm
laps-cli planning plans confirm --id PLAN_ID --file version.json --confirm
laps-cli planning plans import-readiness --file stocking.json --confirm
laps-cli planning followups list [--plan-id PLAN_ID]
laps-cli planning followups save --file followup.json --confirm
laps-cli planning preparations options --plan-line-id LINE_ID
laps-cli planning preparations apply --file preparation.json --confirm
```
plan.json: name、periodStart/periodEnd（日期）、lines数组；每行 orderId、targetWeekStart、targetWeekEnd、quantity，可选id、requiredAt、note。编辑另带id和当前version；可选cadence=week|month、idempotencyKey。保存为草稿，不是自动发布排产。
version.json: {"version":1}，必须换成当前计划版本。
stocking.json: analysisId、orderIds数组、cadence=week|month，可选expectedAt、confirmPurchase。confirmPurchase=true 会确认采购相关变更，另向用户解释该影响。
followup.json: planId、materialId、quantity、expectedAt，可选id、planLineId、inboundId、owner、remark、confirm、version。编辑带当前version；confirm=false 是跟进草稿，true 确认跟进，不能据此称库存已到账。
preparation.json: planLineId、warehouseId、lines，每行reservationAllocationId、quantity，可选batchNo、binCode；可选reason、idempotencyKey。先用 options 获取真实分配ID与数量，apply 会生成正式备货单据，不能当只读操作。相同幂等键复用原业务结果，不换键重放。

list 返回原 records/plans 数据，get 返回计划详情与版本；状态/分析/确认按原业务服务解释。失败非零退出，不自行计算齐套或库存。
