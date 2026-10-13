# 仓库命令
默认JSON；FILE仅JSON，支持 --file - 或重复 --set，不能混用；写入 --confirm，作废额外 --yes。--id 使用精确仓库/单据ID，不能传名称。
```sh
laps-cli warehouse warehouses list
laps-cli warehouse warehouses get --id WAREHOUSE_ID
laps-cli warehouse warehouses create --file warehouse.json --confirm
laps-cli warehouse warehouses update --id WAREHOUSE_ID --file warehouse.json --confirm
laps-cli warehouse warehouses assignments --id WAREHOUSE_ID --file assignments.json --confirm
laps-cli warehouse warehouses mode --id WAREHOUSE_ID --file mode.json --confirm
laps-cli warehouse users list
laps-cli warehouse inventory list [--warehouse-id ID] [--material-code CODE] [--batch-no NO] [--stock-status STATUS]
laps-cli warehouse inventory adjust --file adjustment.json --confirm
laps-cli warehouse inbounds list --warehouse-id ID
laps-cli warehouse documents list [--warehouse-id ID]
laps-cli warehouse documents create --file document.json --confirm
laps-cli warehouse documents post --id DOCUMENT_ID --confirm
laps-cli warehouse documents void --id DOCUMENT_ID --confirm --yes
laps-cli warehouse connectors save --file connector.json --confirm
laps-cli warehouse sync preview --file inventory-sync.json
laps-cli warehouse sync apply --file inventory-sync.json --confirm
laps-cli warehouse sync history
```
warehouse.json: code（创建必填）、name，原设置可选warehouseType、factoryId、address、enabled、readinessEnabled、allocationPriority，按服务规则验证。
assignments.json: primaryManagerId、collaboratorIds数组；users list 需仓库管理权限。
mode.json: dataSourceMode=internal|external_api、operationMode=document|direct，可选externalSystemId、externalWarehouseCode、takeover（切换须明确用户意图）。
adjustment.json: warehouseId、materialCode、stockStatus=qualified|pending_inspection|frozen|rejected、targetQuantity、reason、expectedVersion，可选batchNo、idempotencyKey。以查询返回的版本防并发覆盖。
document.json: documentType=opening|receipt|issue|transfer|stocktake|status_change|preparation、warehouseId、reason、lines数组。每行materialCode、quantity，可选batchNo、stockStatus、countedQuantity、fromStockStatus、toStockStatus、reservationAllocationId、materialInboundId；调拨另指定destinationWarehouseId，可选documentNo、occurredAt、orderId、reservationId。receipt使用待到货真实ID。创建仅为草稿，post才改变库存；void遵循原业务约束。
connector.json: code、name，可选systemType、config；不得把接口密钥放入聊天或Skills，使用受控配置途径。
inventory-sync.json: warehouseId、systemCode、syncMode=full|delta、snapshotAt、externalWarehouseCode、rows。每行externalRecordId、materialCode、stockStatus、quantity、lockedQuantity、availableQuantity，可选batchNo、productionDate、expiryDate、sourceUpdatedAt、deleted。full可能影响未提供行，先说明范围。preview无库存写入，apply沿原事务落库，返回原 counts 与syncJobId，随后查询核对。
