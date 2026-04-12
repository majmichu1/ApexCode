import { createRequire } from "node:module";
var __require = /* @__PURE__ */ createRequire(import.meta.url);

// src/server.ts
import { tool } from "@opencode-ai/plugin/tool";
import { spawn } from "child_process";

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

// src/enhance.ts
async function injectContext(workDir, userPrompt, backendUrl = "http://localhost:7777") {
  const result = await enhance(workDir, userPrompt, backendUrl);
  if (!result?.connected)
    return null;
  const sections = [];
  if (result.repomap) {
    sections.push("<repository_map>", "The following is a repository map showing the most relevant files for your current task.", "This map is built using PageRank analysis of code dependencies.", result.repomap, "</repository_map>");
  }
  if (result.memory) {
    sections.push("<memory_context>", "The following is relevant context from your memory (MemPalace system).", "This knowledge was retrieved from previous sessions and interactions.", result.memory, "</memory_context>");
  }
  if (sections.length === 0)
    return null;
  return [
    "<apexcode_context>",
    "You are augmented with ApexCode's codebase intelligence.",
    `Token savings from context injection: ${result.token_savings ?? "N/A"}`,
    ...sections,
    "</apexcode_context>"
  ].join(`
`);
}

// src/lmstudio.ts
async function discoverModels(baseUrl = "http://127.0.0.1:1234") {
  const models = await discoverLmStudioModels(baseUrl);
  return models.map((m) => ({
    id: m.id,
    name: m.name,
    url: `${baseUrl}/v1`
  }));
}

// src/swarm.ts
var AVAILABLE_AGENTS = [
  { id: "planner", name: "Planner", desc: "Break down tasks into actionable steps" },
  { id: "architect", name: "Architect", desc: "Design architecture and define interfaces" },
  { id: "coder", name: "Coder", desc: "Write clean, efficient code" },
  { id: "reviewer", name: "Reviewer", desc: "Code review for correctness and security" },
  { id: "tester", name: "Tester", desc: "Write comprehensive tests" },
  { id: "documenter", name: "Documenter", desc: "Create documentation and guides" }
];
async function executeSwarm(task, agents = ["planner", "coder", "reviewer"], mode = "parallel") {
  const result = await runSwarm(task, agents, mode);
  if (!result)
    return null;
  return {
    status: result.status,
    message: result.message,
    agents: result.agents,
    mode: result.mode
  };
}
function formatSwarmResult(result) {
  return [
    `Swarm: ${result.status}`,
    `Agents: ${result.agents.join(", ")}`,
    `Mode: ${result.mode}`,
    result.message
  ].join(`
`);
}

// src/server.ts
var _workDir = "";
var _backendUrl = "http://localhost:7777";
var _goProcess = null;
function findApexBinary() {
  const candidates = [
    "apex",
    process.env.HOME + "/.local/bin/apex",
    process.env.HOME + "/go/bin/apex"
  ];
  for (const c of candidates) {
    try {
      const { execSync } = __require("child_process");
      if (c === "apex") {
        execSync("which apex", { stdio: "pipe" });
        return "apex";
      }
      const { statSync } = __require("fs");
      statSync(c);
      return c;
    } catch {}
  }
  return null;
}
async function startGoBackend(workDir) {
  if (await isHealthy())
    return true;
  const binary = findApexBinary();
  if (!binary) {
    console.error("[apexcode] Go backend binary not found. Install 'apex' or build from source.");
    return false;
  }
  console.log(`[apexcode] Starting Go backend from ${binary}...`);
  return new Promise((resolve) => {
    const proc = spawn(binary, ["--serve"], {
      cwd: workDir,
      stdio: ["pipe", "pipe", "pipe"],
      detached: true
    });
    _goProcess = proc;
    proc.stdout?.on("data", (d) => {
      const msg = d.toString();
      if (msg.includes("starting") || msg.includes("listening")) {
        console.log(`[apexcode] Go backend stdout: ${msg.trim()}`);
      }
    });
    proc.stderr?.on("data", (d) => {
      const msg = d.toString();
      console.log(`[apexcode] Go backend: ${msg.trim()}`);
    });
    proc.on("error", (e) => {
      console.error(`[apexcode] Go backend failed: ${e.message}`);
      resolve(false);
    });
    const poll = async (attempt = 0) => {
      if (attempt > 15) {
        console.error("[apexcode] Go backend did not become healthy after 15s");
        resolve(false);
        return;
      }
      await new Promise((r) => setTimeout(r, 1000));
      const ok = await isHealthy();
      if (ok) {
        console.log(`[apexcode] Go backend ready on port 7777 (${attempt + 1}s)`);
        resolve(true);
      } else {
        poll(attempt + 1);
      }
    };
    poll(0);
  });
}
var server = async (input) => {
  _workDir = input.directory;
  startGoBackend(input.directory).catch(() => {});
  const hooks = {};
  hooks["experimental.chat.system.transform"] = async (input2, output) => {
    const ctx = await injectContext(_workDir, "", _backendUrl);
    if (ctx) {
      output.system.push(ctx);
    }
  };
  hooks.provider = {
    id: "lmstudio",
    async models() {
      const modelsList = await discoverModels();
      const result = {};
      for (const m of modelsList) {
        result[m.id] = {
          id: m.id,
          name: m.name,
          provider: {
            id: "lmstudio",
            name: "LM Studio",
            options: { baseURL: m.url },
            npm: "@ai-sdk/openai-compatible"
          }
        };
      }
      return result;
    }
  };
  hooks.tool = {
    apexcode_enhance: tool({
      description: "Refresh ApexCode context (MemPalace memory + repository map). Use when you need updated codebase intelligence.",
      args: {},
      async execute() {
        const ctx = await injectContext(_workDir, "current context", _backendUrl);
        if (!ctx)
          return "ApexCode Go backend not available. Ensure `apex --serve` is running.";
        return ctx;
      }
    }),
    apexcode_swarm: tool({
      description: "Execute a multi-agent swarm. Spawns specialized agents (planner, architect, coder, reviewer, tester, documenter) to collaborate on complex tasks.",
      args: {
        task: tool.schema.string().describe("The task description for the swarm"),
        agents: tool.schema.array(tool.schema.string().describe("Agent ID")).describe(`List of agent IDs. Available: ${AVAILABLE_AGENTS.map((a) => `${a.id} (${a.name})`).join(", ")}. Default: ["planner", "coder", "reviewer"]`).optional(),
        mode: tool.schema.enum(["parallel", "sequential"]).describe("Execution mode. Default: parallel").optional()
      },
      async execute(args) {
        const agents = args.agents ?? ["planner", "coder", "reviewer"];
        const mode = args.mode ?? "parallel";
        const result = await executeSwarm(args.task, agents, mode);
        if (!result) {
          return "Swarm execution failed. Ensure the ApexCode Go backend is running on port 7777.";
        }
        return formatSwarmResult(result);
      }
    }),
    apexcode_health: tool({
      description: "Check if the ApexCode Go backend is healthy and connected.",
      args: {},
      async execute() {
        const ok = await isHealthy(_backendUrl);
        return ok ? "ApexCode Go backend is healthy and connected on port 7777." : "ApexCode Go backend is not responding. Run `apex --serve` to start it.";
      }
    })
  };
  return hooks;
};
var plugin = {
  id: "apexcode",
  server
};
var server_default = plugin;
export {
  server_default as default
};

//# debugId=1697F8668C18A0DD64756E2164756E21
