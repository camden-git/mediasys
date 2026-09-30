import React, { useEffect, useRef, useState } from 'react';
import { Heading } from '../../elements/Heading';
import { Text } from '../../elements/Text';
import PageContentBlock from '../../elements/PageContentBlock.tsx';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { UntaggedFaceResult } from '../../../types';
import { getUntaggedFaces, UntaggedFaceParams } from '../../../api';
import FaceThumbnail from './shared/FaceThumbnail';
import FaceLightboxModal from './shared/FaceLightboxModal';

const DEFAULT_FILTERS: UntaggedFaceParams = {
    limit: 100,
    sort_by: 'created_at',
    sort_order: 'desc',
};

// refetch the queue once fewer than this many faces remain
const REFILL_THRESHOLD = 20;

const FaceTaggingContainer: React.FC = () => {
    const [faces, setFaces] = useState<UntaggedFaceResult[]>([]);
    const [loading, setLoading] = useState(true);
    const [fetching, setFetching] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [lightboxIndex, setLightboxIndex] = useState<number | null>(null);

    // Applied filter state
    const [filters, setFilters] = useState<UntaggedFaceParams>(DEFAULT_FILTERS);

    // Pending filter inputs (not yet applied)
    const [pendingMinQuality, setPendingMinQuality] = useState('');
    const [pendingMinConfidence, setPendingMinConfidence] = useState('');
    const [pendingSortBy, setPendingSortBy] = useState<UntaggedFaceParams['sort_by']>('created_at');
    const [pendingSortOrder, setPendingSortOrder] = useState<UntaggedFaceParams['sort_order']>('desc');

    // Group-by-image: the backend keeps only the best face per image (applied with Apply)
    const [pendingGroupByImage, setPendingGroupByImage] = useState(false);

    // true when the last fetch filled the page, i.e. more untagged faces may exist on the server
    const [hasMore, setHasMore] = useState(false);

    const abortRef = useRef<AbortController | null>(null);
    const displayFaces = faces;

    const fetchFaces = (params: UntaggedFaceParams) => {
        if (abortRef.current) abortRef.current.abort();
        const ctrl = new AbortController();
        abortRef.current = ctrl;

        setFetching(true);
        setError(null);
        getUntaggedFaces(params, ctrl.signal)
            .then((f) => {
                setFaces(f);
                setHasMore(f.length >= (params.limit ?? DEFAULT_FILTERS.limit!));
            })
            .catch((e) => {
                if (e.name === 'AbortError') return;
                setError(e.message);
                setHasMore(false);
            })
            .finally(() => {
                if (abortRef.current === ctrl) {
                    setFetching(false);
                    setLoading(false);
                }
            });
    };

    // Initial load
    useEffect(() => {
        fetchFaces(DEFAULT_FILTERS);
        return () => abortRef.current?.abort();
    }, []);

    // The backend has no offset, so the queue is capped at one page. When the reviewer works it
    // down, pull the next batch of untagged faces in automatically.
    useEffect(() => {
        if (loading || fetching || !hasMore || lightboxIndex !== null) return;
        if (faces.length < REFILL_THRESHOLD) fetchFaces(filters);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [faces.length, hasMore, loading, fetching, lightboxIndex]);

    const handleApply = () => {
        const params: UntaggedFaceParams = {
            limit: filters.limit,
            sort_by: pendingSortBy,
            sort_order: pendingSortOrder,
        };
        if (pendingGroupByImage) params.group_by_image = true;
        if (pendingMinQuality !== '') {
            const v = parseFloat(pendingMinQuality);
            if (!isNaN(v)) params.min_quality = v;
        }
        if (pendingMinConfidence !== '') {
            const v = parseFloat(pendingMinConfidence) / 100;
            if (!isNaN(v)) params.min_confidence = v;
        }
        setFilters(params);
        fetchFaces(params);
    };

    const handleTagged = (faceId: number) => {
        setFaces((prev) => {
            const next = prev.filter((f) => f.face_id !== faceId);
            setLightboxIndex((idx) => {
                if (idx === null) return null;
                if (next.length === 0) return null;
                return Math.min(idx, next.length - 1);
            });
            return next;
        });
    };

    const handleDeleted = (faceId: number) => handleTagged(faceId);

    const handleSuggestionUpdated = (
        faceId: number,
        personId: number | null,
        personName: string | null,
        suggestionCount: number,
    ) => {
        setFaces((prev) =>
            prev.map((f) =>
                f.face_id === faceId
                    ? {
                          ...f,
                          suggested_person_id: personId,
                          suggested_person_name: personName,
                          suggestion_count: suggestionCount,
                      }
                    : f,
            ),
        );
    };

    if (loading) return <LoadingSpinner />;

    return (
        <PageContentBlock title='Face Tagging'>
            <div className='mb-6'>
                <Heading>Face Tagging Queue</Heading>
                <Text className='mt-1'>Review untagged faces and assign them to people.</Text>
            </div>

            {/* Filter bar */}
            <div className='mb-4 flex flex-wrap items-end gap-3 rounded-lg border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-700 dark:bg-zinc-800/40'>
                <div className='flex flex-col gap-1'>
                    <label htmlFor='face-min-quality' className='text-xs font-medium text-zinc-600 dark:text-zinc-400'>
                        Min Quality (0–100)
                    </label>
                    <input
                        id='face-min-quality'
                        type='number'
                        min={0}
                        max={100}
                        placeholder='0'
                        value={pendingMinQuality}
                        onChange={(e) => setPendingMinQuality(e.target.value)}
                        className='w-24 rounded border border-zinc-300 bg-white px-2 py-1 text-sm dark:border-zinc-600 dark:bg-zinc-900'
                    />
                </div>
                <div className='flex flex-col gap-1'>
                    <label
                        htmlFor='face-min-confidence'
                        className='text-xs font-medium text-zinc-600 dark:text-zinc-400'
                    >
                        Min Confidence (%)
                    </label>
                    <input
                        id='face-min-confidence'
                        type='number'
                        min={0}
                        max={100}
                        placeholder='0'
                        value={pendingMinConfidence}
                        onChange={(e) => setPendingMinConfidence(e.target.value)}
                        className='w-24 rounded border border-zinc-300 bg-white px-2 py-1 text-sm dark:border-zinc-600 dark:bg-zinc-900'
                    />
                </div>
                <div className='flex flex-col gap-1'>
                    <label htmlFor='face-sort-by' className='text-xs font-medium text-zinc-600 dark:text-zinc-400'>
                        Sort by
                    </label>
                    <select
                        id='face-sort-by'
                        value={pendingSortBy}
                        onChange={(e) => setPendingSortBy(e.target.value as UntaggedFaceParams['sort_by'])}
                        className='rounded border border-zinc-300 bg-white px-2 py-1 text-sm dark:border-zinc-600 dark:bg-zinc-900'
                    >
                        <option value='created_at'>Date added</option>
                        <option value='quality'>Quality</option>
                        <option value='confidence'>Confidence</option>
                    </select>
                </div>
                <div className='flex flex-col gap-1'>
                    <label htmlFor='face-sort-order' className='text-xs font-medium text-zinc-600 dark:text-zinc-400'>
                        Order
                    </label>
                    <select
                        id='face-sort-order'
                        value={pendingSortOrder}
                        onChange={(e) => setPendingSortOrder(e.target.value as UntaggedFaceParams['sort_order'])}
                        className='rounded border border-zinc-300 bg-white px-2 py-1 text-sm dark:border-zinc-600 dark:bg-zinc-900'
                    >
                        <option value='desc'>High / Newest first</option>
                        <option value='asc'>Low / Oldest first</option>
                    </select>
                </div>
                <label className='flex cursor-pointer items-center gap-2 self-end pb-1 text-sm text-zinc-700 dark:text-zinc-300'>
                    <input
                        type='checkbox'
                        checked={pendingGroupByImage}
                        onChange={(e) => setPendingGroupByImage(e.target.checked)}
                        className='rounded'
                    />
                    Group by image
                </label>
                <button
                    onClick={handleApply}
                    disabled={fetching}
                    className='self-end rounded bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50'
                >
                    {fetching ? 'Loading…' : 'Apply'}
                </button>
            </div>

            {error && <p className='mb-4 text-sm text-red-600'>{error}</p>}

            {displayFaces.length === 0 ? (
                <div className='flex flex-col items-center justify-center rounded-lg border border-dashed border-zinc-300 p-16 text-center'>
                    <Heading level={4}>All caught up!</Heading>
                    <Text className='mt-2 text-sm text-zinc-500'>No untagged faces remaining.</Text>
                </div>
            ) : (
                <>
                    <p className='mb-4 text-sm text-zinc-500'>
                        {displayFaces.length} face{displayFaces.length !== 1 ? 's' : ''} to review
                        {hasMore && ' (more are loaded automatically as you work through the queue)'}
                    </p>
                    <div
                        className={`grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 ${fetching ? 'opacity-50' : ''}`}
                    >
                        {displayFaces.map((face, i) => (
                            <div
                                key={face.face_id}
                                className='group relative overflow-hidden rounded-lg border border-zinc-200 dark:border-zinc-700'
                            >
                                <FaceThumbnail
                                    faceId={face.face_id}
                                    label={`Review face ${i + 1}`}
                                    onClick={() => setLightboxIndex(i)}
                                />
                                <span className='absolute top-1.5 right-1.5 rounded-full bg-black/60 px-1.5 py-0.5 text-[10px] text-white'>
                                    {Math.round(face.detection_confidence * 100)}%
                                </span>
                                {face.suggested_person_name && (
                                    <span className='absolute bottom-1.5 left-1.5 max-w-[80%] truncate rounded bg-indigo-600/80 px-1.5 py-0.5 text-[10px] text-white'>
                                        {face.suggested_person_name}
                                    </span>
                                )}
                            </div>
                        ))}
                    </div>
                </>
            )}

            {lightboxIndex !== null && (
                <FaceLightboxModal
                    faces={displayFaces}
                    currentIndex={lightboxIndex}
                    onClose={() => setLightboxIndex(null)}
                    onNavigate={setLightboxIndex}
                    onTagged={handleTagged}
                    onDeleted={handleDeleted}
                    onSuggestionUpdated={handleSuggestionUpdated}
                />
            )}
        </PageContentBlock>
    );
};

export default FaceTaggingContainer;
