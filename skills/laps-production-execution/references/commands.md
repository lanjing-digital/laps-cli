# 生产执行命令
默认JSON，当前用户权限服务端重核。FILE均JSON，支持 --file - 或重复 --set，不能混用；写入需要 --confirm。
```sh
laps-cli execution progress list [--target-type TYPE] [--status STATUS] [--resource-id ID]
laps-cli execution reports list [--target-type TYPE] [--target-id ID]
laps-cli execution reports create --file report.json --confirm
laps-cli execution usage preview --target-type schedule|operation_sub_lot --target-id ID --output-quantity N
laps-cli execution settings get
laps-cli execution settings update --file settings.json --confirm
laps-cli execution bootstrap preview
laps-cli execution bootstrap apply --file bootstrap.json --confirm
laps-cli execution replans list
laps-cli execution replans get --id CASE_ID
laps-cli execution replans preview --id CASE_ID --file candidate.json
laps-cli execution replans apply --id CASE_ID --file apply.json --confirm
laps-cli execution replans dismiss --id CASE_ID --confirm
```
report.json: targetType=schedule|operation_sub_lot、targetId、eventType=start|quantity|complete|correction|reopen、idempotencyKey；可选goodQty、defectQty、occurredAt、note、productionCondition=normal|material_shortage|downtime|changeover|unknown、correctionOfEventId、materialUsage。quantity表示增量数量，correction关联原报工事件，complete明确确认完工，reopen重开；实际合法数量与状态由原服务核验。operatorId由服务端当前身份填写。相同事件重试保留原幂等键。
settings.json: riskThresholdMinutes、staleThresholdMinutes、recentIntervalCount、minimumRateWindowMinutes、impactHorizonDays，先读取现值并仅调整用户要求的部分，完整提交原设置。
系统已有用料联动时，quantity/complete报工可能必须带materialUsage。先以本次产量（合格加不良数量）做usage preview，读取enabled、version、lines、可见warehouses及issues；这是原只读用料预览，不占库存、不提交领料。enabled=true时逐项核对实际用料，issues需先处理，不猜换算与替代料。materialUsage结构：warehouseId、expectedVersion（本次预览version）、lines数组；每行key、materialCode、quantity、unit，可选lossQuantity、batchNo、binCode、reservationAllocationId、sourceDocumentLineId、reason。使用预览的真实key、单位与仓库ID，不照搬建议数量当实际消耗；确认用料会随原报工联动正式业务。enabled=false时不伪造用料绑定。每次报工重新读取，不能复用旧指纹。
bootstrap.json: {"records":[{"targetType":"schedule","targetId":"真实ID","status":"not_started"}]}，status=not_started|in_progress|completed。仅按用户核实或正式结果确认，不按时间推测。
candidate.json: {"candidateId":"真实候选ID"}；apply.json另带 previewToken。查询CASE详情确认候选可用后预览；凭证过期或数据变化重新预览。dismiss是保存关闭处理，不等同于风险解除。
列表返回原records与风险状态；报工成功返回事件和重排关联，apply返回正式变更。每一步成功后查询对应进度/风险/记录核对，不以聊天回复证明实际完成。
