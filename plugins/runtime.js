/**
 * ApexCode TypeScript Plugin Runtime
 * 
 * This runtime loads and executes TypeScript plugins that extend
 * ApexCode's capabilities with custom tools and MCP servers.
 */

const path = require('path');
const fs = require('fs');

class PluginRuntime {
  constructor() {
    this.plugins = [];
    this.tools = new Map();
    this.mcpServers = new Map();
  }

  /**
   * Initialize the plugin runtime
   */
  async initialize() {
    console.log('[PluginRuntime] Initializing...');
    
    // Load plugins from directories
    const pluginDirs = [
      path.join(__dirname, 'mcp-servers'),
      path.join(__dirname, 'custom-tools'),
      path.join(__dirname, 'extensions'),
    ];

    for (const dir of pluginDirs) {
      if (fs.existsSync(dir)) {
        await this.loadPluginsFromDir(dir);
      }
    }

    console.log(`[PluginRuntime] Loaded ${this.plugins.length} plugins`);
  }

  /**
   * Load plugins from a directory
   */
  async loadPluginsFromDir(dir) {
    const files = fs.readdirSync(dir);
    
    for (const file of files) {
      if (file.endsWith('.ts') || file.endsWith('.js')) {
        const pluginPath = path.join(dir, file);
        try {
          const plugin = require(pluginPath);
          if (plugin.name && plugin.init) {
            this.plugins.push(plugin);
            await plugin.init(this);
            console.log(`[PluginRuntime] Loaded plugin: ${plugin.name}`);
          }
        } catch (err) {
          console.error(`[PluginRuntime] Failed to load ${file}:`, err.message);
        }
      }
    }
  }

  /**
   * Register a custom tool
   */
  registerTool(name, description, schema, handler) {
    this.tools.set(name, {
      name,
      description,
      schema,
      handler,
    });
  }

  /**
   * Register an MCP server
   */
  registerMCPServer(name, url) {
    this.mcpServers.set(name, { name, url });
  }

  /**
   * Execute a tool
   */
  async executeTool(name, args) {
    const tool = this.tools.get(name);
    if (!tool) {
      throw new Error(`Tool not found: ${name}`);
    }
    return await tool.handler(args);
  }

  /**
   * Get all registered tools
   */
  getTools() {
    return Array.from(this.tools.values()).map(t => ({
      name: t.name,
      description: t.description,
      schema: t.schema,
    }));
  }

  /**
   * Get all MCP servers
   */
  getMCPServers() {
    return Array.from(this.mcpServers.values());
  }
}

// Export for plugins
const runtime = new PluginRuntime();
module.exports = runtime;

// Start if run directly
if (require.main === module) {
  runtime.initialize().catch(console.error);
}
