// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
	site: 'https://mcp-mysql.docs',
	integrations: [
		starlight({
			title: 'MCP Go MySQL',
			description: 'Enterprise-Grade MySQL/MariaDB MCP Server for Claude Desktop',
			expressiveCode: {
				themes: ['starlight-dark', 'starlight-light'],
			},
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
					attrs: { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
				},
				{
					tag: 'link',
					attrs: { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: true },
				},
				{
					tag: 'link',
					attrs: {
						rel: 'stylesheet',
						href: 'https://fonts.googleapis.com/css2?family=DM+Sans:ital,opsz,wght@0,9..40,300;0,9..40,400;0,9..40,500;0,9..40,600;1,9..40,400&family=Space+Mono:ital,wght@0,400;0,700;1,400&display=swap',
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
