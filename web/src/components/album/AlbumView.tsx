import React, { useEffect, useMemo, useRef, useState, useCallback } from 'react';
import { useStoreState, State } from 'easy-peasy';
import { useStoreActions, Actions } from 'easy-peasy';
import { StoreModel } from '../../store';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import ErrorMessage from '../elements/ErrorMessage.tsx';
import { Heading } from '../elements/Heading.tsx';
import AdvancedImageGrid from './AdvancedImageGrid.tsx';
import { getAlbumDownloadUrl, getBannerUrl, getOriginalImageUrl } from '../../api.ts';
import { FileInfo } from '../../types.ts';
import ImageLightbox from './ImageLightbox.tsx';
//
//
import { ArrowDownIcon, CameraIcon, MapPinIcon, PhotoIcon, ShareIcon } from '@heroicons/react/16/solid';
import { useFlash } from '../../hooks/useFlash.ts';
import FlashMessageRender from '../elements/FlashMessageRender.tsx';
import DownloadDialog from './DownloadDialog.tsx';
import ShareChunksDialog from './ShareChunksDialog.tsx';

// max payload size for a single Web Share operation
const MAX_SHARE_CHUNK_BYTES = 40 * 1024 * 1024;

type ShareProgress = { current: number; total: number; size: number } | null;

function chunkImagesBySize(images: FileInfo[], maxBytes: number): FileInfo[][] {
    const chunks: FileInfo[][] = [];
    let currentChunk: FileInfo[] = [];
    let currentSize = 0;

    for (const image of images) {
        if (currentSize + image.size > maxBytes) {
            if (currentChunk.length > 0) chunks.push(currentChunk);
            currentChunk = [image];
            currentSize = image.size;
        } else {
            currentChunk.push(image);
            currentSize += image.size;
        }
    }

    if (currentChunk.length > 0) chunks.push(currentChunk);
    return chunks;
}

// Dialog components moved to separate files for clarity

const AlbumView: React.FC = () => {
    const { currentAlbum, directoryListing, isLoading, error } = useStoreState(
        (state: State<StoreModel>) => state.contentView,
    );
    const fetchMoreAlbumContents = useStoreActions(
        (actions: Actions<StoreModel>) => actions.contentView.fetchMoreAlbumContents,
    );
    const { addFlash } = useFlash();

    const [selectedImage, setSelectedImage] = useState<FileInfo | null>(null);
    const [downloadModalOpen, setDownloadModalOpen] = useState(false);
    const [shareChunksDialogOpen, setShareChunksDialogOpen] = useState(false);
    const [shareChunks, setShareChunks] = useState<FileInfo[][]>([]);
    const [currentChunkIndex, setCurrentChunkIndex] = useState(0);
    const [isSharing, setIsSharing] = useState(false);
    const [shareProgress, setShareProgress] = useState<ShareProgress>(null);

    const sentinelRef = useRef<HTMLDivElement | null>(null);
    const isFetchingMoreRef = useRef(false);
    const hasUserScrolledRef = useRef(false);
    const prefillCountRef = useRef(0);

    const imageFiles = useMemo(() => {
        if (!directoryListing?.files) {
            return [];
        }
        return directoryListing.files.filter((file) => !file.is_dir && file.thumbnail_path);
    }, [directoryListing]);

    const canLoadMore = useMemo(() => {
        if (!directoryListing) return false;
        return Boolean(directoryListing.has_more);
    }, [directoryListing]);

    const loadMore = useCallback(async () => {
        if (!currentAlbum || isFetchingMoreRef.current || !canLoadMore) return;
        isFetchingMoreRef.current = true;
        try {
            await fetchMoreAlbumContents({ identifier: currentAlbum.slug ?? String(currentAlbum.id), limit: 50 });
        } finally {
            isFetchingMoreRef.current = false;
        }
    }, [currentAlbum, fetchMoreAlbumContents, canLoadMore]);

    useEffect(() => {
        const onScroll = () => {
            if (window.scrollY > 0) {
                hasUserScrolledRef.current = true;
            }
        };
        window.addEventListener('scroll', onScroll, { passive: true });
        return () => window.removeEventListener('scroll', onScroll);
    }, []);

    // reset guards when album changes
    useEffect(() => {
        prefillCountRef.current = 0;
        hasUserScrolledRef.current = false;
    }, [currentAlbum?.id, currentAlbum?.slug]);

    useEffect(() => {
        if (!sentinelRef.current) return;
        const el = sentinelRef.current;
        const observer = new IntersectionObserver(
            (entries) => {
                const entry = entries[0];
                // start fetching before fully visible to pre-load
                if (entry.isIntersecting && hasUserScrolledRef.current) {
                    void loadMore();
                }
            },
            { rootMargin: '0px 0px 300px 0px', threshold: 0.01 },
        );
        observer.observe(el);
        return () => observer.disconnect();
    }, [loadMore]);

    // if page content doesn't fill the viewport, prefetch at most once without requiring scroll
    useEffect(() => {
        if (!canLoadMore) return;
        const docEl = document.documentElement;
        const viewportH = window.innerHeight || 0;
        const contentH = docEl.scrollHeight || 0;
        if (contentH <= viewportH + 40 && prefillCountRef.current < 1) {
            prefillCountRef.current += 1;
            void loadMore();
        }
    }, [imageFiles.length, canLoadMore, loadMore]);

    const handleImageClick = (image: FileInfo) => {
        setSelectedImage(image);
    };

    const handleCloseLightbox = () => {
        setSelectedImage(null);
    };

    const selectedIndex = useMemo(() => {
        if (!selectedImage) return -1;
        return imageFiles.findIndex((f) => f.path === selectedImage.path);
    }, [selectedImage, imageFiles]);

    const canPrev = selectedIndex > 0;
    const canNext = selectedIndex >= 0 && selectedIndex < imageFiles.length - 1;

    const handlePrevImage = () => {
        if (!canPrev) return;
        const prev = imageFiles[selectedIndex - 1];
        if (prev) setSelectedImage(prev);
    };

    const handleNextImage = () => {
        if (!canNext) return;
        const next = imageFiles[selectedIndex + 1];
        if (next) setSelectedImage(next);
    };

    const handleDownloadZip = () => {
        if (!currentAlbum?.zip_path) {
            return;
        }

        const link = document.createElement('a');
        link.href = getAlbumDownloadUrl(currentAlbum?.slug);

        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        setDownloadModalOpen(false);
    };

    const shareChunk = async (chunk: FileInfo[], chunkNumber: number, totalChunks: number) => {
        setIsSharing(true);
        setShareProgress(null);
        try {
            const files: File[] = [];
            for (let i = 0; i < chunk.length; i++) {
                const image = chunk[i];
                setShareProgress({
                    current: i + 1,
                    total: chunk.length,
                    size: chunk.reduce((sum, img) => sum + img.size, 0),
                });

                const response = await fetch(getOriginalImageUrl(image.path));
                const blob = await response.blob();
                const file = new File([blob], `photo${i + 1}.jpg`, { type: 'image/jpeg' });
                files.push(file);
            }

            await navigator.share({
                files: files,
                title: `${currentAlbum?.name} (Part ${chunkNumber} of ${totalChunks})`,
                text: `Check out these photos from ${currentAlbum?.name}! (Part ${chunkNumber} of ${totalChunks})`,
            });
        } catch (error) {
            console.error('Error sharing images:', error);
        } finally {
            setIsSharing(false);
            setShareProgress(null);
        }
    };

    const handleShare = async () => {
        if (!navigator.share) {
            addFlash({
                key: 'album',
                type: 'error',
                title: 'Failed to share',
                message:
                    'Your browser does not support the Web Share API. Please consider downloading the album zip instead.',
            });
            return;
        }

        if (!currentAlbum || imageFiles.length === 0) {
            return;
        }

        const chunks = chunkImagesBySize(imageFiles, MAX_SHARE_CHUNK_BYTES);

        if (chunks.length === 0) {
            console.warn('No images could be shared');
            return;
        }

        if (chunks.length === 1) {
            await shareChunk(chunks[0], 1, 1);
            return;
        }

        setShareChunks(chunks);
        setCurrentChunkIndex(0);
        setShareChunksDialogOpen(true);
    };

    const handleShareNextChunk = async () => {
        if (currentChunkIndex < shareChunks.length - 1) {
            setCurrentChunkIndex(currentChunkIndex + 1);
        } else {
            setShareChunksDialogOpen(false);
        }
    };

    const handleShareCurrentChunk = async () => {
        await shareChunk(shareChunks[currentChunkIndex], currentChunkIndex + 1, shareChunks.length);
    };

    return (
        <>
            <FlashMessageRender byKey={'album'} />
            <div className='relative mx-auto'>
                <div className='absolute inset-x-0 top-0 -z-10 h-80 overflow-hidden rounded-t-2xl mask-b-from-60% sm:h-88 md:h-112 lg:h-128'>
                    {currentAlbum?.banner_image_path && (
                        <img
                            alt=''
                            src={getBannerUrl(currentAlbum?.banner_image_path)}
                            className='absolute inset-0 h-full w-full mask-l-from-60% object-cover object-center opacity-40'
                        />
                    )}
                    <div className='absolute inset-0 rounded-t-2xl outline-1 -outline-offset-1 outline-gray-950/10 dark:outline-white/10' />
                </div>
                <div className='mx-auto'>
                    <div className='relative'>
                        <div className='px-8 pt-48 pb-12 lg:py-24'>
                            <h1 className='sr-only'>{currentAlbum?.name} overview</h1>
                            <Heading className={'truncate font-bold'} huge>
                                {currentAlbum?.name}
                            </Heading>
                            <p className='mt-7 max-w-lg text-base/7 text-pretty text-gray-600 dark:text-gray-400'>
                                {currentAlbum?.description}
                            </p>
                            <div className='mt-6 flex flex-wrap items-center gap-x-4 gap-y-3 text-sm/7 font-semibold text-gray-950 sm:gap-3'>
                                <div className='flex items-center gap-1.5'>
                                    <PhotoIcon className='size-4 text-gray-950/40' />
                                    {directoryListing?.total ?? directoryListing?.files.length ?? 0} photos
                                </div>
                                <span className='hidden text-gray-950/25 sm:inline dark:text-white/25'>&middot;</span>
                                <div className='flex items-center gap-1.5'>
                                    <CameraIcon className='size-4 text-gray-950/40' />
                                    {currentAlbum?.artists && currentAlbum.artists.length > 0
                                        ? currentAlbum.artists
                                              .map((u) =>
                                                  u.first_name || u.last_name
                                                      ? `${u.first_name ?? ''} ${u.last_name ?? ''}`.trim()
                                                      : u.username,
                                              )
                                              .join(', ')
                                        : ''}
                                </div>
                                {currentAlbum?.location && (
                                    <>
                                        <span className='hidden text-gray-950/25 sm:inline dark:text-white/25'>
                                            &middot;
                                        </span>
                                        <div className='flex items-center gap-1.5'>
                                            <MapPinIcon className='size-4 text-gray-950/40' />
                                            {currentAlbum.location}
                                        </div>
                                    </>
                                )}
                            </div>
                            <div className='mt-10 flex gap-3'>
                                {currentAlbum?.zip_size && (
                                    <>
                                        <button
                                            onClick={() => setDownloadModalOpen(true)}
                                            className='inline-flex items-center gap-x-2 rounded-full bg-gray-950 px-3 py-0.5 text-sm/7 font-semibold text-white hover:bg-gray-800 dark:bg-gray-700 dark:hover:bg-gray-600'
                                        >
                                            <ArrowDownIcon className='size-2 fill-white' />
                                            Download
                                        </button>
                                        <DownloadDialog
                                            open={downloadModalOpen}
                                            onClose={setDownloadModalOpen}
                                            albumName={currentAlbum?.name}
                                            zipSize={currentAlbum.zip_size}
                                            onDownload={handleDownloadZip}
                                        />
                                    </>
                                )}

                                <ShareChunksDialog
                                    open={shareChunksDialogOpen}
                                    onClose={setShareChunksDialogOpen}
                                    images={imageFiles}
                                    chunks={shareChunks}
                                    currentIndex={currentChunkIndex}
                                    isSharing={isSharing}
                                    progress={shareProgress}
                                    onShareCurrent={handleShareCurrentChunk}
                                    onNext={handleShareNextChunk}
                                />

                                {imageFiles.length > 0 && (
                                    <button
                                        onClick={handleShare}
                                        disabled={isSharing}
                                        className='inline-flex items-center gap-x-2 rounded-full bg-gray-950 px-3 py-0.5 text-sm/7 font-semibold text-white hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-gray-700 dark:hover:bg-gray-600'
                                    >
                                        <ShareIcon className='size-2 fill-white' />
                                        {isSharing
                                            ? shareProgress
                                                ? `Processing ${shareProgress.current}/${shareProgress.total} (${(shareProgress.size / (1024 * 1024)).toFixed(1)}MB)`
                                                : 'Sharing...'
                                            : 'Experimental: Navigator Web Share API'}
                                    </button>
                                )}
                            </div>
                        </div>

                        <div className='mt-4'>
                            <ErrorMessage message={error} />

                            {isLoading && <LoadingSpinner />}

                            {!isLoading && !error && directoryListing && (
                                <AdvancedImageGrid
                                    images={imageFiles}
                                    targetRowHeight={280}
                                    boxSpacing={4}
                                    onImageClick={handleImageClick}
                                />
                            )}
                            {/* Sentinel for infinite scroll */}
                            {canLoadMore && imageFiles.length > 0 && <div ref={sentinelRef} className='h-1 w-full' />}
                            <ImageLightbox
                                image={selectedImage}
                                onClose={handleCloseLightbox}
                                onPrev={handlePrevImage}
                                onNext={handleNextImage}
                                canPrev={canPrev}
                                canNext={canNext}
                            />
                        </div>
                    </div>
                </div>
            </div>
        </>
    );
};

export default AlbumView;
