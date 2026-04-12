/**
 * KAIROS Proactive Analysis Suggestions
 *
 * Fetches code quality issues from the Go backend
 * and formats them for display in the TUI.
 */
import { getSuggestions as goGetSuggestions, type Suggestion } from "./go-backend"

export interface KairosIssue {
  file: string
  line: number
  severity: "low" | "medium" | "high" | "critical"
  message: string
}

export async function getIssues(severity?: string): Promise<KairosIssue[]> {
  const result = await goGetSuggestions(severity)
  if (!result) return []

  return result.suggestions.map((s: Suggestion) => ({
    file: s.file,
    line: s.line,
    severity: s.severity.toLowerCase() as KairosIssue["severity"],
    message: s.message,
  }))
}

export function formatIssue(issue: KairosIssue): string {
  const icon = {
    low: "ℹ",
    medium: "⚠",
    high: "△",
    critical: "✗",
  }[issue.severity] ?? "•"

  return `${icon} ${issue.file}:${issue.line} — ${issue.message}`
}

export function formatIssues(issues: KairosIssue[]): string {
  if (issues.length === 0) return "No issues found."

  const bySeverity: Record<string, KairosIssue[]> = {}
  for (const issue of issues) {
    (bySeverity[issue.severity] ??= []).push(issue)
  }

  const order = ["critical", "high", "medium", "low"]
  const lines: string[] = []

  for (const sev of order) {
    const group = bySeverity[sev]
    if (!group?.length) continue
    lines.push(`\n--- ${sev.toUpperCase()} (${group.length}) ---`)
    for (const issue of group) {
      lines.push(formatIssue(issue))
    }
  }

  return lines.join("\n")
}
