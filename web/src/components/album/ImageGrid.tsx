import React from 'react';
import { FileInfo } from '../../types.ts';
import ImageGridItem from './ImageGridItem.tsx';

interface ImageGridProps {
    images: FileInfo[];
}

const ImageGrid: React.FC<ImageGridProps> = ({ images }) => {
    if (!images || images.length === 0) {
        return <p className='my-8 text-center text-gray-500'>No images found in this album.</p>;
    }

    return (
        <div className='grid grid-cols-3 gap-1 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-7 xl:grid-cols-9'>
            {images.map((image) => (image.thumbnail_path ? <ImageGridItem key={image.path} image={image} /> : null))}
        </div>
    );
};

export default ImageGrid;
