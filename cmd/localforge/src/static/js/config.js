export class ConfigPanel {
  constructor(appState) {
    this.state = appState;
    this.container = document.getElementById("config-body");
    this.configWrapper = this.container?.closest(".config-section-wrapper");
    this.reloadButton = document.getElementById("reload-agent");
    this.toggleButton = document.getElementById("config-toggle");
    
    if (this.reloadButton) {
      this.reloadButton.addEventListener("click", () => this.reloadAgent());
    }
    
    if (this.toggleButton) {
      this.toggleButton.addEventListener("click", () => this.toggleConfig());
    }
    
    // Start collapsed by default
    this.isCollapsed = true;
    this.updateToggleState();
  }
  
  toggleConfig() {
    this.isCollapsed = !this.isCollapsed;
    this.updateToggleState();
  }
  
  updateToggleState() {
    if (!this.configWrapper) return;
    
    if (this.isCollapsed) {
      this.container.classList.add("config-collapsed");
      this.configWrapper.classList.remove("config-expanded");
    } else {
      this.container.classList.remove("config-collapsed");
      this.configWrapper.classList.add("config-expanded");
    }
  }

  async load() {
    try {
      const res = await fetch("/api/config");
      if (!res.ok) {
        throw new Error("failed");
      }
      const config = await res.json();
      this.state.agentName = config.name || null;
      this.render(config);
    } catch (err) {
      this.container.textContent = "Failed to load config";
    }
  }

  render(config) {
    this.container.innerHTML = "";
    this.container.appendChild(this.renderAgentSection(config));
    config.tools.forEach((tool) => {
      this.container.appendChild(this.renderToolSection(tool));
    });
  }

  renderAgentSection(config) {
    const section = document.createElement("div");
    section.className = "config-section";
    section.innerHTML = `
      <h3>Agent</h3>
      <div class="config-field"><label>Name</label><input disabled value="${config.name || ""}"></div>
      <div class="config-field"><label>Model</label><input disabled value="${config.model || ""}"></div>
      <div class="config-field"><label>Working Dir</label><input disabled value="${config.workingDir || ""}"></div>
      <div class="config-field"><label>Persistence</label><input disabled value="${config.persistence || ""}"></div>
    `;
    return section;
  }

  renderToolSection(tool) {
    const section = document.createElement("div");
    section.className = "config-section";
    const header = document.createElement("h3");
    header.textContent = `Tool: ${tool.name}`;
    section.appendChild(header);

    const fields = [];
    if (tool.root !== undefined) {
      fields.push(this.createInput("Root", "root", tool.root));
    }
    if (tool.postgresURL !== undefined) {
      fields.push(this.createInput("Postgres URL", "postgresURL", tool.postgresURL));
    }
    if (tool.mode !== undefined) {
      fields.push(this.createInput("Mode", "mode", tool.mode));
    }
    if (tool.allowedTables !== undefined) {
      fields.push(
        this.createInput(
          "Allowed Tables (comma separated)",
          "allowedTables",
          (tool.allowedTables || []).join(", ")
        )
      );
    }
    if (tool.allowedSchemas !== undefined) {
      fields.push(
        this.createInput(
          "Allowed Schemas (comma separated)",
          "allowedSchemas",
          (tool.allowedSchemas || []).join(", ")
        )
      );
    }

    fields.forEach((field) => section.appendChild(field));

    const saveButton = document.createElement("button");
    saveButton.className = "btn";
    saveButton.textContent = "Save";
    saveButton.addEventListener("click", () => {
      this.saveTool(tool.name, fields);
    });
    section.appendChild(saveButton);

    return section;
  }

  createInput(labelText, name, value) {
    const wrapper = document.createElement("div");
    wrapper.className = "config-field";
    const label = document.createElement("label");
    label.textContent = labelText;
    const input = document.createElement("input");
    input.name = name;
    input.value = value || "";
    wrapper.appendChild(label);
    wrapper.appendChild(input);
    return wrapper;
  }

  async saveTool(toolName, fields) {
    const payload = {};
    fields.forEach((field) => {
      const input = field.querySelector("input");
      if (!input) {
        return;
      }
      if (input.name === "allowedTables" || input.name === "allowedSchemas") {
        const values = input.value
          .split(",")
          .map((item) => item.trim())
          .filter(Boolean);
        payload[input.name] = values;
        return;
      }
      payload[input.name] = input.value;
    });

    try {
      const res = await fetch(`/api/config/tools/${toolName}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        throw new Error("failed");
      }
    } catch (err) {
      alert("Failed to save tool config");
    }
  }

  async reloadAgent() {
    try {
      const res = await fetch("/api/agent/reload", { method: "POST" });
      if (!res.ok) {
        throw new Error("failed");
      }
      await this.load();
    } catch (err) {
      alert("Failed to reload agent");
    }
  }
}
