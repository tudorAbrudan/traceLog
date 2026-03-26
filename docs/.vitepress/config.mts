import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'TraceLog',
  description: 'Lightweight server monitoring in a single binary',
  // Must match GitHub repo name in Pages URL: …github.io/traceLog/
  base: '/traceLog/',

  themeConfig: {
    logo: undefined,
    nav: [
      { text: 'Guide', link: '/guide/quickstart' },
      { text: 'GitHub', link: 'https://github.com/tudorAbrudan/tracelog' },
    ],
    sidebar: [
      {
        text: 'Getting Started',
        items: [
          { text: 'Quick Start', link: '/guide/quickstart' },
          { text: 'Configuration', link: '/guide/configuration' },
        ],
      },
      {
        text: 'Advanced',
        items: [
          { text: 'Multi-Server Setup', link: '/guide/multi-server' },
          { text: 'Alerts', link: '/guide/alerts' },
          { text: 'Reverse Proxy', link: '/guide/reverse-proxy' },
          { text: 'Logs & HTTP analytics', link: '/guide/logs-http-analytics' },
        ],
      },
    ],
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024-present Tudor Abrudan',
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/tudorAbrudan/tracelog' },
    ],
  },
})
