import { defineConfig } from "@rspress/core";
import { pluginSitemap } from "@rspress/plugin-sitemap";
import path from "node:path";

export default defineConfig({
  root: "content",
  lang: "en",
  title: "hclapi",
  description:
    "A declarative backend engine that turns HCL manifests into HTTP APIs.",
  logoText: "hclapi",
  outDir: "dist",
  globalStyles: path.join(__dirname, "theme/index.css"),
  head: [["meta", { name: "theme-color", content: "#d97706" }]],
  plugins: [
    pluginSitemap({
      siteUrl: "https://ju4n97.github.io/hclapi/",
    }),
  ],
  llms: true,
  themeConfig: {
    enableContentAnimation: false,
    enableAppearanceAnimation: false,
    nav: [
      {
        text: "Documentation",
        link: "/",
        activeMatch: "^(?!/manifest/|/steps/|/guides/|/cli/).*",
      },
      {
        text: "Manifest",
        link: "/manifest/structure",
        activeMatch: "^/manifest/",
      },
      {
        text: "Pipeline steps",
        link: "/steps/sql",
        activeMatch: "^/steps/",
      },
      {
        text: "Guides",
        link: "/guides/go",
        activeMatch: "^/guides/",
      },
      {
        text: "CLI reference",
        link: "/cli/hclapi",
        activeMatch: "^/cli/",
      },
    ],
    sidebar: {
      "/": [
        {
          text: "Getting Started",
          items: [
            { text: "Introduction", link: "/" },
            { text: "Why hclapi", link: "/why" },
            { text: "Installation", link: "/installation" },
            { text: "Quickstart", link: "/quickstart" },
          ],
        },
        {
          text: "Concepts",
          collapsible: true,
          collapsed: false,
          items: [
            { text: "Request lifecycle", link: "/concepts/lifecycle" },
            { text: "Execution context", link: "/concepts/context" },
            { text: "Pipelines and steps", link: "/concepts/pipelines" },
            { text: "Expressions", link: "/concepts/expressions" },
            { text: "Errors", link: "/concepts/errors" },
          ],
        },
        { text: "Patterns", items: [{ text: "Patterns", link: "/patterns" }] },
        { text: "Manifest", link: "/manifest/structure" },
        { text: "Pipeline steps", link: "/steps/sql" },
      ],
      "/manifest/": [
        {
          text: "Manifest",
          items: [
            { text: "Files and merging", link: "/manifest/structure" },
            { text: "server", link: "/manifest/server" },
            { text: "connection", link: "/manifest/connections" },
            { text: "schema", link: "/manifest/schemas" },
            { text: "endpoint", link: "/manifest/endpoints" },
            { text: "Scalar types", link: "/manifest/types" },
          ],
        },
        {
          text: "Functions / System",
          items: [
            { text: "env", link: "/manifest/functions/system/env" },
            { text: "uuid / uuid_v4", link: "/manifest/functions/system/uuid" },
            { text: "uuid_v7", link: "/manifest/functions/system/uuid_v7" },
            { text: "now", link: "/manifest/functions/system/now" },
          ],
        },
        {
          text: "Functions / Encoding",
          items: [
            {
              text: "json_encode",
              link: "/manifest/functions/encoding/json_encode",
            },
            {
              text: "json_decode",
              link: "/manifest/functions/encoding/json_decode",
            },
            {
              text: "base64_encode",
              link: "/manifest/functions/encoding/base64_encode",
            },
            {
              text: "base64_decode",
              link: "/manifest/functions/encoding/base64_decode",
            },
            {
              text: "url_encode",
              link: "/manifest/functions/encoding/url_encode",
            },
            {
              text: "url_decode",
              link: "/manifest/functions/encoding/url_decode",
            },
          ],
        },
        {
          text: "Functions / Strings",
          items: [
            { text: "lower", link: "/manifest/functions/strings/lower" },
            { text: "upper", link: "/manifest/functions/strings/upper" },
            {
              text: "trim_space",
              link: "/manifest/functions/strings/trim_space",
            },
            { text: "trim", link: "/manifest/functions/strings/trim" },
            {
              text: "trim_prefix",
              link: "/manifest/functions/strings/trim_prefix",
            },
            {
              text: "trim_suffix",
              link: "/manifest/functions/strings/trim_suffix",
            },
            { text: "split", link: "/manifest/functions/strings/split" },
            { text: "join", link: "/manifest/functions/strings/join" },
            { text: "replace", link: "/manifest/functions/strings/replace" },
            { text: "format", link: "/manifest/functions/strings/format" },
          ],
        },
        {
          text: "Functions / Collections & Objects",
          items: [
            {
              text: "coalesce",
              link: "/manifest/functions/collections/coalesce",
            },
            { text: "length", link: "/manifest/functions/collections/length" },
            { text: "merge", link: "/manifest/functions/collections/merge" },
            { text: "lookup", link: "/manifest/functions/collections/lookup" },
            { text: "keys", link: "/manifest/functions/collections/keys" },
            { text: "values", link: "/manifest/functions/collections/values" },
            {
              text: "contains",
              link: "/manifest/functions/collections/contains",
            },
          ],
        },
        {
          text: "Functions / Cryptography",
          items: [
            { text: "sha256", link: "/manifest/functions/cryptography/sha256" },
            { text: "md5", link: "/manifest/functions/cryptography/md5" },
            {
              text: "hmac_sha256",
              link: "/manifest/functions/cryptography/hmac_sha256",
            },
          ],
        },
        {
          text: "Functions / Math",
          items: [
            { text: "min", link: "/manifest/functions/math/min" },
            { text: "max", link: "/manifest/functions/math/max" },
            { text: "abs", link: "/manifest/functions/math/abs" },
            { text: "ceil", link: "/manifest/functions/math/ceil" },
            { text: "floor", link: "/manifest/functions/math/floor" },
            { text: "parse_int", link: "/manifest/functions/math/parse_int" },
          ],
        },
      ],
      "/steps/": [
        {
          text: "Pipeline steps",
          items: [
            { text: "sql", link: "/steps/sql" },
            { text: "starlark", link: "/steps/starlark" },
            { text: "redis", link: "/steps/redis" },
            { text: "transaction", link: "/steps/transaction" },
            { text: "parallel", link: "/steps/parallel" },
            { text: "go", link: "/steps/go" },
            { text: "respond", link: "/steps/respond" },
          ],
        },
      ],
      "/guides/": [{ text: "Go integration", link: "/guides/go" }],
      "/cli/": [
        {
          text: "CLI reference",
          items: [
            { text: "hclapi", link: "/cli/hclapi" },
            { text: "serve", link: "/cli/hclapi-serve" },
            { text: "version", link: "/cli/hclapi-version" },
          ],
        },
      ],
    },
    lastUpdated: true,
    enableScrollToTop: true,
    llmsUI: true,
    editLink: { docRepoBaseUrl: "https://github.com/ju4n97/hclapi/edit/main/" },
    socialLinks: [
      {
        icon: "github",
        mode: "link",
        content: "https://github.com/ju4n97/hclapi",
      },
    ],
    footer: { message: "Built with Rspress." },
  },
  route: {
    useTransitions: false,
  }
});
