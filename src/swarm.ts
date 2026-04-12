/**
 * Multi-Agent Swarm Tool
 *
 * Executes a task across multiple specialized agents
 * via the Go backend's /api/swarm endpoint.
 */
import { runSwarm as goRunSwarm } from "./go-backend"

export const AVAILABLE_AGENTS = [
  { id: "planner", name: "Planner", desc: "Break down tasks into actionable steps" },
  { id: "architect", name: "Architect", desc: "Design architecture and define interfaces" },
  { id: "coder", name: "Coder", desc: "Write clean, efficient code" },
  { id: "reviewer", name: "Reviewer", desc: "Code review for correctness and security" },
  { id: "tester", name: "Tester", desc: "Write comprehensive tests" },
  { id: "documenter", name: "Documenter", desc: "Create documentation and guides" },
] as const

export interface SwarmResult {
  status: string
  message: string
  agents: string[]
  mode: "parallel" | "sequential"
}

export async function executeSwarm(
  task: string,
  agents: string[] = ["planner", "coder", "reviewer"],
  mode: "parallel" | "sequential" = "parallel",
): Promise<SwarmResult | null> {
  const result = await goRunSwarm(task, agents, mode)
  if (!result) return null

  return {
    status: result.status,
    message: result.message,
    agents: result.agents,
    mode: result.mode,
  }
}

export function formatSwarmResult(result: SwarmResult): string {
  return [
    `Swarm: ${result.status}`,
    `Agents: ${result.agents.join(", ")}`,
    `Mode: ${result.mode}`,
    result.message,
  ].join("\n")
}
