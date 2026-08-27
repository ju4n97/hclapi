import { defineConfig } from 'vitepress';
import { withMermaid } from 'vitepress-plugin-mermaid';

export default withMermaid(
  defineConfig({
    base: '/hclapi/',
    title: 'Hclapi',
    description: 'Declarative HTTP API runtime engine',
    cleanUrls: true,
    lastUpdated: true,

    themeConfig: {
      siteTitle: 'Hclapi',

      nav: [
        { 
          text: 'Guide', 
          link: '/guide/introduction',
          activeMatch: '^/(guide|concepts)/' 
        },
        { 
          text: 'Reference', 
          link: '/manifest/structure',
          activeMatch: '^/(manifest|steps|cli)/' 
        },
        { 
          text: 'Go integration', 
          link: '/go/embedding',
          activeMatch: '^/go/' 
        },
        { 
          text: 'Examples', 
          link: 'https://github.com/ju4n97/hclapi/tree/main/examples' 
        },
        {
          text: 'v0.1.0',
          items: [
            { text: 'Changelog', link: 'https://github.com/ju4n97/hclapi/releases' },
            { text: 'Roadmap', link: '/guide/roadmap' },
          ]
        }
      ],

      sidebar: [
        {
          text: 'Getting started',
          collapsed: false,
          items: [
            { text: 'Introduction', link: '/guide/introduction' },
            { text: 'Quickstart (5 mins)', link: '/guide/quickstart' },
          ]
        },
        {
          text: 'Core concepts',
          collapsed: false,
          items: [
            { text: 'The mental model & lifecycle', link: '/concepts/mental-model' },
            { text: 'The context topology', link: '/concepts/context' },
            { text: 'Dynamic expression evaluation', link: '/concepts/expressions' },
            { text: 'Error standards (RFC 9457)', link: '/concepts/errors' },
          ]
        },
        {
          text: 'Manifest reference',
          collapsed: false,
          items: [
            { text: 'Files & tree merging', link: '/manifest/structure' },
            { text: 'Server block', link: '/manifest/server' },
            { text: 'Scalar types', link: '/manifest/types' },
            { text: 'Connections & pooling', link: '/manifest/connections' },
            { text: 'Schemas & payload validation', link: '/manifest/schemas' },
            { text: 'Endpoints & route patterns', link: '/manifest/endpoints' },
          ]
        },
        {
          text: 'Pipeline steps',
          collapsed: false,
          items: [
            { text: 'Steps overview', link: '/steps/overview' },
            { text: 'Starlark', link: '/steps/starlark' },
            { text: 'SQL', link: '/steps/sql' },
            { text: 'Redis', link: '/steps/redis' },
            { text: 'Go ', link: '/steps/go' },
            { text: 'Transaction', link: '/steps/transaction' },
            { text: 'Parallel', link: '/steps/parallel' },
            { text: 'Respond', link: '/steps/respond' },
          ]
        },
        {
          text: 'Go SDK & embedding',
          collapsed: false,
          items: [
            { text: 'Embedding in http.ServeMux', link: '/go/embedding' },
            { text: 'Registering custom Go steps', link: '/go/step-registry' },
            { text: 'Custom error handlers', link: '/go/error-handling' },
            { text: 'Options & logging', link: '/go/options' },
          ]
        },
        {
          text: 'CLI & tooling',
          collapsed: false,
          items: [
            { text: 'hclapi serve (Server)', link: '/cli/serve' },
            { text: 'hclapi validate (Linter)', link: '/cli/validate' },
            { text: 'hclapi openapi (Spec generator)', link: '/cli/openapi' },
          ]
        }
      ],

      socialLinks: [
        { icon: 'github', link: 'https://github.com/ju4n97/hclapi' }
      ],

      search: {
        provider: 'local',
        options: {
          detailedView: true
        }
      },

      editLink: {
        pattern: 'https://github.com/ju4n97/hclapi/edit/main/docs/:path',
        text: 'Edit this page on GitHub'
      },

      footer: {
        message: 'Released under the MIT License.',
        copyright: 'Copyright © 2026 Juan Mesa and contributors'
      }
    },

    mermaid: {
      theme: 'dark',   
    }
  })
)