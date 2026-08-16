// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking
// (when paired with `@ts-check`).
// See: https://docusaurus.io/docs/api/docusaurus-config

import {themes as prismThemes} from 'prism-react-renderer';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

const repoUrl = 'https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go';
const pkgGoDevUrl = 'https://pkg.go.dev/github.com/NerdIT-Tech/tplink-omada-sdk-for-go';

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Omada SDK for Go',
  tagline: 'Go SDK for the TP-Link Omada Controller Open API',
  favicon: 'img/favicon.ico',

  future: {
    v4: true, // Improve compatibility with the upcoming Docusaurus v4
  },

  // GitHub Pages project-site deployment (served via GitHub Actions, see
  // ../.github/workflows/docs.yml, not the `docusaurus deploy` script).
  url: 'https://NerdIT-Tech.github.io',
  baseUrl: '/tplink-omada-sdk-for-go/',
  organizationName: 'NerdIT-Tech',
  projectName: 'tplink-omada-sdk-for-go',

  onBrokenLinks: 'throw',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        // This is the "User Guide" mode: the default docs instance, served at
        // /docs/, sourced from docs/user/ at the repo root (not copied into
        // website/ — this stays the single source of truth, still readable
        // directly on GitHub).
        docs: {
          path: '../docs/user',
          routeBasePath: 'docs',
          sidebarPath: './sidebars.user.js',
          editUrl: `${repoUrl}/tree/main/docs/user/`,
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      }),
    ],
  ],

  plugins: [
    // The "Contributor Guide" mode: a second, independent docs instance,
    // served at /contributing/, sourced from docs/contributor/. This is the
    // supported Docusaurus pattern for more than one docs "mode" on one site
    // — each instance gets its own sidebar and URL space.
    [
      '@docusaurus/plugin-content-docs',
      /** @type {import('@docusaurus/plugin-content-docs').Options} */
      ({
        id: 'contributor',
        path: '../docs/contributor',
        routeBasePath: 'contributing',
        sidebarPath: './sidebars.contributor.js',
        editUrl: `${repoUrl}/tree/main/docs/contributor/`,
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      colorMode: {
        respectPrefersColorScheme: true,
      },
      navbar: {
        title: 'Omada SDK for Go',
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'userSidebar',
            docsPluginId: 'default',
            position: 'left',
            label: 'User Guide',
          },
          {
            type: 'docSidebar',
            sidebarId: 'contributorSidebar',
            docsPluginId: 'contributor',
            position: 'left',
            label: 'Contributor Guide',
          },
          {
            href: pkgGoDevUrl,
            label: 'API Reference',
            position: 'right',
          },
          {
            href: repoUrl,
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              {label: 'User Guide', to: '/docs/'},
              {label: 'Contributor Guide', to: '/contributing/'},
            ],
          },
          {
            title: 'More',
            items: [
              {label: 'API Reference (pkg.go.dev)', href: pkgGoDevUrl},
              {label: 'GitHub', href: repoUrl},
              {label: 'Releases', href: `${repoUrl}/releases`},
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} NerdIT-Tech. Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
        additionalLanguages: ['go', 'bash', 'json'],
      },
    }),
};

export default config;
