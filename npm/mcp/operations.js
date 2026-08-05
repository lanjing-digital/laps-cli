import { mkdtemp, rm, stat, writeFile } from "node:fs/promises";
import { randomUUID } from "node:crypto";
import os from "node:os";
import path from "node:path";

const scalar = (value) => ["string", "number", "boolean"].includes(typeof value);
const flag = (name) => `--${name}`;
const operation = (tokens, config = {}) => ({ tokens, mutates: false, file: false, output: false, outputRequired: false, options: [], fields: {}, ...config });
const crud = (root, fields = {}) => ({
  list: operation([root, "list"], { options: ["query", "status", "page", "limit", "page-token"] }),
  get: operation([root, "get"], { fields: { id: "id" }, required: ["id"] }),
  create: operation([root, "create"], { mutates: true, file: true }),
  update: operation([root, "update"], { mutates: true, file: true, fields: { id: "id" }, required: ["id"] }),
  delete: operation([root, "delete"], { mutates: true, destructive: true, fields: { id: "id" }, required: ["id"] }),
  ...fields,
});

export const domainOperations = {
  orders: {
    ...crud("orders"),
    export: operation(["orders", "export"], { output: true, outputRequired: true, options: ["format"] }),
    "import-template": operation(["orders", "import", "template"], { output: true, outputRequired: true }),
    "import-preview": operation(["orders", "import", "preview"], { file: true, options: ["mode"] }),
    "import-apply": operation(["orders", "import", "apply"], { mutates: true, file: true, options: ["mode"] }),
  },
  material_master: {
    "materials-summary": operation(["materials", "summary"]),
    ...Object.fromEntries(Object.entries(crud("materials")).map(([name, spec]) => [`materials-${name}`, spec])),
    ...Object.fromEntries(Object.entries(crud("boms")).map(([name, spec]) => [`boms-${name}`, spec])),
    "import-template": operation(["material-import", "template"], { output: true, outputRequired: true }),
    "import-history": operation(["material-import", "history"], { options: ["limit"] }),
    "import-preview": operation(["material-import", "preview"], { file: true }),
    "import-apply": operation(["material-import", "apply"], { mutates: true, file: true }),
  },
  material_readiness: {
    status: operation(["readiness", "status"]),
    latest: operation(["readiness", "latest"], { options: ["full"] }),
    get: operation(["readiness", "get"], { fields: { analysisId: "analysis-id" }, required: ["analysisId"] }),
    schema: operation(["readiness", "schema"]),
    analyze: operation(["readiness", "analyze"], { file: true, fields: { orderIds: "order-id" }, options: ["source", "persist"] }),
    "external-import": operation(["readiness", "external", "import"], { mutates: true, file: true }),
  },
  scheduling: {
    "teams-list": operation(["teams", "list"]),
    ...Object.fromEntries(Object.entries(crud("schedules")).map(([name, spec]) => [`schedules-${name}`, spec])),
    "schedules-list": operation(["schedules", "list"], { options: ["team-id", "order-id", "limit", "page-token", "format"], output: true, htmlPreview: true }),
    "schedules-lock": operation(["schedules", "lock"], { mutates: true, fields: { id: "id" }, required: ["id"], options: ["locked"] }),
    "schedules-apply": operation(["schedules", "apply"], { mutates: true, file: true }),
    "auto-preview": operation(["auto-schedule", "preview"], { fields: { orderIds: "order-id", resourceIds: "resource-id" }, options: ["plan-id", "ref-date", "capacity-mode", "prefer-same-product-resource", "replan-unstarted-orders", "readiness-enabled", "readiness-mode", "readiness-source", "readiness-max-age-minutes", "solver-mode", "include-candidate-plans", "format"], output: true, htmlPreview: true }),
    "auto-apply": operation(["auto-schedule", "apply"], { mutates: true, fields: { orderIds: "order-id", resourceIds: "resource-id" }, options: ["plan-id", "ref-date", "capacity-mode", "prefer-same-product-resource", "replan-unstarted-orders", "readiness-enabled", "readiness-mode", "readiness-source", "readiness-max-age-minutes", "format"], output: true }),
    "auto-apply-preview": operation(["auto-schedule", "apply"], { mutates: true, fields: { previewToken: "preview-token", candidateSolver: "candidate-solver" }, required: ["previewToken", "candidateSolver"], options: ["format"], output: true }),
    "move-order-preview": operation(["move", "order", "preview"], { fields: { orderId: "order-id", toTeamId: "to-team-id" }, required: ["orderId", "toTeamId"], options: ["format"], output: true, htmlPreview: true }),
    "move-order-apply": operation(["move", "order", "apply"], { mutates: true, fields: { orderId: "order-id", toTeamId: "to-team-id" }, required: ["orderId", "toTeamId"], options: ["format"], output: true }),
    "move-schedule-preview": operation(["move", "schedule", "preview"], { fields: { scheduleId: "schedule-id", toTeamId: "to-team-id" }, required: ["scheduleId", "toTeamId"], options: ["format"], output: true, htmlPreview: true }),
    "move-schedule-apply": operation(["move", "schedule", "apply"], { mutates: true, fields: { scheduleId: "schedule-id", toTeamId: "to-team-id" }, required: ["scheduleId", "toTeamId"], options: ["format"], output: true }),
  },
  capacity: {
    resolved: operation(["capacity", "resolved"]),
    list: operation(["capacity", "list"], { options: ["resource"] }),
    get: operation(["capacity", "get"], { fields: { id: "id" }, required: ["id"], options: ["resource"] }),
    create: operation(["capacity", "create"], { mutates: true, file: true, options: ["resource"] }),
    update: operation(["capacity", "update"], { mutates: true, file: true, fields: { id: "id" }, required: ["id"], options: ["resource"] }),
    delete: operation(["capacity", "delete"], { mutates: true, destructive: true, fields: { id: "id" }, required: ["id"], options: ["resource"] }),
    "profiles-get": operation(["capacity", "profiles", "get"], { fields: { factoryId: "factory-id" }, required: ["factoryId"] }),
    "profiles-apply": operation(["capacity", "profiles", "apply"], { mutates: true, file: true, fields: { factoryId: "factory-id" }, required: ["factoryId"] }),
    "import-template": operation(["capacity", "import", "template"], { output: true, outputRequired: true }),
    "import-preview": operation(["capacity", "import", "preview"], { file: true, options: ["plan-code", "plan-name", "period-start", "period-end", "version"] }),
    "import-apply": operation(["capacity", "import", "apply"], { mutates: true, file: true, options: ["plan-code", "plan-name", "period-start", "period-end", "version"] }),
    validate: operation(["capacity", "validate"], { fields: { planId: "plan-id" }, required: ["planId"] }),
    publish: operation(["capacity", "publish"], { mutates: true, fields: { planId: "plan-id" }, required: ["planId"] }),
    "calendar-days": operation(["capacity", "calendar", "days"], { fields: { resourceId: "resource-id" }, options: ["start-date", "end-date"] }),
    "calendar-category-days": operation(["capacity", "calendar", "category-days"], { fields: { resourceId: "resource-id", categoryId: "category-id" }, options: ["start-date", "end-date"] }),
    "calendar-history": operation(["capacity", "calendar", "history"], { options: ["limit"] }),
    "calendar-range-set": operation(["capacity", "calendar", "range", "set"], { mutates: true, fields: { resourceId: "resource-id" }, required: ["resourceId"], options: ["start-date", "end-date", "daily-output", "reason"] }),
    "calendar-range-reset": operation(["capacity", "calendar", "range", "reset"], { mutates: true, fields: { resourceId: "resource-id" }, required: ["resourceId"], options: ["start-date", "end-date"] }),
    "calendar-category-range-set": operation(["capacity", "calendar", "category-range", "set"], { mutates: true, fields: { resourceId: "resource-id", categoryId: "category-id" }, required: ["resourceId", "categoryId"], options: ["start-date", "end-date", "daily-output", "reason"] }),
    "calendar-category-range-reset": operation(["capacity", "calendar", "category-range", "reset"], { mutates: true, fields: { resourceId: "resource-id", categoryId: "category-id" }, required: ["resourceId", "categoryId"], options: ["start-date", "end-date"] }),
    "calendar-import-template": operation(["capacity", "calendar", "category-import", "template"], { output: true, outputRequired: true }),
    "calendar-import-preview": operation(["capacity", "calendar", "category-import", "preview"], { file: true }),
    "calendar-import-apply": operation(["capacity", "calendar", "category-import", "apply"], { mutates: true, file: true }),
  },
  master_data: {
    "resources-list": operation(["resources", "list"], { options: ["include-inactive"] }),
    "resources-get": operation(["resources", "get"], { fields: { factoryId: "factory-id" }, required: ["factoryId"] }),
    "resources-apply": operation(["resources", "apply"], { mutates: true, file: true, fields: { factoryId: "factory-id" } }),
    "resources-delete-factory": operation(["resources", "delete-factory"], { mutates: true, destructive: true, fields: { factoryId: "factory-id" }, required: ["factoryId"] }),
    "resources-delete-team": operation(["resources", "delete-team"], { mutates: true, destructive: true, fields: { teamId: "team-id" }, required: ["teamId"] }),
    "resources-batch-settings": operation(["resources", "batch-settings"], { mutates: true, file: true, fields: { factoryIds: "factory-id" }, options: ["enabled", "ownership-type", "is-headquarters"] }),
    ...Object.fromEntries(["efficiencies", "calendars", "holidays"].flatMap((root) => Object.entries(crud(root)).map(([name, spec]) => [`${root}-${name}`, spec]))),
    "calendars-bind": operation(["calendars", "bind"], { mutates: true, fields: { calendarId: "calendar-id", teamId: "team-id" }, required: ["calendarId", "teamId"] }),
  },
  scheduling_policy: {
    ...crud("scheduling-policy"),
    clone: operation(["scheduling-policy", "clone"], { mutates: true, fields: { id: "id" }, required: ["id"] }),
    validate: operation(["scheduling-policy", "validate"], { mutates: true, fields: { id: "id" }, required: ["id"] }),
    publish: operation(["scheduling-policy", "publish"], { mutates: true, fields: { id: "id" }, required: ["id"] }),
    "runs-list": operation(["scheduling-policy", "runs", "list"], { options: ["mode", "status", "solver", "from", "to", "limit", "page-token"] }),
    "runs-get": operation(["scheduling-policy", "runs", "get"], { fields: { runId: "run-id" }, required: ["runId"] }),
  },
};

export const domainNames = Object.keys(domainOperations);
export const operationNames = (domain) => Object.keys(domainOperations[domain] || {});

function valueArgs(name, value) {
  if (Array.isArray(value)) return value.flatMap((item) => valueArgs(name, item));
  if (!scalar(value)) throw new Error(`选项 ${name} 的值必须是文本、数字或 true/false`);
  return [flag(name), String(value)];
}

async function validateInputFile(filePath) {
  const extension = path.extname(filePath).toLowerCase();
  if (![".json", ".xlsx", ".xls"].includes(extension)) throw new Error("导入文件仅支持 JSON、XLSX 或 XLS 格式");
  const info = await stat(filePath);
  if (!info.isFile()) throw new Error("请选择一个可读取的文件");
}

export async function prepareInvocation(domain, input) {
  const spec = domainOperations[domain]?.[input.operation];
  if (!spec) throw new Error("不支持此业务操作");
  if (spec.mutates && input.confirm !== true) throw new Error("此操作会修改系统数据。请先向用户说明影响并取得确认后重试。");
  if (spec.destructive && input.yes !== true) throw new Error("删除前需要再次确认目标对象，随后以明确的删除确认重试。");
  for (const key of spec.required || []) {
    const value = input[key];
    if (value === undefined || value === "" || (Array.isArray(value) && value.length === 0)) throw new Error("缺少完成此业务操作所需的对象标识");
  }
  if ((input.filePath || input.payload !== undefined) && !spec.file) throw new Error("此业务操作不接受导入数据");
  if (input.filePath && input.payload !== undefined) throw new Error("请在文件和结构化数据中二选一");
  if (input.outputPath && !spec.output) throw new Error("此业务操作不会生成可下载文件");
  if (spec.outputRequired && !input.outputPath) throw new Error("请提供保存模板或导出结果的位置");

  if (input.format !== undefined && input.options?.format !== undefined) throw new Error("请只选择一种展示方式");
  const requestedFormat = input.format ?? input.options?.format;
  const useDefaultHtmlPreview = spec.htmlPreview && (requestedFormat === undefined || requestedFormat === "html") && !input.outputPath;
  const args = [...spec.tokens];
  for (const [field, optionName] of Object.entries(spec.fields)) {
    const value = input[field];
    if (value !== undefined && value !== "") args.push(...valueArgs(optionName, value));
  }
  for (const [name, value] of Object.entries(input.options || {})) {
    if (!spec.options.includes(name)) throw new Error("包含当前业务操作不支持的设置项");
    if (name === "format") continue;
    args.push(...valueArgs(name, value));
  }
  if (requestedFormat !== undefined) {
    if (!["json", "timeline", "svg", "html"].includes(requestedFormat)) throw new Error("展示格式不正确");
    if (!spec.options.includes("format")) throw new Error("此业务操作不支持该展示方式");
    if (!useDefaultHtmlPreview) args.push("--format", requestedFormat);
  }
  if (input.outputPath) args.push("--output", input.outputPath);

  let temporaryDirectory = "";
  if (input.payload !== undefined || useDefaultHtmlPreview) temporaryDirectory = await mkdtemp(path.join(os.tmpdir(), "laps-mcp-"));
  if (input.filePath) {
    await validateInputFile(input.filePath);
    args.push("--file", input.filePath);
  } else if (input.payload !== undefined) {
    const payloadPath = path.join(temporaryDirectory, "request.json");
    await writeFile(payloadPath, `${JSON.stringify(input.payload)}\n`, { mode: 0o600 });
    args.push("--file", payloadPath);
  }
  const previewArtifact = useDefaultHtmlPreview
    ? { path: path.join(temporaryDirectory, "schedule-preview.html"), uri: `laps://scheduling-preview/${randomUUID()}.html`, mimeType: "text/html" }
    : undefined;
  if (previewArtifact) args.push("--format", "html", "--output", previewArtifact.path);
  return { args, spec, previewArtifact, cleanup: async () => { if (temporaryDirectory) await rm(temporaryDirectory, { recursive: true, force: true }); } };
}
