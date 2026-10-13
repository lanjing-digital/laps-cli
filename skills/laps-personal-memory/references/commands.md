# 个人记忆命令
以下命令默认 JSON，复用当前 CLI 登录和服务地址；可用 --base-url 临时指定目标。写入需要 --confirm；删除/清空另需 --yes。嵌套 JSON 使用 --file path.json 或 --file -（标准输入）；简单字段可重复 --set field=value，两种输入互斥。无输出文件选项，按需由调用方安全保管结果。
```sh
laps-cli personal profile get
laps-cli personal memories list [--category preference|parameter|information|progress]
laps-cli personal memories get --id ID
laps-cli personal memories create --file memory.json --confirm
laps-cli personal memories update --id ID --file memory.json --confirm
laps-cli personal memories delete --id ID --version N --confirm --yes
laps-cli personal memories clear --confirm --yes
laps-cli personal preferences list
laps-cli personal preferences create --file preference.json --confirm
laps-cli personal preferences update --id ID --file preference.json --confirm
laps-cli personal learning get
laps-cli personal learning set --enabled=true|false --confirm
laps-cli personal learning initialize --confirm
laps-cli personal learning retry --confirm
laps-cli personal sync preview --file external.json
laps-cli personal sync apply --file external.json --preview-token TOKEN --confirm
laps-cli conversations list
laps-cli conversations get --id ID
```
Memory JSON: title（非空，最长200字符）, content（非空，最多64KiB）, category（四种分类）, scope（非空，最长160字符）。修改另带 get 返回的 version（正整数）。preferences 自动限定 category=preference。禁止 userId、path、certainty、source 等输入字段。手工修正为已确认；个人记忆按用户保存到原 PostgreSQL。关闭学习传 false，不能省略 --enabled。initialize 提交本人历史学习，retry 重试失败任务，提交成功不等于提炼已完成。delete/clear沿原遗忘流程：取消旧学习任务、清理派生习惯证据、首页分析及未抑制的推测关注，保留人工关注和主动取消记录；应在用户确认前说明这一影响。

外部同步 JSON（schemaVersion 必须为1；agent 为1–80字符的字母/数字/下划线/点/横线；memories与interests至少一项）：
```json
{"schemaVersion":1,"agent":"third-party-agent","memories":[{"externalId":"daily-goal","title":"日常优化目标","category":"preference","scope":"daily","content":"默认使用加权延期","observedAt":"2026-10-13T08:00:00Z"}],"interests":[{"featureId":"orders","pinned":false,"parameters":{}}]}
```
externalId 是同一来源长期稳定的主题标识（1–160字符，同agent字符集）；observedAt 可选，不能用它证明真实业务完成。每次最多50条记忆、30项关注、512KiB。关注 featureId、parameters 从 personal interests list 返回的 features 与 parameterKeys 选择，权限由服务器重核。省略的固定设置与未提供关注保留，参数合并到原关注；没有自动删除或整档替换。导入内容不接收自行确认状态，不产生网页行为观测。

preview 返回 persisted=false、逐条 action=create|update|unchanged、内容、场景、最终关注与十分钟有效的 previewToken。凭证绑定用户、内容及个人配置版本，修改/遗忘/权限变化后重新预览。来源记录在 source，导入 certainty=recent 且受保护；手工修正后不能被后续同步覆盖。相同来源与内容重复导入返回无变更；已遗忘来源拒绝重新导入。apply 原子写入记忆和关注，任一失败全部回滚；仅发送文件不自动同步。

conversations list 为本人最近50条对话，单条消息遵循原历史上限，get 读取本人确切ID。普通附件保留元数据，图表遵循原展示规则。输出含私人文本，仅向用户授权目标分享。成功结果 success=true；persisted=true 表示已保存，false表示预览或无变化；API失败保留非零退出码，不降级到共享目录或浏览器存储。
