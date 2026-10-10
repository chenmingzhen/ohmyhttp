import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

const sidebar = [
  {
    text: '开始',
    items: [
      { text: '概述', link: '/' },
      { text: '快速开始', link: '/guide/getting-started' },
      { text: '请求处理流程', link: '/guide/lifecycle' },
    ],
  },
  {
    text: '路由',
    items: [
      { text: '路由注册', link: '/guide/routing' },
      { text: '处理函数', link: '/guide/handlers' },
    ],
  },
  {
    text: '请求',
    items: [
      { text: '请求绑定', link: '/guide/binding' },
      { text: '请求校验', link: '/guide/validation' },
    ],
  },
  {
    text: '响应',
    items: [
      { text: '响应编码', link: '/guide/responses' },
      { text: '错误处理', link: '/guide/errors' },
      { text: '响应转换', link: '/guide/envelope' },
    ],
  },
  {
    text: '中间件',
    items: [
      { text: '中间件', link: '/guide/middleware' },
      { text: '可观测性', link: '/guide/observability' },
      { text: 'Panic 恢复', link: '/guide/recovery' },
    ],
  },
  {
    text: '静态文件与文档',
    items: [
      { text: '静态文件', link: '/guide/static' },
      { text: 'OpenAPI', link: '/guide/openapi' },
    ],
  },
  {
    text: '运行',
    items: [
      { text: 'HTTP 服务器', link: '/guide/server' },
      { text: '运行环境', link: '/guide/environment' },
    ],
  },
  {
    text: '参考',
    items: [
      { text: '应用选项', link: '/reference/options' },
      { text: '路由元数据', link: '/reference/metadata' },
      { text: '错误类型', link: '/reference/errors' },
      { text: '示例', link: '/reference/examples' },
    ],
  },
]

export default withMermaid(defineConfig({
  title: 'ohmyhttp',
  description: '构建在 Go net/http 上的 HTTP 框架。',
  base: process.env.GITHUB_ACTIONS ? '/ohmyhttp/' : '/',
  lang: 'zh-CN',
  cleanUrls: true,
  lastUpdated: true,
  themeConfig: {
    outline: { level: [2, 3], label: '本页目录' },
    docFooter: { prev: '上一页', next: '下一页' },
    lastUpdated: { text: '更新于' },
    darkModeSwitchLabel: '外观',
    sidebarMenuLabel: '目录',
    returnToTopLabel: '回到顶部',
    search: { provider: 'local' },
    nav: [
      { text: '指南', link: '/guide/getting-started' },
      { text: '参考', link: '/reference/options' },
      { text: '示例', link: '/reference/examples' },
    ],
    sidebar,
    socialLinks: [
      { icon: 'github', link: 'https://github.com/chenmingzhen/ohmyhttp' },
    ],
  },
}))
