import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react-swc';
import tailwindcss from '@tailwindcss/vite';

// https://vite.dev/config/
export default defineConfig({
    plugins: [react(), tailwindcss()],
    server: {
        proxy: {
            '/api': {
                target: process.env.VITE_DEV_API_PROXY_TARGET || 'http://localhost:8080',
                changeOrigin: true,
                ws: true,
            },
        },
    },
});
