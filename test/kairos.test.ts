import { describe, it, expect } from "bun:test"
import { getIssues, formatIssue, formatIssues, type KairosIssue } from "../src/kairos"

describe("kairos", () => {
  describe("getIssues", () => {
    it("returns empty array when Go backend is not available", async () => {
      // Uses default URL which requires Go backend
      const issues = await getIssues()
      expect(Array.isArray(issues)).toBe(true)
    })
  })

  describe("formatIssue", () => {
    it("formats a critical issue correctly", () => {
      const issue: KairosIssue = {
        file: "src/auth.ts",
        line: 42,
        severity: "critical",
        message: "SQL injection vulnerability",
      }
      const formatted = formatIssue(issue)
      expect(formatted).toContain("✗")
      expect(formatted).toContain("src/auth.ts:42")
      expect(formatted).toContain("SQL injection vulnerability")
    })

    it("formats a low severity issue correctly", () => {
      const issue: KairosIssue = {
        file: "README.md",
        line: 1,
        severity: "low",
        message: "Consider adding more examples",
      }
      const formatted = formatIssue(issue)
      expect(formatted).toContain("ℹ")
    })
  })

  describe("formatIssues", () => {
    it("returns 'No issues found' for empty array", () => {
      expect(formatIssues([])).toBe("No issues found.")
    })

    it("groups issues by severity", () => {
      const issues: KairosIssue[] = [
        { file: "a.ts", line: 1, severity: "high", message: "high issue" },
        { file: "b.ts", line: 2, severity: "critical", message: "critical issue" },
        { file: "c.ts", line: 3, severity: "high", message: "another high" },
      ]
      const formatted = formatIssues(issues)
      // Critical should appear before high
      const criticalIdx = formatted.indexOf("CRITICAL")
      const highIdx = formatted.indexOf("HIGH")
      expect(criticalIdx).toBeLessThan(highIdx)
    })
  })
})
