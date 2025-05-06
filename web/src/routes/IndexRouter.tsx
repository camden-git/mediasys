import React from 'react';
import { Routes, Route } from 'react-router-dom';
import AlbumList from '../components/index/AlbumList.tsx';

const IndexRouter: React.FC = () => {
    return (
        <Routes>
            <Route index element={<AlbumList />} />
        </Routes>
    );
};

export default IndexRouter;
