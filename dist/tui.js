import { createRequire } from "node:module";
var __require = /* @__PURE__ */ createRequire(import.meta.url);

// src/go-backend.ts
var DEFAULT_URL = "http://localhost:7777";
var _cachedHealth = { ok: false, at: 0 };
var HEALTH_TTL = 5000;
async function isHealthy(url = DEFAULT_URL) {
  const now = Date.now();
  if (now - _cachedHealth.at < HEALTH_TTL)
    return _cachedHealth.ok;
  try {
    const resp = await fetch(`${url}/health`, { signal: AbortSignal.timeout(2000) });
    _cachedHealth = { ok: resp.ok, at: now };
    return resp.ok;
  } catch {
    _cachedHealth = { ok: false, at: now };
    return false;
  }
}
async function enhance(workDir, prompt, url = DEFAULT_URL) {
  try {
    const resp = await fetch(`${url}/api/enhance`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ work_dir: workDir, prompt }),
      signal: AbortSignal.timeout(3000)
    });
    if (!resp.ok)
      return null;
    return await resp.json();
  } catch {
    return null;
  }
}
async function runSwarm(task, agents, mode, url = DEFAULT_URL) {
  try {
    const resp = await fetch(`${url}/api/swarm`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ task, agents: agents ?? [], mode: mode ?? "parallel" }),
      signal: AbortSignal.timeout(1e4)
    });
    if (!resp.ok)
      return null;
    return await resp.json();
  } catch {
    return null;
  }
}
async function discoverLmStudioModels(baseUrl = "http://127.0.0.1:1234") {
  try {
    const resp = await fetch(`${baseUrl}/v1/models`, { signal: AbortSignal.timeout(3000) });
    if (!resp.ok)
      return [];
    const data = await resp.json();
    return (data.data ?? []).map((m) => ({ id: m.id, name: m.id.split("/").pop() ?? m.id }));
  } catch {
    return [];
  }
}

// src/tui.tsx
var PLUGIN_ID = "apexcode";
var tui = async (api) => {
  const GO_BACKEND_URL = "http://localhost:7777";
  api.command.register(() => [
    {
      title: "ApexCode: Run Swarm",
      value: "apexcode.swarm",
      category: "ApexCode",
      slash: { name: "swarm", aliases: ["agents"] },
      onSelect() {
        api.ui.toast({
          variant: "info",
          title: "Swarm",
          message: "Use the apexcode_swarm tool to execute a multi-agent swarm."
        });
      }
    },
    {
      title: "Sentinel: Code Issues",
      value: "apexcode.sentinel",
      category: "ApexCode",
      slash: { name: "sentinel", aliases: ["suggest", "issues"] },
      onSelect() {
        api.ui.toast({
          variant: "info",
          title: "Sentinel",
          message: "Check the sidebar for proactive code analysis issues."
        });
      }
    },
    {
      title: "ApexCode: Health Check",
      value: "apexcode.health",
      category: "ApexCode",
      slash: { name: "apex", aliases: ["apexcode"] },
      async onSelect() {
        const ok = await isHealthy(GO_BACKEND_URL);
        api.ui.toast({
          variant: ok ? "success" : "error",
          title: "ApexCode Go Backend",
          message: ok ? "Healthy and connected on port 7777." : "Not responding. Run `apex --serve` to start it."
        });
      }
    }
  ]);
};
var plugin = {
  id: PLUGIN_ID,
  tui
};
var tui_default = plugin;
export {
  tui_default as default
};

//# debugId=24828E51D68004FE64756E2164756E21
