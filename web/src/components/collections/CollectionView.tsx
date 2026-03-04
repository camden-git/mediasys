import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { FileInfo } from '../../types.ts';
import { getCollectionPhotos } from '../../api/collections.ts';
import { getBannerUrl } from '../../api.ts';
import { PhotoIcon } from '@heroicons/react/16/solid';
import { useCollection } from '../../hooks/useCollections.ts';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import PhotoPageLayout from '../album/PhotoPageLayout.tsx';

const PAGE_SIZE = 120;

const CollectionView: React.FC = () => {
    const { slug } = useParams<{ slug: string }>();
    const { collection, isLoading: collectionLoading, error: collectionError } = useCollection(slug);

    const [files, setFiles] = useState<FileInfo[]>([]);
    const [total, setTotal] = useState(0);
    const [hasMore, setHasMore] = useState(false);
    const [photosLoading, setPhotosLoading] = useState(true);
    const [photosError, setPhotosError] = useState<string | null>(null);
    const [selectedImage, setSelectedImage] = useState<FileInfo | null>(null);

    const isFetchingRef = useRef(false);
    const hasUserScrolledRef = useRef(false);
    const prefillCountRef = useRef(0);
    const sentinelRef = useRef<HTMLDivElement | null>(null);

    const fetchInitial = useCallback(async (s: string) => {
        setFiles([]);
        setTotal(0);
        setHasMore(false);
        setPhotosLoading(true);
        setPhotosError(null);
        prefillCountRef.current = 0;
        hasUserScrolledRef.current = false;
        try {
            const data = await getCollectionPhotos(s, 0, PAGE_SIZE);
            setFiles(data.files ?? []);
            setTotal(data.total ?? 0);
            setHasMore(data.has_more ?? false);
        } catch (err: any) {
            if (err.name !== 'AbortError') setPhotosError(err.message || 'Failed to load photos');
        } finally {
            setPhotosLoading(false);
        }
    }, []);

    const loadMore = useCallback(async () => {
        if (!slug || isFetchingRef.current || !hasMore) return;
        isFetchingRef.current = true;
        try {
            const data = await getCollectionPhotos(slug, files.length, PAGE_SIZE);
            setFiles((prev) => [...prev, ...(data.files ?? [])]);
            setHasMore(data.has_more ?? false);
            setTotal(data.total ?? 0);
        } catch {
            // ignore
        } finally {
            isFetchingRef.current = false;
        }
    }, [slug, files.length, hasMore]);

    useEffect(() => {
        if (slug) fetchInitial(slug);
    }, [slug, fetchInitial]);

    useEffect(() => {
        const onScroll = () => {
            if (window.scrollY > 0) hasUserScrolledRef.current = true;
        };
        window.addEventListener('scroll', onScroll, { passive: true });
        return () => window.removeEventListener('scroll', onScroll);
    }, []);

    useEffect(() => {
        if (!sentinelRef.current) return;
        const el = sentinelRef.current;
        const observer = new IntersectionObserver(
            (entries) => {
                if (entries[0].isIntersecting && hasUserScrolledRef.current) void loadMore();
            },
            { rootMargin: '0px 0px 300px 0px', threshold: 0.01 },
        );
        observer.observe(el);
        return () => observer.disconnect();
    }, [loadMore]);

    // Prefill: load once without scroll if content doesn't fill viewport
    useEffect(() => {
        if (!hasMore) return;
        const viewportH = window.innerHeight || 0;
        const contentH = document.documentElement.scrollHeight || 0;
        if (contentH <= viewportH + 40 && prefillCountRef.current < 1) {
            prefillCountRef.current += 1;
            void loadMore();
        }
    }, [files.length, hasMore, loadMore]);

    const imageFiles = useMemo(() => files.filter((f) => !f.is_dir && f.thumbnail_path), [files]);

    const selectedIndex = useMemo(() => {
        if (!selectedImage) return -1;
        return imageFiles.findIndex((f) => f.path === selectedImage.path);
    }, [selectedImage, imageFiles]);

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
            error={photosError}
            hasMore={hasMore}
            sentinelRef={sentinelRef}
            selectedImage={selectedImage}
            selectedIndex={selectedIndex}
            totalImages={total}
            onImageClick={setSelectedImage}
            onClose={() => setSelectedImage(null)}
            onPrev={() => {
                if (selectedIndex > 0) setSelectedImage(imageFiles[selectedIndex - 1]);
            }}
            onNext={() => {
                if (selectedIndex < imageFiles.length - 1) setSelectedImage(imageFiles[selectedIndex + 1]);
            }}
            canPrev={selectedIndex > 0}
            canNext={selectedIndex >= 0 && selectedIndex < imageFiles.length - 1}
            emptyMessage="No photos match this collection's filters."
        />
    );
};

export default CollectionView;
