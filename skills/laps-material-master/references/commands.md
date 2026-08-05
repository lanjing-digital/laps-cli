# 完整命令

## 物料和 BOM

```sh
laps-cli materials summary
laps-cli materials list [--query TEXT] [--page N] [--limit N]
laps-cli materials get --id MATERIAL_ID
laps-cli materials create|update --file material.json [--id MATERIAL_ID]
laps-cli materials create --set code=FAB-001 --set name="面料" --set unit=米
laps-cli materials update --id MATERIAL_ID --set disabled=false
laps-cli materials delete --id MATERIAL_ID --yes

laps-cli boms list [--query TEXT] [--page N] [--limit N]
laps-cli boms get --id BOM_ID
laps-cli boms create|update --file bom.json [--id BOM_ID]
laps-cli boms delete --id BOM_ID --yes
```

物料快捷字段：`code`、`name`、`specification`、`unit`、`disabled`、`default-warehouse-code`。复杂 BOM 必须使用 JSON 文件，例如：

```json
{"code":"BOM-SUIT-01","name":"西装 BOM","items":[{"materialId":"mat_1","quantity":2.4,"unit":"米"}]}
```

## 库存导入

```sh
laps-cli material-import template --output inventory-template.xlsx
laps-cli material-import history [--limit 50]
laps-cli material-import preview --file inventory.xlsx
laps-cli material-import apply --file inventory.xlsx
```

`--file` 支持 JSON、XLSX、XLS，JSON 可使用 `-` 从标准输入读取。导入采用覆盖更新、不会删除文件中未出现的数据；先预检，再说明影响并提交。
