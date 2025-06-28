import React, { useEffect } from 'react';
import { Routes, Route, BrowserRouter } from 'react-router-dom';
import IndexRouter from './routes/IndexRouter';
import AlbumRouter from './routes/AlbumRouter';
import AuthRouter from './routes/AuthRouter';
import AdminRouter from './routes/AdminRouter';
import { useStoreActions } from './store/hooks';
import ProgressBar from './components/elements/ProgressBar';

function App() {
    const initializeAuth = useStoreActions((actions: any) => actions.auth.initializeAuth);

    useEffect(() => {
        initializeAuth();
    }, [initializeAuth]);

    return (
        <BrowserRouter>
            <ProgressBar />
            <Routes>
                <Route path='/auth/*' element={<AuthRouter />} />

                <Route path='/admin/*' element={<AdminRouter />} />

                <Route path='/album/:identifier/*' element={<AlbumRouter />} />
                <Route path='/*' element={<IndexRouter />} />
            </Routes>
        </BrowserRouter>
    );
}

export default App;
