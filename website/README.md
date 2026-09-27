# MCP Go MySQL — Documentation Website

Documentation site for MCP Go MySQL, built with [Astro](https://astro.build) and the [Starlight](https://starlight.astro.build) docs theme.

## Stack

- Astro 5.x + Starlight
- Bilingual (English + Spanish), both languages fully translated
- Static output (deployable to any static host: GitHub Pages, Vercel, Netlify, S3, …)

## Structure

```
website/
├── astro.config.mjs            # Astro + Starlight config (sidebar, locales, …)
├── src/
│   ├── content.config.ts       # Starlight collections schema
│   ├── content/
│   │   └── docs/
│   │       ├── index.mdx                    # English home
│   │       ├── getting-started/
│   │       │   ├── introduction.md
│   │       │   └── configuration.md
│   │       ├── tools/overview.md            # 10-tool reference
│   │       ├── security/overview.md         # Security model
│   │       └── es/                          # Spanish mirror of the above
│   │           ├── index.mdx
│   │           ├── getting-started/
│   │           ├── tools/
│   │           └── security/
│   ├── styles/                 # Custom CSS overrides
│   └── assets/                 # Images
├── public/                     # Static assets served verbatim
└── package.json
```

Each English doc has a Spanish counterpart at the mirrored path under `es/`. When editing, update both.

## Development

```bash
cd website
npm install
npm run dev          # http://localhost:4321
npm run build        # output → dist/
npm run preview      # serve dist/ locally
```

## Content map

| Path                                | Topic                                       |
|-------------------------------------|---------------------------------------------|
| `getting-started/introduction.md`   | What it is, flow diagram, key features      |
| `getting-started/configuration.md`  | `.env`, Claude Desktop config (Win/macOS/Linux) |
| `tools/overview.md`                 | Reference for the 10 MCP tools              |
| `security/overview.md`              | Two-layer model: MySQL grants + verb classifier |

The Spanish versions live under `es/` with the same filenames.

## Editing

1. Find the right `.md` file in `src/content/docs/...`.
2. Edit. Frontmatter (`title`, `description`) controls the page header and the sidebar label; the sidebar order itself is configured in `astro.config.mjs`.
3. Update the Spanish counterpart at `src/content/docs/es/...`.
4. `npm run dev` → check the page renders.
5. `npm run build` → confirm there are no broken links or build warnings.

When adding a new page:

- Create `src/content/docs/<section>/<page>.md` (English).
- Create `src/content/docs/es/<section>/<page>.md` (Spanish).
- Register both entries in the `sidebar` block of `astro.config.mjs`.

## License

MIT — same as the MCP Go MySQL project.
