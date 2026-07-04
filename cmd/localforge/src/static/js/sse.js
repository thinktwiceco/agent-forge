// ─── SSE stream parser ─────────────────────────────────────────────────────
export async function parseSSEStream(body, onEvent) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split("\n\n");
    buffer = parts.pop() || "";

    parts.forEach((part) => {
      const lines = part.split("\n");
      let eventType = "message";
      let data = "";
      lines.forEach((line) => {
        if (line.startsWith("event:")) eventType = line.slice(6).trim();
        else if (line.startsWith("data:")) data += line.slice(5).trim();
      });
      if (!data) return;
      try {
        onEvent(eventType, JSON.parse(data));
      } catch {
        onEvent("error", { content: "Failed to parse stream data" });
      }
    });
  }
}

// ─── Persistent SSE reconnect loop ─────────────────────────────────────────
export function subscribeSSE(url, { signal, onEvent, notOkDelayMs = 0, errorDelayMs = 2000, breakOnNotOk = false }) {
  (async () => {
    while (!signal.aborted) {
      try {
        const res = await fetch(url, { signal });
        if (!res.ok || !res.body) {
          if (breakOnNotOk) break;
          if (notOkDelayMs > 0) {
            await new Promise((r) => setTimeout(r, notOkDelayMs));
          }
          continue;
        }
        await parseSSEStream(res.body, onEvent);
      } catch (err) {
        if (err.name === "AbortError") break;
        await new Promise((r) => setTimeout(r, errorDelayMs));
      }
    }
  })();
}
