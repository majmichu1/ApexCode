import type { TuiPlugin, TuiPluginModule } from "@opencode-ai/plugin/tui"
import { Text, Box } from "@opentui/core"
import { createSignal, createEffect, For, Show, onCleanup } from "solid-js"
import { isHealthy, getSuggestions } from "./go-backend"
import { getIssues, formatIssues, type KairosIssue } from "./kairos"

const PLUGIN_ID = "apexcode"

// ---------------------------------------------------------------
// TUI Plugin
// ---------------------------------------------------------------
const tui: TuiPlugin = async (api) => {
  const GO_BACKEND_URL = "http://localhost:7777"

  // ---- Sidebar Footer: ApexCode status ----
  api.slots.register({
    order: 100,
    slots: {
      sidebar_footer() {
        return <SidebarFooter api={api} />
      },
    },
  })

  // ---- Sidebar Content: KAIROS issues panel ----
  api.slots.register({
    order: 90,
    slots: {
      sidebar_content(ctx, props: { session_id: string }) {
        return <KairosPanel api={api} sessionId={props.session_id} />
      },
    },
  })

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
      title: "ApexCode: KAIROS Issues",
      value: "apexcode.kairos",
      category: "ApexCode",
      slash: { name: "kairos", aliases: ["suggest", "issues"] },
      onSelect() {
        api.ui.toast({
          variant: "info",
          title: "KAIROS",
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

// ---------------------------------------------------------------
// Components
// ---------------------------------------------------------------

function SidebarFooter(props: { api: Parameters<TuiPlugin>[0] }) {
  const { api } = props
  const [connected, setConnected] = createSignal(false)

  // Poll health every 30s
  let interval: ReturnType<typeof setInterval> | undefined

  async function check() {
    const ok = await isHealthy(GO_BACKEND_URL)
    setConnected(ok)
  }

  void check()
  interval = setInterval(check, 30000)

  // Cleanup on unmount — we use a cleanup effect via onCleanup in the parent
  // Since this is a slot component re-rendered by Solid, we handle it differently
  return (
    <Box flexDirection="row" gap={1} justifyContent="space-between">
      <Text fg={api.theme.current.textMuted}>
        <Text style={{ fg: api.theme.current.success }}>⚡</Text> ApexCode v1.0.0
      </Text>
      <Show when={connected()} fallback={<Text fg={api.theme.current.textMuted}>○</Text>}>
        <Text fg={api.theme.current.success}>●</Text>
      </Show>
    </Box>
  )
}

function KairosPanel(props: { api: Parameters<TuiPlugin>[0]; sessionId: string }) {
  const { api, sessionId } = props
  const [issues, setIssues] = createSignal<KairosIssue[]>([])
  const [loading, setLoading] = createSignal(false)

  async function fetchIssues() {
    setLoading(true)
    const result = await getIssues()
    setIssues(result)
    setLoading(false)
  }

  // Fetch on mount and periodically
  void fetchIssues()

  return (
    <Box flexDirection="column" gap={1}>
      <Box>
        <Text bold fg={api.theme.current.text}>KAIROS Analysis</Text>
      </Box>
      <Show when={loading()}>
        <Text fg={api.theme.current.textMuted}>Scanning...</Text>
      </Show>
      <Show when={!loading() && issues().length > 0}>
        <Box flexDirection="column" gap={0}>
          <For each={issues().slice(0, 10)}>
            {(issue) => {
              const color = {
                critical: api.theme.current.error,
                high: api.theme.current.warning,
                medium: api.theme.current.info,
                low: api.theme.current.textMuted,
              }[issue.severity] ?? api.theme.current.textMuted

              return (
                <Text fg={color} wrap="truncate">
                  {issue.file.split("/").pop()}:{issue.line} — {issue.message.slice(0, 60)}
                </Text>
              )
            }}
          </For>
        </Box>
      </Show>
      <Show when={!loading() && issues().length === 0}>
        <Text fg={api.theme.current.success}>✓ No issues found</Text>
      </Show>
      <Show when={issues().length > 10}>
        <Text fg={api.theme.current.textMuted}>+{issues().length - 10} more</Text>
      </Show>
    </Box>
  )
}

const plugin: TuiPluginModule = {
  id: PLUGIN_ID,
  tui,
}

export default plugin
