import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";
import { connectionStatus, executeDomain } from "./business.js";
import { domainNames, operationNames } from "./operations.js";

const scalar = z.union([z.string(), z.number(), z.boolean()]);
const inputShape = (domain) => ({
  operation: z.enum(operationNames(domain)),
  id: z.string().min(1).optional(),
  orderId: z.string().min(1).optional(),
  orderIds: z.array(z.string().min(1)).optional(),
  scheduleId: z.string().min(1).optional(),
  toTeamId: z.string().min(1).optional(),
  teamId: z.string().min(1).optional(),
  factoryId: z.string().min(1).optional(),
  factoryIds: z.array(z.string().min(1)).optional(),
  resourceId: z.string().min(1).optional(),
  resourceIds: z.array(z.string().min(1)).optional(),
  categoryId: z.string().min(1).optional(),
  calendarId: z.string().min(1).optional(),
  analysisId: z.string().min(1).optional(),
  planId: z.string().min(1).optional(),
  options: z.record(scalar.or(z.array(scalar))).optional(),
  payload: z.unknown().optional(),
  filePath: z.string().min(1).optional(),
  outputPath: z.string().min(1).optional(),
  format: z.enum(["json", "timeline", "svg", "html"]).optional(),
  confirm: z.boolean().optional(),
  yes: z.boolean().optional(),
});

const outputShape = z.object({
  success: z.boolean(),
  message: z.string(),
  guidance: z.string().optional(),
  saved: z.boolean().optional(),
  preview: z.boolean().optional(),
  data: z.unknown().optional(),
  outputPath: z.string().optional(),
});

function registerDomainTool(server, domain) {
  server.registerTool(`laps_${domain}`, {
    title: `LAPS ${domain.replaceAll("_", " ")}`,
    description: "执行业务范围内的 LAPS 操作。系统会在写入前要求明确确认，并以业务语言返回结果。",
    inputSchema: inputShape(domain),
    outputSchema: outputShape,
    annotations: { readOnlyHint: false, destructiveHint: false, idempotentHint: false, openWorldHint: false },
  }, async (input) => {
    const result = await executeDomain(domain, input);
    return { content: [{ type: "text", text: result.guidance ? `${result.message} ${result.guidance}` : result.message }], structuredContent: result, isError: !result.success };
  });
}

export function createServer(version = "0.1.8") {
  const server = new McpServer({ name: "laps-mcp", version });
  server.registerTool("laps_connection", {
    title: "LAPS 连接状态",
    description: "检查 APS 系统地址和当前用户账号是否已准备就绪。",
    outputSchema: outputShape,
    annotations: { readOnlyHint: true, destructiveHint: false, idempotentHint: true, openWorldHint: false },
  }, async () => {
    const result = await connectionStatus();
    return { content: [{ type: "text", text: result.guidance ? `${result.message} ${result.guidance}` : result.message }], structuredContent: result, isError: !result.success };
  });
  for (const domain of domainNames) registerDomainTool(server, domain);
  return server;
}

export async function serve(version) {
  const server = createServer(version);
  await server.connect(new StdioServerTransport());
}
