import type { TuiPlugin, TuiPluginModule } from "@opencode-ai/plugin/tui"
import { isHealthy } from "./go-backend"

const PLUGIN_ID = "apexcode"

// ---------------------------------------------------------------
// TUI Plugin — registers slash commands
// ---------------------------------------------------------------
const tui: TuiPlugin = async (api) => {
  const GO_BACKEND_URL = "http://localhost:7777"

  // ---- Slash Commands ----
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
          message: "Use the apexcode_swarm tool to execute a multi-agent swarm.",
        })
      },
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
          message: "Check the sidebar for proactive code analysis issues.",
        })
      },
    },
    {
      title: "ApexCode: Health Check",
      value: "apexcode.health",
      category: "ApexCode",
      slash: { name: "apex", aliases: ["apexcode"] },
      async onSelect() {
        const ok = await isHealthy(GO_BACKEND_URL)
        api.ui.toast({
          variant: ok ? "success" : "error",
          title: "ApexCode Go Backend",
          message: ok ? "Healthy and connected on port 7777." : "Not responding. Run `apex --serve` to start it.",
        })
      },
    },
  ])
}

const plugin: TuiPluginModule = {
  id: PLUGIN_ID,
  tui,
}

export default plugin
