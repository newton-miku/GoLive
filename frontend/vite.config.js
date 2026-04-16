import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const bustCache = () => ({
  name: 'cache-bust',
  transformIndexHtml(html) {
    const v = Date.now()
    // 给所有 JS 追加随机数，强制浏览器每次拉取最新文件
    return html.replace(/(<script\s+[^>]*src=")([^"]+\.js)(")/g, (_, prefix, src, suffix) => {
      const r = Math.random().toString(36).slice(2, 8)
      return `${prefix}${src}?v=${v}-${r}${suffix}`
    })
  },
})

export default defineConfig({
  plugins: [react(), bustCache()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/status': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/recommend': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/whep': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/whip': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
