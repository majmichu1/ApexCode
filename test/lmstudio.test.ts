import { describe, it, expect } from "bun:test"
import { discoverModels } from "../src/lmstudio"

describe("discoverModels", () => {
  it("returns empty array when LM Studio is not running", async () => {
    const models = await discoverModels("http://localhost:19999")
    expect(models).toEqual([])
  })

  it("returns models when LM Studio is running", async () => {
    // Requires LM Studio running on default port
    const models = await discoverModels()
    if (models.length > 0) {
      expect(models[0]).toHaveProperty("id")
      expect(models[0]).toHaveProperty("name")
      expect(models[0]).toHaveProperty("url")
      expect(models[0].url).toContain("/v1")
    }
  })
})
