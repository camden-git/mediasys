import React, { RefObject, useEffect, useState } from 'react';
import { FileInfo } from '../../types.ts';
import { Heading } from '../elements/Heading.tsx';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import ErrorMessage from '../elements/ErrorMessage.tsx';
import AdvancedImageGrid from './AdvancedImageGrid.tsx';
import ImageLightbox from './ImageLightbox.tsx';

interface PhotoPageLayoutProps {
    title: string;
    description?: string;
    bannerUrls?: string[];
    metadata?: React.ReactNode;
    actions?: React.ReactNode;
    images: FileInfo[];
    isLoading: boolean;
    error?: string | null;
    hasMore: boolean;
    sentinelRef: RefObject<HTMLDivElement | null>;
    selectedImage: FileInfo | null;
    selectedIndex: number;
    totalImages: number;
    onImageClick: (img: FileInfo) => void;
    onClose: () => void;
    onPrev: () => void;
    onNext: () => void;
    canPrev: boolean;
    canNext: boolean;
    emptyMessage?: string;
}

const PhotoPageLayout: React.FC<PhotoPageLayoutProps> = ({
    title,
    description,
    bannerUrls,
    metadata,
    actions,
    images,
    isLoading,
    error,
    hasMore,
    sentinelRef,
    selectedImage,
    selectedIndex,
    totalImages,
    onImageClick,
    onClose,
    onPrev,
    onNext,
    canPrev,
    canNext,
    emptyMessage = 'No photos here yet.',
}) => {
    const [activeIdx, setActiveIdx] = useState(0);

    useEffect(() => {
        if (!bannerUrls || bannerUrls.length <= 1) return;
        const t = setInterval(() => setActiveIdx((i) => (i + 1) % bannerUrls.length), 7000);
        return () => clearInterval(t);
    }, [bannerUrls]);

    // reset index when banner set changes (e.g. switching albums)
    useEffect(() => {
        setActiveIdx(0);
    }, [bannerUrls]);

    return (
        <div className='relative isolate mx-auto'>
            <div className='absolute inset-x-0 top-0 -z-10 h-80 overflow-hidden rounded-t-2xl mask-b-from-60% sm:h-88 md:h-112 lg:h-128'>
                {bannerUrls && bannerUrls.length > 0 && (
                    <>
                        {bannerUrls.map((url, i) => (
                            <img
                                key={url}
                                alt=''
                                src={url}
                                loading='lazy'
                                decoding='async'
                                className='absolute inset-0 h-full w-full mask-l-from-60% object-cover object-center'
                                style={{
                                    opacity: i === activeIdx ? 0.4 : 0,
                                    transition: bannerUrls.length > 1 ? 'opacity 500ms ease-in-out' : 'none',
                                }}
                            />
                        ))}
                    </>
                )}
                <div className='absolute inset-0 rounded-t-2xl outline-1 -outline-offset-1 outline-gray-950/10 dark:outline-white/10' />
            </div>

            <div className='mx-auto'>
                <div className='relative'>
                    <div className='px-8 pt-48 pb-12 lg:py-24'>
                        <h1 className='sr-only'>{title}</h1>
                        <Heading className='truncate font-bold' huge>
                            {title}
                        </Heading>
                        {description && (
                            <p className='mt-7 max-w-lg text-base/7 text-pretty text-gray-600 dark:text-gray-400'>
                                {description}
                            </p>
                        )}
                        {metadata && (
                            <div className='mt-6 flex flex-wrap items-center gap-x-4 gap-y-3 text-sm/7 font-semibold text-gray-950 sm:gap-3'>
                                {metadata}
                            </div>
                        )}
                        {actions && <div className='mt-10 flex flex-wrap gap-3'>{actions}</div>}
                    </div>

                    <div className='mx-2 mt-4'>
                        <ErrorMessage message={error ?? null} />

                        {isLoading && images.length === 0 && <LoadingSpinner />}

                        {!isLoading && !error && images.length === 0 && (
                            <p className='px-8 text-gray-500 dark:text-gray-400'>{emptyMessage}</p>
                        )}

                        {images.length > 0 && (
                            <AdvancedImageGrid
                                images={images}
                                targetRowHeight={280}
                                boxSpacing={4}
                                onImageClick={onImageClick}
                            />
                        )}

                        {hasMore && images.length > 0 && <div ref={sentinelRef} className='h-1 w-full' />}

                        <ImageLightbox
                            image={selectedImage}
                            imageIndex={selectedIndex >= 0 ? selectedIndex : undefined}
                            totalImages={totalImages}
                            onClose={onClose}
                            onPrev={onPrev}
                            onNext={onNext}
                            canPrev={canPrev}
                            canNext={canNext}
                        />
                    </div>
                </div>
            </div>
        </div>
    );
};

export default PhotoPageLayout;
