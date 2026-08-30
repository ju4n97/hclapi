import { defineConfig, markdown } from "sourcey";

export default defineConfig({
  name: "hclapi",
  prettyUrls: "slash",
  repo: "https://github.com/ju4n97/hclapi",
  editBranch: "main",  
  theme: {
    preset: "api-first",
    colors: {
      primary: "#FFB000",
      light: "#FFD166",
      dark: "#B36B00",
    },
    fonts: {
      sans: "Inter",
      mono: "JetBrains Mono",
    },
    layout: {
      sidebar: "17rem",
      toc: "0rem",
      content: "46rem",
    },
    css: ["./custom.css"],
  },
  search: {
    featured: [
      "introduction",
      "quickstart",
      "concepts",
      "manifest",
      "steps",
      "guides/go",
      "cli",
    ],
  },
  navigation: {
    tabs: [
      {
        tab: "Documentation",
        slug: "",
        source: markdown({
          groups: [
            {
              group: "Getting Started",
              pages: [
                "introduction",
                "why",
                "installation",
                "quickstart",
              ],
            },
            {
              group: "Concepts",
              pages: [
                "concepts/lifecycle",
                "concepts/context",
                "concepts/pipelines",
                "concepts/expressions",
                "concepts/errors",
              ],
            },
            {
              group: "Patterns",
              pages: ["patterns"],
            },
          ],
        }),
      },
      {
        tab: "Manifest",
        slug: "manifest",
        source: markdown({
          groups: [
            {
              group: "Manifest",
              pages: [
                "manifest/structure",
                "manifest/server",
                "manifest/connections",
                "manifest/schemas",
                "manifest/endpoints",
                "manifest/types",
              ],
            },
            {
              group: "Functions / System",
              pages: [
                "manifest/functions/system/env",
                "manifest/functions/system/uuid",
                "manifest/functions/system/uuid_v7",
                "manifest/functions/system/now",
              ],
            },
            {
              group: "Functions / Encoding",
              pages: [
                "manifest/functions/encoding/json_encode",
                "manifest/functions/encoding/json_decode",
                "manifest/functions/encoding/base64_encode",
                "manifest/functions/encoding/base64_decode",
                "manifest/functions/encoding/url_encode",
                "manifest/functions/encoding/url_decode",
              ],
            },
            {
              group: "Functions / Strings",
              pages: [
                "manifest/functions/strings/lower",
                "manifest/functions/strings/upper",
                "manifest/functions/strings/trim_space",
                "manifest/functions/strings/trim",
                "manifest/functions/strings/trim_prefix",
                "manifest/functions/strings/trim_suffix",
                "manifest/functions/strings/split",
                "manifest/functions/strings/join",
                "manifest/functions/strings/replace",
                "manifest/functions/strings/format",
              ],
            },
            {
              group: "Functions / Collections & Objects",
              pages: [
                "manifest/functions/collections/coalesce",
                "manifest/functions/collections/length",
                "manifest/functions/collections/merge",
                "manifest/functions/collections/lookup",
                "manifest/functions/collections/keys",
                "manifest/functions/collections/values",
                "manifest/functions/collections/contains",
              ],
            },
            {
              group: "Functions / Cryptography",
              pages: [
                "manifest/functions/cryptography/sha256",
                "manifest/functions/cryptography/md5",
                "manifest/functions/cryptography/hmac_sha256",
              ],
            },
            {
              group: "Functions / Math",
              pages: [
                "manifest/functions/math/min",
                "manifest/functions/math/max",
                "manifest/functions/math/abs",
                "manifest/functions/math/ceil",
                "manifest/functions/math/floor",
                "manifest/functions/math/parse_int",
              ],
            },
          ],
        }),
      },
      {
        tab: "Pipeline steps",
        slug: "steps",
        source: markdown({
          groups: [
            {
              group: "Pipeline steps",
              pages: [
                "steps/sql",
                "steps/starlark",
                "steps/redis",
                "steps/transaction",
                "steps/parallel",
                "steps/go",
                "steps/respond",
              ],
            },
          ],
        }),
      },
      {
        tab: "Guides",
        slug: "guides",
        source: markdown({
          groups: [{ group: "Guides", pages: ["guides/go"] }],
        }),
      },
      {
        tab: "CLI reference",
        slug: "cli",
        source: markdown({
          groups: [
            {
              group: "Reference",
              pages: ["cli/hclapi.md"],
            },
            {
              group: "Commands",
              pages: ["cli/hclapi-serve.md", "cli/hclapi-version.md"],
            },
          ],
        }),
      },      
    ],
  },
  navbar: {
    links: [
      {
        type: "github",
        href: "https://github.com/ju4n97/hclapi",
      },
    ],
    primary: {
      type: "button",
      label: "Install",
      href: "/installation",
    },
  },
  footer: {
    links: [
      {
        type: "github",
        href: "https://github.com/ju4n97/hclapi",
      },
    ],
  },
});
