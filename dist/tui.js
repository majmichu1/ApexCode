import { createRequire } from "node:module";
var __require = /* @__PURE__ */ createRequire(import.meta.url);

// src/tui.tsx
import { Text, Box } from "@opentui/core";
import { createSignal, For, Show } from "solid-js";

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
async function getSuggestions(severity, url = DEFAULT_URL) {
  try {
    const params = severity ? `?severity=${encodeURIComponent(severity)}` : "";
    const resp = await fetch(`${url}/api/suggestions${params}`, { signal: AbortSignal.timeout(3000) });
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

// src/sentinel.ts
async function getIssues(severity) {
  const result = await getSuggestions(severity);
  if (!result)
    return [];
  return result.suggestions.map((s) => ({
    file: s.file,
    line: s.line,
    severity: s.severity.toLowerCase(),
    message: s.message
  }));
}

// src/tui.tsx
var PLUGIN_ID = "apexcode";
var tui = async (api) => {
  const GO_BACKEND_URL2 = "http://localhost:7777";
  api.slots.register({
    order: 100,
    slots: {
      sidebar_footer() {
        return /* @__PURE__ */ React.createElement(SidebarFooter, {
          api
        });
      }
    }
  });
  api.slots.register({
    order: 90,
    slots: {
      sidebar_content(ctx, props) {
        return /* @__PURE__ */ React.createElement(SentinelPanel, {
          api,
          sessionId: props.session_id
        });
      }
    }
  });
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
        const ok = await isHealthy(GO_BACKEND_URL2);
        api.ui.toast({
          variant: ok ? "success" : "error",
          title: "ApexCode Go Backend",
          message: ok ? "Healthy and connected on port 7777." : "Not responding. Run `apex --serve` to start it."
        });
      }
    }
  ]);
};
function SidebarFooter(props) {
  const { api } = props;
  const [connected, setConnected] = createSignal(false);
  let interval;
  async function check() {
    const ok = await isHealthy(GO_BACKEND_URL);
    setConnected(ok);
  }
  check();
  interval = setInterval(check, 30000);
  return /* @__PURE__ */ React.createElement(Box, {
    flexDirection: "row",
    gap: 1,
    justifyContent: "space-between"
  }, /* @__PURE__ */ React.createElement(Text, {
    fg: api.theme.current.textMuted
  }, /* @__PURE__ */ React.createElement(Text, {
    style: { fg: api.theme.current.success }
  }, "⚡"), " ApexCode v1.0.0"), /* @__PURE__ */ React.createElement(Show, {
    when: connected(),
    fallback: /* @__PURE__ */ React.createElement(Text, {
      fg: api.theme.current.textMuted
    }, "○")
  }, /* @__PURE__ */ React.createElement(Text, {
    fg: api.theme.current.success
  }, "●")));
}
function SentinelPanel(props) {
  const { api, sessionId } = props;
  const [issues, setIssues] = createSignal([]);
  const [loading, setLoading] = createSignal(false);
  async function fetchIssues() {
    setLoading(true);
    const result = await getIssues();
    setIssues(result);
    setLoading(false);
  }
  fetchIssues();
  return /* @__PURE__ */ React.createElement(Box, {
    flexDirection: "column",
    gap: 1
  }, /* @__PURE__ */ React.createElement(Box, null, /* @__PURE__ */ React.createElement(Text, {
    bold: true,
    fg: api.theme.current.text
  }, "KAIROS Analysis")), /* @__PURE__ */ React.createElement(Show, {
    when: loading()
  }, /* @__PURE__ */ React.createElement(Text, {
    fg: api.theme.current.textMuted
  }, "Scanning...")), /* @__PURE__ */ React.createElement(Show, {
    when: !loading() && issues().length > 0
  }, /* @__PURE__ */ React.createElement(Box, {
    flexDirection: "column",
    gap: 0
  }, /* @__PURE__ */ React.createElement(For, {
    each: issues().slice(0, 10)
  }, (issue) => {
    const color = {
      critical: api.theme.current.error,
      high: api.theme.current.warning,
      medium: api.theme.current.info,
      low: api.theme.current.textMuted
    }[issue.severity] ?? api.theme.current.textMuted;
    return /* @__PURE__ */ React.createElement(Text, {
      fg: color,
      wrap: "truncate"
    }, issue.file.split("/").pop(), ":", issue.line, " — ", issue.message.slice(0, 60));
  }))), /* @__PURE__ */ React.createElement(Show, {
    when: !loading() && issues().length === 0
  }, /* @__PURE__ */ React.createElement(Text, {
    fg: api.theme.current.success
  }, "✓ No issues found")), /* @__PURE__ */ React.createElement(Show, {
    when: issues().length > 10
  }, /* @__PURE__ */ React.createElement(Text, {
    fg: api.theme.current.textMuted
  }, "+", issues().length - 10, " more")));
}
var plugin = {
  id: PLUGIN_ID,
  tui
};
var tui_default = plugin;
export {
  tui_default as default
};

//# debugId=3EEF904C0389E67364756E2164756E21
