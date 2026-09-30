import React from 'react';
import AlbumList from '../components/index/AlbumList.tsx';

// Only matches "/" exactly (see App.tsx); unknown paths fall through to the top-level NotFound route.
const IndexRouter: React.FC = () => <AlbumList />;

export default IndexRouter;
