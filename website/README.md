# MCP Go MySQL - Documentation Website

Documentation site for MCP Go MySQL built with Astro.

## Features

- Bilingual: Spanish (complete) and English (structure)
- Astro 5.x
- Responsive Design
- Pure CSS (no frameworks)
- Fast static generation
- Starlight Theme

## Structure

```
website/
├── src/
│   ├── layouts/
│   │   └── Layout.astro      # Main layout with styles
│   └── pages/
│       ├── index.astro        # Home (Spanish)
│       ├── es/                # Spanish pages
│       │   ├── funciones.astro
│       │   ├── configuracion.astro
│       │   └── seguridad.astro
│       └── en/                # English pages (structure)
│           ├── index.astro
│           ├── functions.astro
│           ├── configuration.astro
│           └── security.astro
└── public/                    # Static assets
```

## Development

### Install dependencies

```bash
cd website
npm install
```

### Run development server

```bash
npm run dev
```

The site will be available at `http://localhost:4321`

### Build for production

```bash
npm run build
```

The generated files will be in the `dist/` directory

### Preview production build

```bash
npm run preview
```

## Content

### Spanish (Complete)

- **Home**: General overview of MCP, features and status
- **Functions**: Documentation of the 10 available tools
- **Configuration**: Step-by-step guide for Windows, macOS and Linux
- **Security**: Detailed description of the 6 security phases

### English (Structure)

All English pages have the complete structure but content is marked as `[Content to be added]` to be completed later.

## Customization

### Changing colors

Edit CSS variables in `src/layouts/Layout.astro`:

```css
:root {
  --primary: #2563eb;
  --primary-dark: #1e40af;
  --success: #10b981;
  --warning: #f59e0b;
  --danger: #ef4444;
}
```

### Adding new pages

1. Create file in `src/pages/es/new-page.astro`
2. Add link to the menu on all pages
3. Follow the same structure pattern

## Navigation

The site includes:
- Language selector in the header (ES/EN)
- Consistent navigation menu on all pages
- Footer with project information

## Technologies

- [Astro](https://astro.build) - Static site framework
- TypeScript - Type checking
- Pure CSS - Styles without dependencies

## License

MIT - Same as MCP Go MySQL project