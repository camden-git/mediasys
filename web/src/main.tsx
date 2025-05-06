import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import App from './App.tsx';
import store from './store';
import { StoreProvider } from 'easy-peasy';

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <StoreProvider store={store}>
            <App />
        </StoreProvider>
    </StrictMode>,
);
