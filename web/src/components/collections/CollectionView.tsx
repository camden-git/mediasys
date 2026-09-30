import React, { useMemo } from 'react';
import { useParams } from 'react-router-dom';
import { getBannerUrl } from '../../api/media';
import { PhotoIcon } from '@heroicons/react/16/solid';
import { useCollection, useCollectionPhotos } from '../../hooks/useCollections.ts';
import { useDocumentTitle } from '../../hooks/useDocumentTitle.ts';
import { usePaginatedLightbox } from '../../hooks/usePaginatedLightbox.ts';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import PhotoPageLayout from '../album/PhotoPageLayout.tsx';

const CollectionView: React.FC = () => {
    const { slug } = useParams<{ slug: string }>();
    const { collection, isLoading: collectionLoading, error: collectionError } = useCollection(slug);
    const {
        data,
        isLoading: photosLoading,
        error: photosError,
        hasNextPage,
        isFetchingNextPage,
        fetchNextPage,
    } = useCollectionPhotos(slug);

    useDocumentTitle(collection?.name);

    const files = useMemo(() => data?.pages.flatMap((p) => p.files) ?? [], [data]);
    const total = data?.pages[data.pages.length - 1]?.total ?? 0;
    const imageFiles = useMemo(() => files.filter((f) => !f.is_dir && f.thumbnail_path), [files]);

    const lightbox = usePaginatedLightbox({
        images: imageFiles,
        hasMore: !!hasNextPage,
        isFetchingMore: isFetchingNextPage,
        fetchMore: fetchNextPage,
        resetKey: slug,
    });

    if (collectionLoading) return <LoadingSpinner />;
    if (collectionError)
        return <p className='p-8 text-red-500'>{collectionError.message || 'Failed to load collection'}</p>;
    if (!collection) return null;

    const collectionMetadata = (
        <>
            <div className='flex items-center gap-1.5'>
                <PhotoIcon className='size-4 text-gray-950/40 dark:text-white/40' />
                {total} photo{total !== 1 ? 's' : ''}
            </div>
            {collection.filters && collection.filters.length > 0 && (
                <>
                    <span className='hidden text-gray-950/25 sm:inline dark:text-white/25'>&middot;</span>
                    {collection.filters.map((f) => (
                        <span
                            key={f.id}
                            className='rounded-full bg-gray-950/10 px-3 py-0.5 font-mono text-xs font-normal text-gray-700 dark:bg-white/10 dark:text-gray-300'
                        >
                            {f.tag_key} = {f.tag_value}
                        </span>
                    ))}
                </>
            )}
        </>
    );

    return (
        <PhotoPageLayout
            title={collection.name}
            description={collection.description}
            bannerUrls={collection.banners?.map(getBannerUrl)}
            metadata={collectionMetadata}
            images={imageFiles}
            isLoading={photosLoading}
            error={photosError ? photosError.message || 'Failed to load photos' : null}
            hasMore={!!hasNextPage}
            sentinelRef={lightbox.sentinelRef}
            selectedImage={lightbox.selectedImage}
            selectedIndex={lightbox.selectedIndex}
            totalImages={total}
            onImageClick={lightbox.onImageClick}
            onClose={lightbox.onClose}
            onPrev={lightbox.onPrev}
            onNext={lightbox.onNext}
            canPrev={lightbox.canPrev}
            canNext={lightbox.canNext}
            emptyMessage="No photos match this collection's filters."
        />
    );
};

export default CollectionView;
