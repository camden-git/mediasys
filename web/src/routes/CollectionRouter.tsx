import React from 'react';
import { Routes, Route } from 'react-router-dom';
import CollectionsPage from '../components/collections/CollectionsPage.tsx';
import CollectionView from '../components/collections/CollectionView.tsx';
import NotFound from '../components/NotFound.tsx';

// Rendered inside PublicLayout (see App.tsx), which provides the top nav.
const CollectionRouter: React.FC = () => (
    <Routes>
        <Route index element={<CollectionsPage />} />
        <Route path=':slug' element={<CollectionView />} />
        <Route path='*' element={<NotFound />} />
    </Routes>
);

export default CollectionRouter;
