import { defineConfig } from 'astro/config';

export default defineConfig({
  site: 'https://mcp-mysql.docs',
  base: '/',
  i18n: {
    defaultLocale: 'es',
    locales: ['es', 'en'],
    routing: {
      prefixDefaultLocale: false
    }
  }
});
