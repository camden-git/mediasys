import React from 'react';
import { FileInfo } from '../../types.ts';
import { getThumbnailUrl } from '../../api.ts';

interface ImageGridItemProps {
    image: FileInfo;
}

const ImageGridItem: React.FC<ImageGridItemProps> = React.memo(({ image }) => {
    if (!image.thumbnail_path) {
        return null;
    }

    const thumbnailUrl = getThumbnailUrl(image.thumbnail_path);

    return (
        <div className='relative aspect-square overflow-hidden bg-gray-200'>
            <img
                src={thumbnailUrl}
                alt={image.name}
                className='absolute inset-0 h-full w-full object-cover transition-transform duration-300 ease-in-out'
                loading='lazy'
                onError={(e) => {
                    (e.target as HTMLImageElement).style.display = 'none';
                    console.error(`Failed to load thumbnail: ${thumbnailUrl}`);
                }}
            />
        </div>
    );
});

export default ImageGridItem;
