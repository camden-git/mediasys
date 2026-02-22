import React from 'react';
import { getPreviewImageUrl } from '../../../../api';

interface FaceThumbnailProps {
    imagePath: string;
    /** Pixel coordinates in original image space */
    x1: number;
    y1: number;
    x2: number;
    y2: number;
    /** Original image dimensions (from backend) for correct normalisation */
    imageWidth?: number;
    imageHeight?: number;
    onClick?: () => void;
    className?: string;
}

const FaceThumbnail: React.FC<FaceThumbnailProps> = ({
    imagePath,
    x1,
    y1,
    x2,
    y2,
    imageWidth,
    imageHeight,
    onClick,
    className,
}) => {
    let bgSize = '100%';
    let bgPosX = '50%';
    let bgPosY = '50%';

    if (imageWidth && imageHeight && imageWidth > 0 && imageHeight > 0) {
        // Normalise pixel coords to 0-1 fractions
        const nx1 = x1 / imageWidth;
        const ny1 = y1 / imageHeight;
        const nx2 = x2 / imageWidth;
        const ny2 = y2 / imageHeight;

        const fW = nx2 - nx1;
        const fH = ny2 - ny1;
        const cx = (nx1 + nx2) / 2;
        const cy = (ny1 + ny2) / 2;

        // viewFrac: fraction of the image the viewport covers (20% padding around face)
        const viewFrac = Math.min(1, Math.max(fW, fH) * 1.2);

        if (viewFrac < 1 && 1 - viewFrac > 0.0001) {
            bgSize = `${(1 / viewFrac) * 100}%`;
            bgPosX = `${Math.max(0, Math.min(100, ((cx - viewFrac / 2) / (1 - viewFrac)) * 100))}%`;
            bgPosY = `${Math.max(0, Math.min(100, ((cy - viewFrac / 2) / (1 - viewFrac)) * 100))}%`;
        }
    }

    return (
        <div
            className={`aspect-square w-full cursor-pointer bg-zinc-200 dark:bg-zinc-800 ${className ?? ''}`}
            style={{
                backgroundImage: `url(${getPreviewImageUrl(imagePath)})`,
                backgroundSize: bgSize,
                backgroundPosition: `${bgPosX} ${bgPosY}`,
                backgroundRepeat: 'no-repeat',
            }}
            onClick={onClick}
        />
    );
};

export default FaceThumbnail;
