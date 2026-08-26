import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

export default withMermaid(
  defineConfig({
    base: '/hclapi/',
    title: "Hclapi",
    description: "Declarative HTTP API runtime",
    cleanUrls: true,
    themeConfig: {
      nav: [
        { text: 'Documentation', link: '/guide/getting-started' },
        { text: 'Manifest syntax', link: '/manifest/configuration' },
        { text: 'Pipeline steps', link: '/steps/starlark' },
        { text: 'Examples', link: '/examples/overview' },
        { text: 'GitHub', link: 'https://github.com/ju4n97/hclapi' }
      ],

      sidebar: [
        {
          text: 'Getting started',
          collapsed: false,
          items: [
            { text: 'Introduction', link: '/guide/introduction' },
            { text: 'Getting started', link: '/guide/getting-started' },
            { text: 'Why Hclapi & limitations', link: '/guide/why-hclapi' },
          ]
        },
        {
          text: 'Architecture & concepts',
          collapsed: false,
          items: [
            { text: 'Pipeline state machine', link: '/guide/state-machine' },
            { text: 'The context topology', link: '/guide/context' },
            { text: 'Compilation & AST', link: '/guide/compilation' },
          ]
        },
        {
          text: 'Manifest syntax',
          collapsed: false,
          items: [
            { text: 'Global configuration', link: '/manifest/configuration' },
            { text: 'Connections & pools', link: '/manifest/connections' },
            { text: 'Schemas & validation', link: '/manifest/schemas' },
            { text: 'Endpoints & routing', link: '/manifest/endpoints' },
          ]
        },
        {
          text: 'Pipeline steps',
          collapsed: false,
          items: [
            { text: 'Starlark (Scripting)', link: '/steps/starlark' },
            { text: 'SQL (Database)', link: '/steps/sql' },
            { text: 'Redis (Caching)', link: '/steps/redis' },
            { text: 'Go (Extensions)', link: '/steps/go' },
            { text: 'Transactions', link: '/steps/transaction' },
            { text: 'Parallel execution', link: '/steps/parallel' },
            { text: 'Respond (Terminal)', link: '/steps/respond' },
          ]
        },
        {
          text: 'Go integration',
          collapsed: false,
          items: [
            { text: 'Embedding in Go', link: '/go/embedding' },
            { text: 'Custom step registry', link: '/go/step-registry' },
          ]
        },
        {
          text: 'Tooling & CLI',
          collapsed: false,
          items: [
            { text: 'CLI reference', link: '/cli/commands' },
            { text: 'OpenAPI generation', link: '/cli/openapi' },
          ]
        },
        {
          text: 'Examples catalog',
          collapsed: false,
          items: [
            { text: 'Catalog overview', link: '/examples/overview' },
            { text: '01. Zero dependency', link: '/examples/zero-dependency' },
            { text: '02. SQLite CRUD', link: '/examples/sqlite-crud' },
            { text: '03. PostgreSQL transactions', link: '/examples/postgres-transactions' },
            { text: '04. Redis caching', link: '/examples/redis-caching' },
            { text: '05. Parallel execution', link: '/examples/parallel-execution' },
            { text: '06. Go embedded plugin', link: '/examples/go-embedded' },
            { text: '07. Modular production', link: '/examples/modular-production' },
          ]
        }
      ],

      socialLinks: [
        { icon: 'github', link: 'https://github.com/ju4n97/hclapi' }
      ],

      search: {
        provider: 'local'
      },

      footer: {
        message: 'Released under the MIT License.',
        copyright: 'Copyright © 2026 Juan and contributors'
      }
    },
    mermaid: {
      theme: 'dark',
    }
  })
)
