/**
 * Go Backend Client
 *
 * HTTP client for the ApexCode Go backend running on localhost:7777.
 * Used by the plugin to fetch MemPalace memory, Repomap, KAIROS suggestions,
 * and execute swarms.
 */

const DEFAULT_URL = "http://localhost:7777"

export interface EnhanceResult {
  connected: boolean
  repomap?: string
  memory?: string
  token_savings?: string
}

export interface Suggestion {
  file: string
  line: number
  severity: string
  message: string
}

export interface SuggestionsResponse {
  suggestions: Suggestion[]
  count: number
  last_scan: string
}

export interface SwarmResponse {
  status: string
  task: string
  agents: string[]
  mode: "parallel" | "sequential"
  message: string
}

export interface HealthResponse {
  status: string
  version: string
}

let _cachedHealth: { ok: boolean; at: number } = { ok: false, at: 0 }
const HEALTH_TTL = 5000 // 5s cache

/**
 * Check if Go backend is reachable (with TTL caching)
 */
export async function isHealthy(url = DEFAULT_URL): Promise<boolean> {
  const now = Date.now()
  if (now - _cachedHealth.at < HEALTH_TTL) return _cachedHealth.ok

  try {
    const resp = await fetch(`${url}/health`, { signal: AbortSignal.timeout(2000) })
    _cachedHealth = { ok: resp.ok, at: now }
    return resp.ok
  } catch {
    _cachedHealth = { ok: false, at: now }
    return false
  }
}

/**
 * Get Repomap + MemPalace context enhancement for a given prompt
 */
export async function enhance(workDir: string, prompt: string, url = DEFAULT_URL): Promise<EnhanceResult | null> {
  try {
    const resp = await fetch(`${url}/api/enhance`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ work_dir: workDir, prompt }),
      signal: AbortSignal.timeout(3000),
    })
    if (!resp.ok) return null
    return await resp.json() as EnhanceResult
  } catch {
    return null
  }
}

/**
 * Get proactive Sentinel analysis suggestions
 */
export async function getSuggestions(severity?: string, url = DEFAULT_URL): Promise<SuggestionsResponse | null> {
  try {
    const params = severity ? `?severity=${encodeURIComponent(severity)}` : ""
    const resp = await fetch(`${url}/api/suggestions${params}`, { signal: AbortSignal.timeout(3000) })
    if (!resp.ok) return null
    return await resp.json() as SuggestionsResponse
  } catch {
    return null
  }
}

/**
 * Execute multi-agent swarm
 */
export async function runSwarm(
  task: string,
  agents?: string[],
  mode?: "parallel" | "sequential",
  url = DEFAULT_URL,
): Promise<SwarmResponse | null> {
  try {
    const resp = await fetch(`${url}/api/swarm`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ task, agents: agents ?? [], mode: mode ?? "parallel" }),
      signal: AbortSignal.timeout(10000),
    })
    if (!resp.ok) return null
    return await resp.json() as SwarmResponse
  } catch {
    return null
  }
}

/**
 * Discover LM Studio models from local API
 */
export async function discoverLmStudioModels(baseUrl = "http://127.0.0.1:1234"): Promise<Array<{ id: string; name: string }>> {
  try {
    const resp = await fetch(`${baseUrl}/v1/models`, { signal: AbortSignal.timeout(3000) })
    if (!resp.ok) return []
    const data = await resp.json()
    return (data.data ?? []).map((m: { id: string }) => ({ id: m.id, name: m.id.split("/").pop() ?? m.id }))
  } catch {
    return []
  }
}
