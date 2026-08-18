import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // Same-origin from the browser's perspective, so the BFF's session
      // cookie (Path=/bff/portal, SameSite=Lax) works with plain fetch()
      // calls — no CORS handling needed in the Go BFF.
      '/bff': {
        target: 'http://localhost:8092',
        changeOrigin: true,
      },
    },
  },
});
