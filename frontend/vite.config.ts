import {defineConfig} from 'vitest/config'
import {fileURLToPath} from 'node:url'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  // Fast Refresh injects browser-only preamble code that Vitest does not need.
  plugins: [react({fastRefresh: false})],
  build: {
    rollupOptions: {
      input: {
        app: fileURLToPath(new URL('./index.html', import.meta.url)),
        themeAtlas: fileURLToPath(new URL('./theme-atlas.html', import.meta.url)),
      },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts'
  }
})
