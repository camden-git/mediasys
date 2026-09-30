import React from 'react';
import { getFaceThumbnailUrl } from '../../../../api';

interface FaceThumbnailProps {
    faceId: number;
    onClick?: () => void;
    className?: string;
    label?: string;
}

const FaceThumbnail: React.FC<FaceThumbnailProps> = ({ faceId, onClick, className, label = 'Face' }) => {
    const image = (
        <img
            src={getFaceThumbnailUrl(faceId)}
            alt=''
            loading='lazy'
            className='h-full w-full object-cover'
            draggable={false}
        />
    );
    const base = `block aspect-square w-full bg-zinc-200 dark:bg-zinc-800 ${className ?? ''}`;

    if (!onClick) return <div className={base}>{image}</div>;

    return (
        <button type='button' aria-label={label} className={`${base} cursor-pointer`} onClick={onClick}>
            {image}
        </button>
    );
};

export default FaceThumbnail;
