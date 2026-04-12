/**
 * MemPalace + Repomap System Prompt Injection
 *
 * Called on each agent turn to inject codebase-aware context
 * from the Go backend into the system prompt.
 */
import { enhance as goEnhance } from "./go-backend"

export async function injectContext(
  workDir: string,
  userPrompt: string,
  backendUrl = "http://localhost:7777",
): Promise<string | null> {
  const result = await goEnhance(workDir, userPrompt, backendUrl)
  if (!result?.connected) return null

  const sections: string[] = []

  if (result.repomap) {
    sections.push(
      "<repository_map>",
      "The following is a repository map showing the most relevant files for your current task.",
      "This map is built using PageRank analysis of code dependencies.",
      result.repomap,
      "</repository_map>",
    )
  }

  if (result.memory) {
    sections.push(
      "<memory_context>",
      "The following is relevant context from your memory (MemPalace system).",
      "This knowledge was retrieved from previous sessions and interactions.",
      result.memory,
      "</memory_context>",
    )
  }

  if (sections.length === 0) return null

  return [
    "<apexcode_context>",
    "You are augmented with ApexCode's codebase intelligence.",
    `Token savings from context injection: ${result.token_savings ?? "N/A"}`,
    ...sections,
    "</apexcode_context>",
  ].join("\n")
}
