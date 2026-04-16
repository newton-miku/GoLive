import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
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
