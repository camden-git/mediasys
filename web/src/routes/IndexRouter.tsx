import React from 'react';
import { Routes, Route } from 'react-router-dom';
import NotFound from '../components/NotFound.tsx';
import AlbumList from '../components/index/AlbumList.tsx';

const IndexRouter: React.FC = () => {
    return (
        <Routes>
            <Route index element={<AlbumList />} />
            <Route path='*' element={<NotFound />} />
        </Routes>
    );
};

export default IndexRouter;
