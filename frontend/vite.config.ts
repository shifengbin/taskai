import {defineConfig} from 'vitest/config'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  // Fast Refresh injects browser-only preamble code that Vitest does not need.
  plugins: [react({fastRefresh: false})],
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts'
  }
})
