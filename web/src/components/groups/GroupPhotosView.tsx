import React, { useMemo, useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { SparklesIcon, PhotoIcon, ArrowLeftIcon } from '@heroicons/react/16/solid';
import { useGroup, useGroupPhotos } from '../../hooks/useGroups.ts';
import { useDocumentTitle } from '../../hooks/useDocumentTitle.ts';
import { usePaginatedLightbox } from '../../hooks/usePaginatedLightbox.ts';
import PhotoPageLayout from '../album/PhotoPageLayout.tsx';

const GroupPhotosView: React.FC = () => {
    const { slug } = useParams<{ slug: string }>();
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const [highlightsMode, setHighlightsMode] = useState(searchParams.get('highlights') === '1');
    const minRating = highlightsMode ? 4 : undefined;

    const { group } = useGroup(slug);
    const { data, isLoading, error, hasNextPage, isFetchingNextPage, fetchNextPage } = useGroupPhotos(slug, minRating);

    useDocumentTitle(group ? `${group.name} photos` : null);

    const files = useMemo(() => data?.pages.flatMap((p) => p.files ?? []) ?? [], [data]);
    const total = data?.pages[data.pages.length - 1]?.total ?? 0;
    const imageFiles = useMemo(() => files.filter((f) => !f.is_dir && f.thumbnail_path), [files]);

    const lightbox = usePaginatedLightbox({
        images: imageFiles,
        hasMore: !!hasNextPage,
        isFetchingMore: isFetchingNextPage,
        fetchMore: fetchNextPage,
        resetKey: `${slug}:${minRating ?? ''}`,
    });

    const handleToggleHighlights = () => {
        const next = !highlightsMode;
        setHighlightsMode(next);
        navigate(`/groups/${slug}/photos${next ? '?highlights=1' : ''}`, { replace: true });
    };

    const metadata = (
        <>
            <div className='flex items-center gap-1.5'>
                <PhotoIcon className='size-4 text-gray-950/40 dark:text-white/40' />
                {total} photo{total !== 1 ? 's' : ''}
            </div>
            {highlightsMode && <span className='text-yellow-500'>(4★+)</span>}
        </>
    );

    const actions = (
        <>
            <Link
                to={`/groups/${slug}`}
                className='inline-flex items-center gap-1.5 rounded-full bg-gray-950/10 px-3 py-1 text-sm font-semibold text-gray-950 hover:bg-gray-950/20 dark:bg-white/10 dark:text-white dark:hover:bg-white/20'
            >
                <ArrowLeftIcon className='size-3.5' />
                Back to group
            </Link>
            <button
                type='button'
                onClick={handleToggleHighlights}
                aria-pressed={highlightsMode}
                className={`inline-flex items-center gap-x-1.5 rounded-full px-3 py-1 text-sm font-semibold transition-colors ${
                    highlightsMode
                        ? 'bg-yellow-400 text-gray-950 hover:bg-yellow-300'
                        : 'bg-gray-950/10 text-gray-950 hover:bg-gray-950/20 dark:bg-white/10 dark:text-white dark:hover:bg-white/20'
                }`}
            >
                <SparklesIcon className='size-3.5' />
                {highlightsMode ? 'Show all' : 'Highlights'}
            </button>
        </>
    );

    return (
        <PhotoPageLayout
            title={group?.name ?? 'Group photos'}
            metadata={metadata}
            actions={actions}
            images={imageFiles}
            isLoading={isLoading}
            error={error ? error.message || 'Failed to load photos' : null}
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
            emptyMessage={highlightsMode ? 'No highlighted photos in this group.' : 'No photos in this group.'}
        />
    );
};

export default GroupPhotosView;
