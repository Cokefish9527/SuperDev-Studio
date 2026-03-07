import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

function sanitizeChunkName(name: string) {
  return name.replace(/@/g, '').replace(/[^a-zA-Z0-9-_]/g, '-')
}

function normalizeVendorChunkName(id: string) {
  const normalized = id.replaceAll(String.fromCharCode(92), '/')
  const packagePath = normalized.split('/node_modules/').pop() ?? ''
  const segments = packagePath.split('/')
  const packageName = segments[0]?.startsWith('@')
    ? `${segments[0]}-${segments[1] ?? 'pkg'}`
    : segments[0] ?? 'vendor'

  return `vendor-${sanitizeChunkName(packageName)}`
}

function manualChunks(id: string) {
  if (!id.includes('node_modules')) {
    return undefined
  }

  const normalized = id.replaceAll(String.fromCharCode(92), '/')
  const packagePath = normalized.split('/node_modules/').pop() ?? ''

  if (
    packagePath.startsWith('react-router-dom/') ||
    packagePath.startsWith('react-router/') ||
    packagePath.startsWith('@remix-run/')
  ) {
    return 'vendor-router'
  }

  if (packagePath.startsWith('@tanstack/')) {
    return 'vendor-query'
  }

  if (
    packagePath.startsWith('axios/') ||
    packagePath.startsWith('dayjs/') ||
    packagePath.startsWith('cookie/') ||
    packagePath.startsWith('json2mq/') ||
    packagePath.startsWith('set-cookie-parser/') ||
    packagePath.startsWith('string-convert/')
  ) {
    return 'vendor-utils'
  }

  if (
    packagePath.startsWith('react/') ||
    packagePath.startsWith('react-dom/') ||
    packagePath.startsWith('scheduler/')
  ) {
    return 'vendor-react'
  }

  return normalizeVendorChunkName(id)
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: '127.0.0.1',
    port: 5273,
    strictPort: true,
  },
  preview: {
    host: '127.0.0.1',
    port: 5273,
    strictPort: true,
  },
  build: {
    chunkSizeWarningLimit: 550,
    rollupOptions: {
      output: {
        manualChunks,
      },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    globals: true,
  },
})
