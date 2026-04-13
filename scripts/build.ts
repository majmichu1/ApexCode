#!/usr/bin/env bun
import { $ } from "bun"
import path from "path"

const root = path.join(import.meta.dir, "..")
const src = path.join(root, "src")
const dist = path.join(root, "dist")

await $`rm -rf ${dist}`
await $`mkdir -p ${dist}`

// Use bun build for TypeScript → ESM
await Bun.build({
  entrypoints: [
    path.join(src, "server.ts"),
  ],
  outdir: dist,
  target: "node",
  format: "esm",
  external: [
    "@opencode-ai/plugin",
    "@opencode-ai/plugin/tui",
    "@opencode-ai/plugin/tool",
    "@opentui/core",
    "@opentui/solid",
    "solid-js",
    "solid-js/web",
    "solid-js/store",
  ],
  minify: false,
  sourcemap: "external",
})

console.log("✓ Built to dist/")
