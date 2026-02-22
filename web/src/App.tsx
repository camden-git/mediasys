import { useEffect } from 'react';
import { Routes, Route, BrowserRouter } from 'react-router-dom';
import IndexRouter from './routes/IndexRouter';
import AlbumRouter from './routes/AlbumRouter';
import AuthRouter from './routes/AuthRouter';
import AdminRouter from './routes/AdminRouter';
import AdminAlbumRouter from './routes/AdminAlbumRouter';
import { useStoreActions } from './store/hooks';
import ProgressBar from './components/elements/ProgressBar';
import GroupsPage from './components/groups/GroupsPage';
import GroupView from './components/groups/GroupView';
import GroupPhotosView from './components/groups/GroupPhotosView';
import PeoplePage from './components/people/PeoplePage';
import PersonView from './components/people/PersonView';
import CollectionsPage from './components/collections/CollectionsPage';
import CollectionView from './components/collections/CollectionView';
import CollectionPhotosView from './components/collections/CollectionPhotosView';

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

                <Route path='/admin/albums/view/:id/*' element={<AdminAlbumRouter />} />
                <Route path='/admin/*' element={<AdminRouter />} />

                <Route path='/groups' element={<GroupsPage />} />
                <Route path='/groups/:slug' element={<GroupView />} />
                <Route path='/groups/:slug/photos' element={<GroupPhotosView />} />

                <Route path='/collections' element={<CollectionsPage />} />
                <Route path='/collections/:slug' element={<CollectionView />} />
                <Route path='/collections/:slug/photos' element={<CollectionPhotosView />} />

                <Route path='/people' element={<PeoplePage />} />
                <Route path='/people/:personId' element={<PersonView />} />

                <Route path='/album/:identifier/*' element={<AlbumRouter />} />
                <Route path='/*' element={<IndexRouter />} />
            </Routes>
        </BrowserRouter>
    );
}

export default App;
