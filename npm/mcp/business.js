import { configuredServerURL } from "../lib/launcher.js";
import { prepareInvocation } from "./operations.js";
import { LapsCommandError, runLapsCommand } from "./runner.js";

function recordCount(data) {
  if (!data || typeof data !== "object") return 0;
  if (Number.isFinite(data.total)) return data.total;
  if (Array.isArray(data.records)) return data.records.length;
  if (Array.isArray(data.items)) return data.items.length;
  if (Number.isFinite(data.count)) return data.count;
  return 0;
}

function operationLabel(domain, operation) {
  const labels = {
    orders: "生产订单", material_master: "物料与 BOM", material_readiness: "物料齐套", scheduling: "生产排程", capacity: "产能配置", master_data: "工厂与基础资料",
  };
  if (operation.includes("preview")) return `${labels[domain]}试算`;
  if (operation.includes("template")) return `${labels[domain]}模板准备`;
  return labels[domain] || "业务操作";
}

function failureMessage(error) {
  if (!(error instanceof LapsCommandError)) return { message: error.message, guidance: "请检查操作范围后重试。" };
  const content = `${error.stdout}\n${error.stderr}`.toLowerCase();
  if (error.message.includes("地址尚未设置") || content.includes("base-url") || content.includes("remote aps server")) return { message: "尚未设置 APS 系统地址。", guidance: "请先填写企业 APS 系统的域名或 IP 地址。" };
  if (content.includes("auth") || content.includes("oauth") || content.includes("401") || content.includes("credential")) return { message: "当前账号尚未完成登录或登录已失效。", guidance: "请在同一电脑账户中完成登录后重试。" };
  if (content.includes("403") || content.includes("permission") || content.includes("rbac") || content.includes("forbidden")) return { message: "当前账号没有执行此业务操作的权限。", guidance: "请联系管理员确认账号权限。" };
  if (content.includes("validation") || content.includes("invalid") || content.includes("required") || content.includes("duplicate")) return { message: "提交的数据不符合业务规则。", guidance: "请核对必填信息、日期、数量和重复编号后重试。" };
  return { message: "本次业务操作未完成。", guidance: "请核对目标对象和输入数据后重试。" };
}

export async function connectionStatus() {
  const baseURL = String(process.env.SCHEDULING_API_BASE_URL || "").trim() || await configuredServerURL();
  if (!baseURL) return { success: false, message: "尚未设置 APS 系统地址。", guidance: "请先设置企业 APS 系统的域名或 IP 地址。" };
  try {
    await runLapsCommand(["auth", "status"]);
    return { success: true, message: "APS 连接与账号登录状态已准备就绪。", saved: false, data: { ready: true } };
  } catch (error) {
    const failure = failureMessage(error);
    return { success: false, ...failure };
  }
}

export async function executeDomain(domain, input) {
  let invocation;
  try {
    invocation = await prepareInvocation(domain, input);
    const result = await runLapsCommand(invocation.args);
    const preview = input.operation.includes("preview");
    const count = recordCount(result.data);
    const saved = invocation.spec.mutates;
    const label = operationLabel(domain, input.operation);
    const scope = count > 0 ? `，涉及 ${count} 条记录` : "";
    const message = preview
      ? `${label}已完成${scope}，本次仅供确认，尚未写入系统。`
      : saved
        ? `${label}已保存${scope}。`
        : `${label}已查询完成${scope}。`;
    return { success: true, message, saved, preview, data: result.data, outputPath: input.outputPath };
  } catch (error) {
    const failure = failureMessage(error);
    return { success: false, ...failure };
  } finally {
    await invocation?.cleanup?.();
  }
}
