/**
 * LM Studio Model Discovery
 *
 * Discovers models from a running LM Studio instance
 * and returns them in OpenCode provider format.
 */
import { discoverLmStudioModels } from "./go-backend"

export interface LmStudioModel {
  id: string
  name: string
  url: string
}

export async function discoverModels(baseUrl = "http://127.0.0.1:1234"): Promise<LmStudioModel[]> {
  const models = await discoverLmStudioModels(baseUrl)
  return models.map((m) => ({
    id: m.id,
    name: m.name,
    url: `${baseUrl}/v1`,
  }))
}
