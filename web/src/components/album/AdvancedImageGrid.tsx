import React, { useState, useEffect } from 'react';
import { FileInfo } from '../../types.ts';
import useResizeObserver from '../../hooks/useResizeObserver.ts';
import { computeLayout, ProcessedRow, LayoutOptions } from '../../lib/galleryLayout.ts';
import JustifiedImageGridItem from './JustifiedImageGridItem.tsx';

interface AdvancedImageGridProps {
    images: FileInfo[];
    targetRowHeight?: number;
    boxSpacing?: number;
    stretchLastRow?: boolean;
    maxRowHeightRatio?: number | null;
    debounceDelay?: number;
    onImageClick: (image: FileInfo) => void;
}

const AdvancedImageGrid: React.FC<AdvancedImageGridProps> = ({
    images,
    targetRowHeight = 180,
    boxSpacing = 5,
    stretchLastRow = false,
    maxRowHeightRatio = null,
    debounceDelay = 250,
    onImageClick,
}) => {
    const [, containerSize, gridRef] = useResizeObserver<HTMLDivElement>();
    const [processedLayout, setProcessedLayout] = useState<ProcessedRow[]>([]);

    // debounce the measured width itself; first non-zero width is applied immediately
    const [width, setWidth] = useState(0);
    useEffect(() => {
        if (containerSize.width === 0 || width === 0) {
            setWidth(containerSize.width);
            return;
        }
        const timer = setTimeout(() => setWidth(containerSize.width), debounceDelay);
        return () => clearTimeout(timer);
    }, [containerSize.width, debounceDelay, width]);

    useEffect(() => {
        if (width > 0 && images.length > 0) {
            setProcessedLayout(
                computeLayout(images, {
                    containerWidth: width,
                    targetRowHeight,
                    boxSpacing,
                    stretchLastRow,
                    maxRowHeightRatio,
                } satisfies LayoutOptions),
            );
        } else {
            setProcessedLayout([]);
        }
    }, [images, width, targetRowHeight, boxSpacing, stretchLastRow, maxRowHeightRatio]);

    return (
        <div ref={gridRef} className='advanced-image-grid w-full'>
            {processedLayout.map((row) => (
                <div
                    key={`row-${row.rowIndex}`}
                    data-row-index={row.rowIndex}
                    data-row-height={row.height.toFixed(1)}
                    className='gallery-row whitespace-nowrap'
                    style={{ marginBottom: `${boxSpacing}px`, height: `${row.height}px` }}
                >
                    {row.items.map((image, itemIndex) => {
                        const aspectRatio = image.width && image.height ? image.width / image.height : 1.5;
                        const calculatedWidth = aspectRatio * (row.height + Number.EPSILON);
                        const margin = itemIndex === row.items.length - 1 ? 0 : boxSpacing;
                        return (
                            <JustifiedImageGridItem
                                key={image.path}
                                image={image}
                                height={row.height}
                                width={calculatedWidth}
                                margin={margin}
                                onImageClick={onImageClick}
                            />
                        );
                    })}
                </div>
            ))}
            {processedLayout.length === 0 && (
                <div className='mx-auto flex justify-center'>
                    <svg
                        className='mr-3 h-6 w-6 animate-spin text-white'
                        xmlns='http://www.w3.org/2000/svg'
                        fill='none'
                        viewBox='0 0 24 24'
                    >
                        <circle
                            className='text-gray-400'
                            cx='12'
                            cy='12'
                            r='10'
                            stroke='currentColor'
                            strokeWidth='4'
                        ></circle>
                        <path
                            className='opacity-50'
                            fill='currentColor'
                            d='M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z'
                        ></path>
                    </svg>
                    <p className='my-auto font-light text-gray-600'>Calculating layout...</p>
                </div>
            )}
        </div>
    );
};

export default AdvancedImageGrid;
