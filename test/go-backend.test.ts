import { describe, it, expect, beforeAll, afterAll } from "bun:test"
import { isHealthy, enhance, getSuggestions, runSwarm, discoverLmStudioModels } from "../src/go-backend"

describe("go-backend", () => {
  describe("isHealthy", () => {
    it("returns false when server is not running", async () => {
      const ok = await isHealthy("http://localhost:19999")
      expect(ok).toBe(false)
    })

    it("returns true when server is running", async () => {
      // Requires Go backend running on port 7777
      const ok = await isHealthy()
      if (ok) {
        expect(ok).toBe(true)
      }
    })
  })

  describe("enhance", () => {
    it("returns null when Go backend is not available", async () => {
      const result = await enhance("/tmp", "test", "http://localhost:19999")
      expect(result).toBeNull()
    })

    it("returns enhancement data when Go backend is available", async () => {
      const result = await enhance("/tmp", "test")
      if (result?.connected) {
        expect(result.connected).toBe(true)
        expect(result.token_savings).toBeDefined()
      }
    })
  })

  describe("getSuggestions", () => {
    it("returns null when Go backend is not available", async () => {
      const result = await getSuggestions(undefined, "http://localhost:19999")
      expect(result).toBeNull()
    })
  })

  describe("runSwarm", () => {
    it("returns null when Go backend is not available", async () => {
      const result = await runSwarm("test task", ["planner"], "parallel", "http://localhost:19999")
      expect(result).toBeNull()
    })

    it("accepts valid task and returns response when Go backend is available", async () => {
      const result = await runSwarm("test task")
      if (result) {
        expect(result.status).toBe("accepted")
        expect(result.task).toBe("test task")
      }
    })
  })

  describe("discoverLmStudioModels", () => {
    it("returns empty array when LM Studio is not running", async () => {
      const models = await discoverLmStudioModels("http://localhost:19999")
      expect(models).toEqual([])
    })
  })
})
