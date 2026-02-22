import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { FileInfo } from '../../types.ts';
import { getGroupPhotos } from '../../api.ts';
import AdvancedImageGrid from '../album/AdvancedImageGrid.tsx';
import ImageLightbox from '../album/ImageLightbox.tsx';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import { SparklesIcon, PhotoIcon, ArrowLeftIcon } from '@heroicons/react/16/solid';

const PAGE_SIZE = 120;

const GroupPhotosView: React.FC = () => {
    const { slug } = useParams<{ slug: string }>();
    const navigate = useNavigate();

    const searchParams = new URLSearchParams(typeof window !== 'undefined' ? window.location.search : '');
    const highlightsParam = searchParams.get('highlights') === '1';

    const [highlightsMode, setHighlightsMode] = useState(highlightsParam);
    const [files, setFiles] = useState<FileInfo[]>([]);
    const [total, setTotal] = useState(0);
    const [hasMore, setHasMore] = useState(false);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [selectedImage, setSelectedImage] = useState<FileInfo | null>(null);

    const isFetchingRef = useRef(false);
    const sentinelRef = useRef<HTMLDivElement | null>(null);

    const minRating = highlightsMode ? 4 : undefined;

    const fetchInitial = useCallback(async (slug: string, minRating?: number) => {
        setFiles([]);
        setTotal(0);
        setHasMore(false);
        setIsLoading(true);
        setError(null);
        try {
            const data = await getGroupPhotos(slug, { offset: 0, limit: PAGE_SIZE, min_rating: minRating });
            setFiles(data.files ?? []);
            setTotal(data.total ?? 0);
            setHasMore(data.has_more ?? false);
        } catch (err: any) {
            if (err.name !== 'AbortError') setError(err.message || 'Failed to load photos');
        } finally {
            setIsLoading(false);
        }
    }, []);

    const loadMore = useCallback(async () => {
        if (!slug || isFetchingRef.current || !hasMore) return;
        isFetchingRef.current = true;
        try {
            const data = await getGroupPhotos(slug, { offset: files.length, limit: PAGE_SIZE, min_rating: minRating });
            setFiles((prev) => [...prev, ...(data.files ?? [])]);
            setHasMore(data.has_more ?? false);
            setTotal(data.total ?? 0);
        } catch {
            // ignore
        } finally {
            isFetchingRef.current = false;
        }
    }, [slug, files.length, hasMore, minRating]);

    useEffect(() => {
        if (slug) fetchInitial(slug, minRating);
    }, [slug, minRating, fetchInitial]);

    useEffect(() => {
        if (!sentinelRef.current) return;
        const el = sentinelRef.current;
        const observer = new IntersectionObserver(
            (entries) => {
                if (entries[0].isIntersecting) void loadMore();
            },
            { rootMargin: '0px 0px 300px 0px', threshold: 0.01 },
        );
        observer.observe(el);
        return () => observer.disconnect();
    }, [loadMore]);

    const imageFiles = useMemo(() => files.filter((f) => !f.is_dir && f.thumbnail_path), [files]);

    const selectedIndex = useMemo(() => {
        if (!selectedImage) return -1;
        return imageFiles.findIndex((f) => f.path === selectedImage.path);
    }, [selectedImage, imageFiles]);

    const handleToggleHighlights = () => {
        const next = !highlightsMode;
        setHighlightsMode(next);
        navigate(`/groups/${slug}/photos${next ? '?highlights=1' : ''}`, { replace: true });
    };

    return (
        <div className='mx-auto max-w-6xl px-4 py-8'>
            {/* Header */}
            <div className='mb-6 flex flex-wrap items-center gap-4'>
                <Link
                    to={`/groups/${slug}`}
                    className='inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-white'
                >
                    <ArrowLeftIcon className='size-3.5' />
                    Back to group
                </Link>
                <div className='flex-1' />
                <div className='flex gap-2'>
                    <button
                        onClick={handleToggleHighlights}
                        className={`inline-flex items-center gap-x-1.5 rounded-full px-3 py-1 text-sm font-semibold transition-colors ${
                            highlightsMode
                                ? 'bg-yellow-400 text-gray-950 hover:bg-yellow-300'
                                : 'bg-gray-950/10 text-gray-950 hover:bg-gray-950/20 dark:bg-white/10 dark:text-white dark:hover:bg-white/20'
                        }`}
                    >
                        <SparklesIcon className='size-3.5' />
                        {highlightsMode ? 'Show all' : 'Highlights'}
                    </button>
                </div>
            </div>

            <div className='mb-4 flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400'>
                <PhotoIcon className='size-4' />
                <span>
                    {total} photo{total !== 1 ? 's' : ''}
                </span>
                {highlightsMode && <span className='text-yellow-500'>(4★+)</span>}
            </div>

            {error && <p className='mb-4 text-red-500'>{error}</p>}

            {!isLoading && imageFiles.length === 0 && !error && (
                <p className='text-gray-500 dark:text-gray-400'>
                    {highlightsMode ? 'No highlighted photos in this group.' : 'No photos in this group.'}
                </p>
            )}

            {imageFiles.length > 0 && (
                <AdvancedImageGrid
                    images={imageFiles}
                    targetRowHeight={280}
                    boxSpacing={4}
                    onImageClick={setSelectedImage}
                />
            )}

            {isLoading && <LoadingSpinner />}
            {hasMore && !isLoading && <div ref={sentinelRef} className='h-1 w-full' />}

            <ImageLightbox
                image={selectedImage}
                imageIndex={selectedIndex >= 0 ? selectedIndex : undefined}
                totalImages={total}
                onClose={() => setSelectedImage(null)}
                onPrev={() => {
                    if (selectedIndex > 0) setSelectedImage(imageFiles[selectedIndex - 1]);
                }}
                onNext={() => {
                    if (selectedIndex < imageFiles.length - 1) setSelectedImage(imageFiles[selectedIndex + 1]);
                }}
                canPrev={selectedIndex > 0}
                canNext={selectedIndex >= 0 && selectedIndex < imageFiles.length - 1}
            />
        </div>
    );
};

export default GroupPhotosView;
