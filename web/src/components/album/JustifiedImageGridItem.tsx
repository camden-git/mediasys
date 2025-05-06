import React from 'react';
import { FileInfo } from '../../types.ts';
import { getThumbnailUrl } from '../../api.ts';

interface JustifiedImageGridItemProps {
    image: FileInfo;
    height: number;
    width: number;
    margin: number;
    onClick: () => void;
}

const JustifiedImageGridItem: React.FC<JustifiedImageGridItemProps> = React.memo(
    ({ image, height, width, margin, onClick }) => {
        if (!image.thumbnail_path) {
            return null;
        }
        const thumbnailUrl = getThumbnailUrl(image.thumbnail_path);

        const style: React.CSSProperties = {
            width: `${width}px`,
            height: `${height}px`,
            marginRight: `${margin}px`,
            display: 'inline-block',
            verticalAlign: 'top',
            position: 'relative',
            overflow: 'hidden',
            backgroundColor: '#eee',
            cursor: 'pointer',
        };

        return (
            <div style={style} onClick={onClick}>
                <img
                    src={thumbnailUrl}
                    alt={image.name}
                    className='absolute top-0 left-0 h-full w-full object-cover'
                    loading='lazy'
                    style={{
                        width: '100%',
                        height: '100%',
                    }}
                    onError={(e) => {
                        (e.target as HTMLImageElement).style.display = 'none';
                        console.error(`Failed to load thumbnail: ${thumbnailUrl}`);
                    }}
                />
            </div>
        );
    },
);

export default JustifiedImageGridItem;
