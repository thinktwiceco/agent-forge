function escapeHtml(s) {
  if (typeof s !== "string") return "";
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

let markedCodeRendererInstalled = false;

function ensureMarkedCodeRenderer() {
  if (markedCodeRendererInstalled || typeof marked === "undefined") return;
  marked.use({
    renderer: {
      code(src, infostring, escaped) {
        const langRaw =
          ((infostring || "").match(/^\S*/)?.[0] || "").trim() || "text";
        const langDisplay = langRaw.toLowerCase();
        const langSlug =
          langRaw.replace(/[^a-zA-Z0-9_.-]/g, "").replace(/^\.+/, "") || "text";
        const body = escaped ? src : escapeHtml(src);
        return `<div class="code-block"><div class="code-header"><span class="code-lang">${escapeHtml(langDisplay)}</span><button type="button" class="code-copy-btn">Copy</button></div><pre><code class="language-${escapeHtml(langSlug)}">${body}</code></pre></div>`;
      },
    },
  });
  markedCodeRendererInstalled = true;
}

export function renderMarkdown(text) {
  if (typeof marked === "undefined" || typeof DOMPurify === "undefined") return text;
  ensureMarkedCodeRenderer();
  const raw = marked.parse(text);
  return DOMPurify.sanitize(raw, {
    ADD_TAGS: ["button", "div", "span"],
    ADD_ATTR: ["class", "type"],
  });
}

export class MarkdownRenderer {
  constructor(onRender, delayMs = 80) {
    this.onRender = onRender;
    this.delayMs = delayMs;
    this.timer = null;
    this.pendingText = "";
  }

  schedule(text) {
    this.pendingText = text;
    if (this.timer) return;
    this.timer = setTimeout(() => this.flush(), this.delayMs);
  }

  flush(text) {
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    const value = text ?? this.pendingText ?? "";
    this.pendingText = "";
    this.onRender(value);
  }

  cancel() {
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    this.pendingText = "";
  }
}
