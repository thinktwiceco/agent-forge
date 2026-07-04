const SENSITIVE_KEYS = new Set([
  "password", "passwd", "secret", "token", "apikey", "api_key", "apiKey",
  "key", "authorization", "auth", "credentials",
]);

function truncate(str, maxLen = 50) {
  if (typeof str !== "string") return String(str);
  return str.length <= maxLen ? str : str.slice(0, maxLen) + "…";
}

export function formatArguments(args) {
  if (!args || typeof args !== "object") return "";
  const parts = [];
  for (const [key, value] of Object.entries(args)) {
    if (SENSITIVE_KEYS.has(String(key).toLowerCase())) continue;
    const str = typeof value === "object" ? JSON.stringify(value) : String(value ?? "");
    parts.push(`${key}=${truncate(str, 40)}`);
  }
  return truncate(parts.join(", "), 100);
}

export function formatToolCallSummary(call) {
  const name = call?.function?.name || call?.name || "Unknown tool";
  const args = call?.function?.arguments ?? call?.arguments ?? {};
  const argSummary = formatArguments(args);
  return argSummary ? `${name}: ${argSummary}` : name;
}

export function formatToolResultSummary(result) {
  const name = result?.toolName || "Tool";
  if (!result?.success) return null;
  const resultStr = (result?.result || "").trim();
  const summary = resultStr ? truncate(resultStr, 80) : "";
  return summary ? `✓ ${name}: ${summary}` : `✓ ${name}`;
}
