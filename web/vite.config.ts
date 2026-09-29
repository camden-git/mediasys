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
    // define: {
    //     // Ensure environment variables are available at build time
    //     'import.meta.env.VITE_API_URL': JSON.stringify(process.env.VITE_API_URL || '/api'),
    //     'import.meta.env.VITE_BACKEND_URL': JSON.stringify(process.env.VITE_BACKEND_URL || '/api'),
    //     'import.meta.env.VITE_TURNSTILE_SITE_KEY': JSON.stringify(process.env.VITE_TURNSTILE_SITE_KEY || ''),
    // },
});
