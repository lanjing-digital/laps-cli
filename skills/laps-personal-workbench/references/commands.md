# 个人关注命令
默认 JSON，使用当前 CLI 登录用户。所有写操作 --confirm；JSON文件与重复 --set 互斥，--file - 只用于JSON标准输入。
```sh
laps-cli personal interests list
laps-cli personal interests replace --file interests.json --confirm
laps-cli personal workbench get
laps-cli personal workbench facts --id FEATURE_ID
laps-cli personal workbench guidance --id FEATURE_ID
laps-cli personal workbench refresh --confirm
```
interests list 返回 version、automaticTour、features（含权限、参数白名单与原路由）、interests（含featureId、pinned、parameters、rank）。根据该结果构造：
```json
{"version":0,"interests":[{"featureId":"orders","pinned":false,"parameters":{}}],"automaticTour":false}
```
version 必须换成最新返回值；整个 interests 数组替换现有设置，空数组取消所有关注，未列出项目会被抑制。pinned与automaticTour可省略，显式false会到达后端。parameters 仅接受该功能 parameterKeys 中的有限标量值；不允许任意URL、脚本、SQL、隐私表单或用户身份。

workbench get 返回原首页卡片、分析状态与来源时间，不复制业务查询算法。facts 查询某个允许的功能，guidance 读取本人记忆及人工关注范围。当前参数通过个人关注设置定义。refresh 返回提交状态，后续 get 才能核对实际分析结果。

第三方追加/合并关注使用 personal sync preview|apply：JSON含 schemaVersion=1、agent、interests 数组。先预览最终关注与差异，再对同一数据确认导入；不上传网页观察事件。冲突、权限撤销、参数不合法时停止并重新读取。
