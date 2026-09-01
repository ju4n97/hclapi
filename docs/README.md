# hclapi docs — Rspress

This repository contains the hclapi documentation site migrated to Rspress 2.

## Development

```sh
npm install
npm run dev
```

## Production build

```sh
npm run build
npm run preview
```

## Structure

- `docs/` — Markdown/MDX documentation content
- `docs/_nav.json` — top navigation
- `docs/_meta.json` — global sidebar structure
- `theme/` — CSS customization and theme re-exports
- `rspress.config.ts` — site/build configuration

The site uses Rspress's native MDX components where they add value, including `Steps`, `Tabs`, `PackageManagerTabs`, `Callout`, and `SourceCode`. Rspress is MIT licensed and supports SSG, built-in search, i18n, multi-version docs, and `llms.txt` generation. See the official documentation for the current Rspress 2 API. 

## Deployment

`npm run build` produces static output under `doc_build/`, which can be served from static hosting or a CDN.
