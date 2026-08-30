import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)
const organizationName = 'saskia-peters';
const projectName = 'app-ovsingen';

const config: Config = {
  title: 'THW OV Singen App',
  tagline: 'Operational equipment management for the THW Ortsverband Singen',
  favicon: 'img/logo.svg',

  future: {
    v4: true,
  },

  // Set the production url of your site here
  url: `https://${organizationName}.github.io`,
  baseUrl: `/${projectName}/`,

  // GitHub pages deployment config.
  organizationName,
  projectName,

  // GitHub Pages serves the subpath without a trailing-slash check unless set
  trailingSlash: true,

  onBrokenLinks: 'throw',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: 'docs',
          editUrl: `https://github.com/${organizationName}/${projectName}/tree/main/docs/`,
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themes: ['@docusaurus/theme-mermaid'],
  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  themeConfig: {
    image: 'img/docusaurus-social-card.jpg',
    colorMode: {
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'THW OV Singen App',
      logo: {
        alt: 'THW OV Singen App Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Docs',
        },
        {
          href: `https://github.com/${organizationName}/${projectName}`,
          label: 'GitHub',
          position: 'right',
        },
        {
          type: 'doc',
          docId: 'management-overview',
          label: 'Overview',
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
            {
              label: 'Intro',
              to: '/docs/intro',
            },
            {
              label: 'PRD',
              to: '/docs/planning/prd',
            },
            {
              label: 'Architecture Spine',
              to: '/docs/planning/architecture-spine',
            },
          ],
        },
        {
          title: 'Source',
          items: [
            {
              label: 'GitHub',
              href: `https://github.com/${organizationName}/${projectName}`,
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} THW OV Singen. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;