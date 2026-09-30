import React, { Suspense, useEffect } from 'react';
import { Routes, Route, RouterProvider, createBrowserRouter } from 'react-router-dom';
import { useAuthStore } from './store/useAuthStore';
import ProgressBar from './components/elements/ProgressBar';
import { UnsavedChangesGuard } from './components/elements/UnsavedChangesGuard';

const IndexRouter = React.lazy(() => import('./routes/IndexRouter'));
const AlbumRouter = React.lazy(() => import('./routes/AlbumRouter'));
const AuthRouter = React.lazy(() => import('./routes/AuthRouter'));
const AdminRouter = React.lazy(() => import('./routes/AdminRouter'));
const AdminAlbumRouter = React.lazy(() => import('./routes/AdminAlbumRouter'));
const GroupsPage = React.lazy(() => import('./components/groups/GroupsPage'));
const GroupView = React.lazy(() => import('./components/groups/GroupView'));
const GroupPhotosView = React.lazy(() => import('./components/groups/GroupPhotosView'));
const PeoplePage = React.lazy(() => import('./components/people/PeoplePage'));
const PersonView = React.lazy(() => import('./components/people/PersonView'));
const NotFound = React.lazy(() => import('./components/NotFound'));
const CollectionRouter = React.lazy(() => import('./routes/CollectionRouter'));

function AppRoutes() {
    return (
        <>
            <ProgressBar />
            <UnsavedChangesGuard />
            <Suspense fallback={null}>
                <Routes>
                    <Route path='/auth/*' element={<AuthRouter />} />

                    <Route path='/admin/albums/view/:slug/*' element={<AdminAlbumRouter />} />
                    <Route path='/admin/*' element={<AdminRouter />} />

                    <Route path='/groups' element={<GroupsPage />} />
                    <Route path='/groups/:slug' element={<GroupView />} />
                    <Route path='/groups/:slug/photos' element={<GroupPhotosView />} />

                    <Route path='/collections/*' element={<CollectionRouter />} />

                    <Route path='/people' element={<PeoplePage />} />
                    <Route path='/people/:personId' element={<PersonView />} />

                    <Route path='/album/:identifier/*' element={<AlbumRouter />} />
                    <Route path='/' element={<IndexRouter />} />
                    <Route path='*' element={<NotFound />} />
                </Routes>
            </Suspense>
        </>
    );
}

// A data router (rather than <BrowserRouter>) is required for useBlocker (see UnsavedChangesGuard).
const router = createBrowserRouter([{ path: '*', element: <AppRoutes /> }]);

function App() {
    const initializeAuth = useAuthStore((s) => s.initializeAuth);

    useEffect(() => {
        initializeAuth();
    }, [initializeAuth]);

    return <RouterProvider router={router} />;
}

export default App;
