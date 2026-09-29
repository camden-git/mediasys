import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import App from './App.tsx';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from './lib/queryClient';
import { registerProgressCallbacks } from './api/http';
import { useProgressStore } from './store/useProgressStore';
import { registerUnauthorizedHandler } from './api/unauthorized';
import { useAuthStore } from './store/useAuthStore';

registerProgressCallbacks({
    onStart: () => useProgressStore.getState().startContinuous(),
    onComplete: () => useProgressStore.getState().setComplete(),
});

registerUnauthorizedHandler(() => {
    // ProtectedRoute redirects to /auth/login (keeping the return path) once auth is cleared.
    useAuthStore.getState().clearAuth();
});

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <QueryClientProvider client={queryClient}>
            <App />
        </QueryClientProvider>
    </StrictMode>,
);
