import type { Plugin, PluginModule, Hooks, PluginInput } from "@opencode-ai/plugin"
import type { Model, Provider } from "@opencode-ai/sdk"
import { tool } from "@opencode-ai/plugin/tool"
import { spawn, type ChildProcess } from "child_process"
import { injectContext } from "./enhance"
import { discoverModels } from "./lmstudio"
import { AVAILABLE_AGENTS, executeSwarm, formatSwarmResult } from "./swarm"
import { isHealthy } from "./go-backend"

let _workDir = ""
let _backendUrl = "http://localhost:7777"
let _goProcess: ChildProcess | null = null

/**
 * Find the apex binary in common locations
 */
function findApexBinary(): string | null {
  const candidates = [
    "apex",                                    // PATH
    process.env.HOME + "/.local/bin/apex",    // local install
    process.env.HOME + "/go/bin/apex",        // go bin
  ]
  for (const c of candidates) {
    try {
      const { execSync } = require("child_process")
      if (c === "apex") {
        execSync("which apex", { stdio: "pipe" })
        return "apex"
      }
      const { statSync } = require("fs")
      statSync(c)
      return c
    } catch { /* not found */ }
  }
  return null
}

/**
 * Auto-start the Go backend if not already running
 */
async function startGoBackend(workDir: string): Promise<boolean> {
  // Check if already healthy
  if (await isHealthy()) return true

  // Find apex binary
  const binary = findApexBinary()
  if (!binary) {
    console.error("[apexcode] Go backend binary not found. Install 'apex' or build from source.")
    return false
  }

  console.log(`[apexcode] Starting Go backend from ${binary}...`)

  return new Promise((resolve) => {
    const proc = spawn(binary, ["--serve"], {
      cwd: workDir,
      stdio: ["pipe", "pipe", "pipe"],
      detached: true,
    })
    _goProcess = proc

    proc.stdout?.on("data", (d) => {
      const msg = d.toString()
      if (msg.includes("starting") || msg.includes("listening")) {
        console.log(`[apexcode] Go backend stdout: ${msg.trim()}`)
      }
    })

    proc.stderr?.on("data", (d) => {
      const msg = d.toString()
      console.log(`[apexcode] Go backend: ${msg.trim()}`)
    })

    proc.on("error", (e) => {
      console.error(`[apexcode] Go backend failed: ${e.message}`)
      resolve(false)
    })

    // Poll for readiness
    const poll = async (attempt = 0) => {
      if (attempt > 15) {
        console.error("[apexcode] Go backend did not become healthy after 15s")
        resolve(false)
        return
      }
      await new Promise((r) => setTimeout(r, 1000))
      const ok = await isHealthy()
      if (ok) {
        console.log(`[apexcode] Go backend ready on port 7777 (${attempt + 1}s)`)
        resolve(true)
      } else {
        poll(attempt + 1)
      }
    }
    poll(0)
  })
}

const server: Plugin = async (input): Promise<Hooks> => {
  _workDir = input.directory

  // Auto-start Go backend (non-blocking — hooks work regardless)
  startGoBackend(input.directory).catch(() => {})

  const hooks: Hooks = {}

  // ---------------------------------------------------------------
  // System Prompt Transform — inject MemPalace + Repomap context
  // ---------------------------------------------------------------
  hooks["experimental.chat.system.transform"] = async (input, output) => {
    const ctx = await injectContext(_workDir, "", _backendUrl)
    if (ctx) {
      output.system.push(ctx)
    }
  }

  // ---------------------------------------------------------------
  // LM Studio Provider — discover local models
  // ---------------------------------------------------------------
  hooks.provider = {
    id: "lmstudio",
    async models() {
      const modelsList = await discoverModels()
      const result: Record<string, Model> = {}
      for (const m of modelsList) {
        result[m.id] = {
          id: m.id,
          name: m.name,
          provider: {
            id: "lmstudio",
            name: "LM Studio",
            options: { baseURL: m.url },
            npm: "@ai-sdk/openai-compatible",
          },
        } as Model
      }
      return result
    },
  }

  // ---------------------------------------------------------------
  // Tools
  // ---------------------------------------------------------------
  hooks.tool = {
    apexcode_enhance: tool({
      description: "Refresh ApexCode context (MemPalace memory + repository map). Use when you need updated codebase intelligence.",
      args: {},
      async execute() {
        const ctx = await injectContext(_workDir, "current context", _backendUrl)
        if (!ctx) return { output: "ApexCode Go backend not available. Ensure it is running on port 7777." }
        return { output: ctx }
      },
    }),

    apexcode_swarm: tool({
      description: "Execute a multi-agent swarm. Spawns specialized agents (planner, architect, coder, reviewer, tester, documenter) to collaborate on complex tasks.",
      args: {
        task: tool.schema.string().describe("The task description for the swarm"),
        agents: tool.schema
          .array(tool.schema.string().describe("Agent ID"))
          .describe(
            `List of agent IDs. Available: ${AVAILABLE_AGENTS.map((a) => `${a.id} (${a.name})`).join(", ")}. Default: ["planner", "coder", "reviewer"]`,
          )
          .optional(),
        mode: tool.schema
          .enum(["parallel", "sequential"])
          .describe("Execution mode. Default: parallel")
          .optional(),
      },
      async execute(args) {
        const agents = args.agents ?? ["planner", "coder", "reviewer"]
        const mode = (args.mode as "parallel" | "sequential") ?? "parallel"
        const result = await executeSwarm(args.task, agents, mode)
        if (!result) {
          return { output: "Swarm execution failed. Ensure the ApexCode Go backend is running on port 7777." }
        }
        return { output: formatSwarmResult(result) }
      },
    }),

    apexcode_health: tool({
      description: "Check if the ApexCode Go backend is healthy and connected.",
      args: {},
      async execute() {
        const ok = await isHealthy(_backendUrl)
        return { output: ok ? "ApexCode Go backend is healthy and connected." : "ApexCode Go backend is not responding." }
      },
    }),
  }

  return hooks
}

const plugin: PluginModule = {
  id: "apexcode",
  server,
}

export default plugin
