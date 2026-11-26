import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

// Match Gin's multipart request cap so the development proxy can return 413
// without forwarding a body that Gin will reject before reading.
const maxUploadRequestBytes = 5 * 1024 * 1024 + 64 * 1024
const uploadPath = /^\/api\/v1\/(register|albums\/[^/]+\/photos)\/?(?:\?|$)/
const uploadLimit = {
  name: 'photo-album-upload-limit',
  configureServer(server) {
    server.middlewares.use((request, response, next) => {
      if (request.method !== 'POST' || !uploadPath.test(request.url) ||
          !(Number(request.headers['content-length']) > maxUploadRequestBytes)) {
        next()
        return
      }
      request.resume()
      response.writeHead(413, { 'Content-Type': 'application/json' })
      response.end(JSON.stringify({ code: 413, message: 'request is too large', data: {} }))
    })
  },
}

export default defineConfig({
  plugins: [vue(), uploadLimit],
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    proxy: {
      '/api/': 'http://127.0.0.1:8080',
      '/uploads/': 'http://127.0.0.1:8080',
    },
  },
})
