import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import App from './App.tsx';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from './lib/queryClient';
import { registerProgressCallbacks } from './api/http';
import { useProgressStore } from './store/useProgressStore';

registerProgressCallbacks({
    onStart: () => useProgressStore.getState().startContinuous(),
    onComplete: () => useProgressStore.getState().setComplete(),
});

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <QueryClientProvider client={queryClient}>
            <App />
        </QueryClientProvider>
    </StrictMode>,
);
