import { describe, it, expect } from "bun:test"
import { injectContext } from "../src/enhance"

describe("injectContext", () => {
  it("returns null when Go backend is not available", async () => {
    const result = await injectContext("/tmp", "test prompt", "http://localhost:19999")
    expect(result).toBeNull()
  })

  it("returns formatted context when Go backend returns data", async () => {
    // This test requires a running Go backend. Skip if not available.
    // To run: start the Go backend first with `apex --serve`
    const result = await injectContext("/tmp", "test prompt")
    if (result) {
      expect(result).toContain("<apexcode_context>")
      expect(result).toContain("</apexcode_context>")
    }
  })
})
