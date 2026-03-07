import { setupAuthUI } from "./auth.js";

// All known plugins the registry supports.
const ALL_PLUGINS = ["todo", "procedures", "vault", "knowledge", "scheduler", "logger"];

class SettingsManager {
  constructor() {
    this.config = null;
    this.providers = [];
    this._bindNav();
    this._bindReload();
  }

  // ─── Navigation ────────────────────────────────────────────────────────────

  _bindNav() {
    document.querySelectorAll(".settings-nav-item[data-section]").forEach((btn) => {
      btn.addEventListener("click", () => {
        document.querySelectorAll(".settings-nav-item").forEach((b) => b.classList.remove("active"));
        document.querySelectorAll(".settings-section").forEach((s) => s.classList.remove("active"));
        btn.classList.add("active");
        document.getElementById(`section-${btn.dataset.section}`)?.classList.add("active");
      });
    });
  }

  // ─── Load ──────────────────────────────────────────────────────────────────

  async load() {
    const [cfgRes, provRes] = await Promise.all([
      fetch("/api/config"),
      fetch("/api/config/providers"),
    ]);

    if (!cfgRes.ok || !provRes.ok) {
      console.error("Failed to load settings");
      return;
    }

    this.config = await cfgRes.json();
    this.providers = await provRes.json();

    this._renderAgent(this.config);
    this._renderSubagents(this.config);
    this._renderPlugins(this.config);
    this._renderTools(this.config);
    this._renderProviders(this.providers);

    const nameEl = document.getElementById("reload-agent-name");
    if (nameEl && this.config.name) nameEl.textContent = this.config.name;
  }

  // ─── Agent ─────────────────────────────────────────────────────────────────

  _renderAgent(cfg) {
    document.getElementById("agent-name").value = cfg.name || "";
    document.getElementById("agent-model").value = cfg.model || "";
    document.getElementById("agent-system-prompt").value = cfg.systemPrompt || "";
    document.getElementById("agent-working-dir").value = cfg.workingDir || "";
    const persistEl = document.getElementById("agent-persistence");
    persistEl.value = cfg.persistence || "";

    document.getElementById("save-agent").addEventListener("click", () => this._saveAgent());
  }

  async _saveAgent() {
    const payload = {
      name: document.getElementById("agent-name").value,
      model: document.getElementById("agent-model").value,
      systemPrompt: document.getElementById("agent-system-prompt").value,
      workingDir: document.getElementById("agent-working-dir").value,
      persistence: document.getElementById("agent-persistence").value,
    };

    await this._save(
      "/api/config",
      "PUT",
      payload,
      "save-agent-status",
      async (updated) => {
        this.config = updated;
        const nameEl = document.getElementById("reload-agent-name");
        if (nameEl && updated.name) nameEl.textContent = updated.name;
      }
    );
  }

  // ─── Sub-agents ────────────────────────────────────────────────────────────

  _renderSubagents(cfg) {
    const list = document.getElementById("subagents-list");
    list.innerHTML = "";

    const subagents = cfg.subagents || {};
    for (const [role, model] of Object.entries(subagents)) {
      list.appendChild(this._subagentRow(role, model));
    }

    document.getElementById("add-subagent").onclick = () => {
      list.appendChild(this._subagentRow("", ""));
    };

    document.getElementById("save-subagents").onclick = () => this._saveSubagents();
  }

  _subagentRow(role, model) {
    const row = document.createElement("div");
    row.className = "subagent-row";
    row.innerHTML = `
      <input type="text" placeholder="role (e.g. reasoning)" value="${_esc(role)}" data-field="role" />
      <input type="text" placeholder="model (e.g. deepseek::deepseek-chat)" value="${_esc(model)}" data-field="model" />
      <button class="btn-icon" title="Remove" style="color:var(--error)">✕</button>
    `;
    row.querySelector("button").onclick = () => row.remove();
    return row;
  }

  async _saveSubagents() {
    const rows = document.querySelectorAll("#subagents-list .subagent-row");
    const subagents = {};
    for (const row of rows) {
      const role = row.querySelector('[data-field="role"]').value.trim();
      const model = row.querySelector('[data-field="model"]').value.trim();
      if (role && model) subagents[role] = model;
    }
    await this._save("/api/config/subagents", "PUT", { subagents }, "save-subagents-status");
  }

  // ─── Plugins ───────────────────────────────────────────────────────────────

  _renderPlugins(cfg) {
    const grid = document.getElementById("plugins-grid");
    grid.innerHTML = "";
    const active = new Set(cfg.plugins || []);

    for (const name of ALL_PLUGINS) {
      const item = document.createElement("label");
      item.className = "plugin-item" + (active.has(name) ? " checked" : "");
      item.innerHTML = `
        <input type="checkbox" value="${name}" ${active.has(name) ? "checked" : ""} />
        <span>${name}</span>
      `;
      const cb = item.querySelector("input");
      cb.addEventListener("change", () => {
        item.classList.toggle("checked", cb.checked);
      });
      grid.appendChild(item);
    }

    document.getElementById("save-plugins").onclick = () => this._savePlugins();
  }

  async _savePlugins() {
    const checked = [...document.querySelectorAll("#plugins-grid input[type=checkbox]:checked")]
      .map((cb) => cb.value);
    await this._save("/api/config/plugins", "PUT", { plugins: checked }, "save-plugins-status");
  }

  // ─── Tools ─────────────────────────────────────────────────────────────────

  _renderTools(cfg) {
    const list = document.getElementById("tools-list");
    list.innerHTML = "";

    if (!cfg.tools || cfg.tools.length === 0) {
      list.innerHTML = `<p style="color:var(--text-muted);font-size:13px">No tools configured in config.yaml.</p>`;
      return;
    }

    for (const tool of cfg.tools) {
      list.appendChild(this._toolCard(tool));
    }
  }

  _toolCard(tool) {
    const card = document.createElement("div");
    card.className = "settings-card";

    const fields = [];

    const addField = (label, name, value) => {
      fields.push({ name, el: null });
      const div = document.createElement("div");
      div.className = "field";
      div.innerHTML = `<label>${label}</label><input type="text" name="${name}" value="${_esc(value || "")}" />`;
      card.appendChild(div);
      fields[fields.length - 1].el = div.querySelector("input");
    };

    card.innerHTML = `<p class="settings-card-title">Tool: ${_esc(tool.name)}</p>`;

    if (tool.postgresURL !== undefined) addField("Postgres URL", "postgresURL", tool.postgresURL);
    if (tool.mode !== undefined) addField("Mode", "mode", tool.mode);
    if (tool.allowedTables !== undefined) addField("Allowed Tables (comma-separated)", "allowedTables", (tool.allowedTables || []).join(", "));
    if (tool.allowedSchemas !== undefined) addField("Allowed Schemas (comma-separated)", "allowedSchemas", (tool.allowedSchemas || []).join(", "));

    if (fields.length === 0) {
      const note = document.createElement("p");
      note.style.cssText = "color:var(--text-muted);font-size:13px;margin:0";
      note.textContent = "No configurable fields for this tool.";
      card.appendChild(note);
      return card;
    }

    const footer = document.createElement("div");
    footer.className = "section-footer";
    const saveBtn = document.createElement("button");
    saveBtn.className = "btn primary";
    saveBtn.textContent = "Save";
    const statusEl = document.createElement("span");
    statusEl.className = "save-status";
    footer.appendChild(saveBtn);
    footer.appendChild(statusEl);
    card.appendChild(footer);

    saveBtn.onclick = async () => {
      const payload = {};
      for (const f of fields) {
        if (!f.el) continue;
        if (f.name === "allowedTables" || f.name === "allowedSchemas") {
          payload[f.name] = f.el.value.split(",").map((s) => s.trim()).filter(Boolean);
        } else {
          payload[f.name] = f.el.value;
        }
      }
      await this._saveTool(tool.name, payload, statusEl);
    };

    return card;
  }

  async _saveTool(toolName, payload, statusEl) {
    try {
      const res = await fetch(`/api/config/tools/${toolName}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: "failed" }));
        _setStatus(statusEl, "error: " + (err.error || "failed"), false);
        return;
      }
      _setStatus(statusEl, "Saved", true);
    } catch {
      _setStatus(statusEl, "Network error", false);
    }
  }

  // ─── Providers / API Keys ──────────────────────────────────────────────────

  _renderProviders(providers) {
    const list = document.getElementById("providers-list");
    list.innerHTML = "";

    const groups = { llm: [], messaging: [] };
    for (const p of providers) {
      (groups[p.group] || groups.llm).push(p);
    }

    const groupLabels = { llm: "LLM Providers", messaging: "Messaging Providers" };

    for (const [group, items] of Object.entries(groups)) {
      if (items.length === 0) continue;

      const card = document.createElement("div");
      card.className = "settings-card";
      card.innerHTML = `<p class="settings-card-title">${groupLabels[group] || group}</p>`;

      for (const p of items) {
        const fieldDiv = document.createElement("div");
        fieldDiv.className = "field";
        fieldDiv.style.marginBottom = "12px";
        fieldDiv.innerHTML = `
          <label style="display:flex;justify-content:space-between;align-items:center">
            <span>${_esc(p.label)}</span>
            <span class="token-badge ${p.isSet ? "set" : "unset"}">${p.isSet ? "set" : "not set"}</span>
          </label>
          <div class="token-field">
            <input type="password"
              data-env="${_esc(p.envKey)}"
              placeholder="${p.isSet ? p.maskedValue || "••••••••" : "Paste token…"}"
              autocomplete="off"
            />
            <button class="btn-icon" data-show="0" title="Show/hide" style="font-size:14px">👁</button>
            <button class="btn primary" style="padding:6px 12px" data-save-key="${_esc(p.envKey)}">Save</button>
          </div>
        `;

        const input = fieldDiv.querySelector("input");
        const toggleBtn = fieldDiv.querySelector("[data-show]");
        toggleBtn.onclick = () => {
          const shown = toggleBtn.dataset.show === "1";
          input.type = shown ? "password" : "text";
          toggleBtn.dataset.show = shown ? "0" : "1";
        };

        const saveBtn = fieldDiv.querySelector("[data-save-key]");
        saveBtn.onclick = async () => {
          const val = input.value.trim();
          if (!val) return;
          await this._saveProvider(p.envKey, val, saveBtn, fieldDiv);
        };

        card.appendChild(fieldDiv);
      }

      list.appendChild(card);
    }
  }

  async _saveProvider(envKey, value, saveBtn, fieldDiv) {
    const orig = saveBtn.textContent;
    saveBtn.disabled = true;
    saveBtn.textContent = "Saving…";
    try {
      const res = await fetch("/api/config/providers", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ providers: { [envKey]: value } }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: "failed" }));
        saveBtn.textContent = "Error";
        setTimeout(() => { saveBtn.textContent = orig; saveBtn.disabled = false; }, 2000);
        return;
      }
      // Update badge
      const badge = fieldDiv.querySelector(".token-badge");
      if (badge) {
        badge.textContent = "set";
        badge.className = "token-badge set";
      }
      const input = fieldDiv.querySelector("input");
      if (input) input.value = "";
      saveBtn.textContent = "Saved";
      setTimeout(() => { saveBtn.textContent = orig; saveBtn.disabled = false; }, 1500);
    } catch {
      saveBtn.textContent = "Error";
      setTimeout(() => { saveBtn.textContent = orig; saveBtn.disabled = false; }, 2000);
    }
  }

  // ─── Generic save helper ───────────────────────────────────────────────────

  async _save(url, method, body, statusId, onSuccess) {
    const statusEl = document.getElementById(statusId);
    try {
      const res = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: "request failed" }));
        _setStatus(statusEl, "Error: " + (err.error || "failed"), false);
        return;
      }
      const data = res.status !== 204 ? await res.json().catch(() => null) : null;
      if (onSuccess) await onSuccess(data);
      _setStatus(statusEl, "Saved", true);
    } catch (e) {
      _setStatus(statusEl, "Network error", false);
    }
  }

  // ─── Reload ────────────────────────────────────────────────────────────────

  _bindReload() {
    document.getElementById("reload-btn").addEventListener("click", () => this._reload());
  }

  async _reload() {
    const btn = document.getElementById("reload-btn");
    const status = document.getElementById("reload-status");
    btn.disabled = true;
    btn.textContent = "Reloading…";
    status.textContent = "";
    status.className = "reload-status";
    try {
      const res = await fetch("/api/agent/reload", { method: "POST" });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: "failed" }));
        status.textContent = "Error: " + (err.error || "failed");
        status.className = "reload-status err";
      } else {
        status.textContent = "Agent reloaded";
        status.className = "reload-status ok";
        await this.load();
        setTimeout(() => { status.textContent = ""; }, 3000);
      }
    } catch {
      status.textContent = "Network error";
      status.className = "reload-status err";
    } finally {
      btn.disabled = false;
      btn.textContent = "Reload Agent";
    }
  }
}

// ─── Utilities ───────────────────────────────────────────────────────────────

function _esc(str) {
  return String(str ?? "").replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function _setStatus(el, msg, ok) {
  if (!el) return;
  el.textContent = msg;
  el.className = "save-status " + (ok ? "ok" : "err");
  if (ok) setTimeout(() => { el.textContent = ""; el.className = "save-status"; }, 2500);
}

// ─── Init ─────────────────────────────────────────────────────────────────────

const settings = new SettingsManager();
(async () => {
  await setupAuthUI();
  await settings.load();
})();
