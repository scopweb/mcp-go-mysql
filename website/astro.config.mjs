// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
	site: 'https://mcp-mysql.docs',
	integrations: [
		starlight({
			title: 'MCP Go MySQL',
			description: 'Enterprise-Grade MySQL/MariaDB MCP Server for Claude Desktop',
			social: [
				{
					icon: 'github',
					label: 'GitHub',
					href: 'https://github.com/scopweb/mcp-go-mysql',
				},
			],
			head: [
				{
					tag: 'link',
					attrs: {
						rel: 'preconnect',
						href: 'https://use.typekit.net',
						crossorigin: 'anonymous',
					},
				},
				{
					tag: 'link',
					attrs: {
						rel: 'stylesheet',
						href: 'https://use.typekit.net/mwu3psf.css',
					},
				},
			],
			defaultLocale: 'root',
			locales: {
				root: {
					label: 'English',
					lang: 'en',
				},
				es: {
					label: 'Español',
					lang: 'es',
				},
			},
			sidebar: [
				{
					label: 'Getting Started',
					translations: { es: 'Comenzar' },
					items: [
						{
							label: 'Introduction',
							slug: 'getting-started/introduction',
							translations: { es: 'Introducción' },
						},
						{
							label: 'Configuration',
							slug: 'getting-started/configuration',
							translations: { es: 'Configuración' },
						},
					],
				},
				{
					label: 'Tools',
					translations: { es: 'Herramientas' },
					items: [
						{
							label: 'All Tools',
							slug: 'tools/overview',
							translations: { es: 'Todas las Herramientas' },
						},
					],
				},
				{
					label: 'Security',
					translations: { es: 'Seguridad' },
					items: [
						{
							label: 'Overview',
							slug: 'security/overview',
							translations: { es: 'Resumen' },
						},
					],
				},
			],
			customCss: [
				'./src/styles/custom.css',
			],
		}),
	],
});
