import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'
import basicSsl from '@vitejs/plugin-basic-ssl'

const useHttps = process.env.HTTPS === 'true' || process.argv.includes('--https')

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    ...(useHttps ? [basicSsl()] : []),
    tailwindcss(),
    svelte()
  ],
  server: {
    host: true, // Libera o acesso para a rede local (0.0.0.0)
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      }
    }
  }
})
