// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking
// (when paired with `@ts-check`).
// There are various equivalent ways to declare your Docusaurus config.
// See: https://docusaurus.io/docs/api/docusaurus-config

import {themes as prismThemes} from 'prism-react-renderer';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

// This same build output backs two different deployments that need different
// baseUrls: the Dockerfile bundles it into the daemon image, served at
// /docs/ by cmd/mcpd/main.go's http.StripPrefix("/docs/", ...); GitHub Pages
// serves the whole gh-pages branch at the repo root, i.e. /linux-mcp-daemon/.
// `npm run deploy` (docs/website/package.json) sets this env var; the
// Dockerfile's plain `npm run build` doesn't, so it keeps the /docs/ default.
const isGithubPagesDeploy = process.env.DOCS_DEPLOY_TARGET === 'github-pages';

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Linux MCPd',
  tagline: 'Securely bridging AI agents to the Linux host via MCP',
  favicon: 'img/favicon.ico',

  // Future flags, see https://docusaurus.io/docs/api/docusaurus-config#future
  future: {
    v4: true, // Improve compatibility with the upcoming Docusaurus v4
  },

  // Set the production url of your site here
  url: 'https://nucleusv.github.io',
  // Set the /<baseUrl>/ pathname under which your site is served
  baseUrl: isGithubPagesDeploy ? '/linux-mcp-daemon/' : '/docs/',

  // docs/imgs holds images shared with the repo README (e.g. the
  // architecture diagram), served from here too so there's one copy.
  staticDirectories: ['static', '../imgs'],

  // GitHub pages deployment config (also used to build "Edit this page" links).
  organizationName: 'nucleusv', // Usually your GitHub org/user name.
  projectName: 'linux-mcp-daemon', // Usually your repo name.
  deploymentBranch: 'gh-pages',

  onBrokenLinks: 'throw',

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: './sidebars.js',
          routeBasePath: '/',
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            'https://github.com/nucleusv/linux-mcp-daemon/tree/main/docs/website/',
        },
        blog: {
          showReadingTime: true,
          feedOptions: {
            type: ['rss', 'atom'],
            xslt: true,
          },
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            'https://github.com/nucleusv/linux-mcp-daemon/tree/main/docs/website/',
          // Useful options to enforce blogging best practices
          onInlineTags: 'warn',
          onInlineAuthors: 'warn',
          onUntruncatedBlogPosts: 'warn',
        },
        theme: {
          customCss: './src/css/custom.css',
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      // Replace with your project's social card
      image: 'img/docusaurus-social-card.jpg',
      colorMode: {
        respectPrefersColorScheme: true,
      },
      navbar: {
        title: 'Linux MCPd',
        logo: {
          alt: 'Logo',
          src: 'img/logo.svg',
        },
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'tutorialSidebar',
            position: 'left',
            label: 'Documentation',
          },
          {
            href: 'https://github.com/nucleusv/linux-mcp-daemon',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        copyright: `Copyright © ${new Date().getFullYear()} NucleusV. Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
      },
    }),
};

export default config;
