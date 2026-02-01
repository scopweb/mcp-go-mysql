# MCP Go MySQL - Documentation Website

Sitio web de documentación para MCP Go MySQL construido con Astro.

## Características

- ✅ Bilingüe: Español (completo) e Inglés (estructura)
- 🚀 Astro 5.x
- 📱 Responsive Design
- 🎨 CSS puro (sin frameworks)
- ⚡ Generación estática rápida

## Estructura

```
website/
├── src/
│   ├── layouts/
│   │   └── Layout.astro      # Layout principal con estilos
│   └── pages/
│       ├── index.astro        # Inicio (Español)
│       ├── es/                # Páginas en español
│       │   ├── funciones.astro
│       │   ├── configuracion.astro
│       │   └── seguridad.astro
│       └── en/                # Páginas en inglés (estructura)
│           ├── index.astro
│           ├── functions.astro
│           ├── configuration.astro
│           └── security.astro
└── public/                    # Archivos estáticos
```

## Desarrollo

### Instalar dependencias

```bash
cd website
npm install
```

### Ejecutar servidor de desarrollo

```bash
npm run dev
```

El sitio estará disponible en `http://localhost:4321`

### Construir para producción

```bash
npm run build
```

Los archivos generados estarán en el directorio `dist/`

### Preview de producción

```bash
npm run preview
```

## Contenido

### Español (Completo)

- **Inicio**: Descripción general del MCP, características y estado
- **Funciones**: Documentación de las 10 herramientas disponibles
- **Configuración**: Guía paso a paso para Windows, macOS y Linux
- **Seguridad**: Descripción detallada de las 6 fases de seguridad

### Inglés (Estructura)

Todas las páginas en inglés tienen la estructura completa pero el contenido está marcado como `[Content to be added]` para ser completado posteriormente.

## Personalización

### Cambiar colores

Edita las variables CSS en `src/layouts/Layout.astro`:

```css
:root {
  --primary: #2563eb;
  --primary-dark: #1e40af;
  --success: #10b981;
  --warning: #f59e0b;
  --danger: #ef4444;
}
```

### Agregar nuevas páginas

1. Crear archivo en `src/pages/es/nueva-pagina.astro`
2. Agregar enlace al menú en todas las páginas
3. Seguir el mismo patrón de estructura

## Navegación

El sitio incluye:
- Selector de idioma en el header (ES/EN)
- Menú de navegación consistente en todas las páginas
- Footer con información del proyecto

## Tecnologías

- [Astro](https://astro.build) - Framework de sitios estáticos
- TypeScript - Type checking
- CSS puro - Estilos sin dependencias

## Licencia

MIT - Mismo que el proyecto MCP Go MySQL
