import React, { useEffect, useMemo, useRef, useState, useCallback } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { getAlbumDownloadUrl, getBannerUrl, getOriginalImageUrl } from '../../api.ts';
import { FileInfo } from '../../types.ts';
import { CameraIcon, MapPinIcon, PhotoIcon, ArrowDownIcon, ShareIcon, SparklesIcon } from '@heroicons/react/16/solid';
import { useFlash } from '../../hooks/useFlash.ts';
import FlashMessageRender from '../elements/FlashMessageRender.tsx';
import DownloadDialog from './DownloadDialog.tsx';
import ShareChunksDialog from './ShareChunksDialog.tsx';
import PhotoPageLayout from './PhotoPageLayout.tsx';
import { usePublicAlbumDetail, usePublicAlbumContents, usePublicAlbumHighlights } from '../../api/query/usePublicAlbum';

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

const AlbumView: React.FC = () => {
    const routeParams = useParams<{ identifier: string; '*': string }>();
    const identifier = routeParams.identifier;
    const imagePathFromUrl = routeParams['*'] || null;

    const [searchParams] = useSearchParams();
    const highlightsMode = searchParams.get('highlights') === '1';
    // preserved on every in-album navigation so lightbox open/close keeps highlights mode
    const search = highlightsMode ? '?highlights=1' : '';

    const { data: currentAlbum, isLoading: albumLoading, error: albumError } = usePublicAlbumDetail(identifier);
    const contentsQuery = usePublicAlbumContents(identifier, !highlightsMode);
    const highlightsQuery = usePublicAlbumHighlights(identifier, highlightsMode);
    const listQuery = highlightsMode ? highlightsQuery : contentsQuery;
    const {
        data: listData,
        fetchNextPage,
        hasNextPage,
        isFetchingNextPage,
        isLoading: listLoading,
        isError: listIsError,
        error: listError,
        isFetchNextPageError,
    } = listQuery;

    const activeFiles = useMemo(() => listData?.pages.flatMap((p) => p.files ?? []) ?? [], [listData]);
    const listMeta = listData?.pages[listData.pages.length - 1];

    const { addFlash } = useFlash();

    const [selectedImage, setSelectedImage] = useState<FileInfo | null>(null);
    const [downloadModalOpen, setDownloadModalOpen] = useState(false);
    const [shareChunksDialogOpen, setShareChunksDialogOpen] = useState(false);
    const [shareChunks, setShareChunks] = useState<FileInfo[][]>([]);
    const [currentChunkIndex, setCurrentChunkIndex] = useState(0);
    const [isSharing, setIsSharing] = useState(false);
    const [shareProgress, setShareProgress] = useState<ShareProgress>(null);
    const [layoutVersion, setLayoutVersion] = useState(0);

    const navigate = useNavigate();

    const folderPrefix = currentAlbum?.folder_path ? currentAlbum.folder_path.replace(/\/?$/, '/') : null;

    const encodeImagePath = useCallback(
        (path: string) => {
            const stripped = (folderPrefix ? path.replace(folderPrefix, '') : path).replace(/^\//, '');
            return stripped.split('/').map(encodeURIComponent).join('/');
        },
        [folderPrefix],
    );

    const normalizePath = useCallback(
        (path: string) => (folderPrefix ? path.replace(folderPrefix, '') : path).replace(/^\//, ''),
        [folderPrefix],
    );

    const sentinelRef = useRef<HTMLDivElement | null>(null);
    const hasUserScrolledRef = useRef(false);
    const prefillCountRef = useRef(0);
    const pendingAdvanceRef = useRef(false);

    const handleToggleHighlights = () => {
        navigate(
            { pathname: `/album/${identifier}`, search: highlightsMode ? '' : '?highlights=1' },
            { replace: true },
        );
    };

    const imageFiles = useMemo(() => activeFiles.filter((file) => !file.is_dir && file.thumbnail_path), [activeFiles]);

    const canLoadMore = !!hasNextPage;

    // auto-loading stops after a failed page fetch; user actions (force) retry
    const loadMore = useCallback(
        async (force = false): Promise<boolean> => {
            if (isFetchingNextPage || !canLoadMore) return false;
            if (isFetchNextPageError && !force) return false;
            const result = await fetchNextPage();
            return !result.isError;
        },
        [fetchNextPage, isFetchingNextPage, canLoadMore, isFetchNextPageError],
    );

    const loadMoreRef = useRef(loadMore);
    useEffect(() => {
        loadMoreRef.current = loadMore;
    });

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
        pendingAdvanceRef.current = false;
        setSelectedImage(null);
    }, [currentAlbum?.id, currentAlbum?.slug]);

    // surface failed next-page loads (initial load errors are shown by the layout)
    useEffect(() => {
        if (!isFetchNextPageError) return;
        pendingAdvanceRef.current = false;
        addFlash({
            key: 'album',
            type: 'error',
            title: 'Failed to load more photos',
            message: (listError as Error | null)?.message ?? 'Something went wrong while loading more photos.',
        });
    }, [isFetchNextPageError, listError, addFlash]);

    // URL → state: seek to image from URL
    useEffect(() => {
        if (!imagePathFromUrl) {
            if (selectedImage) setSelectedImage(null);
            return;
        }

        if (selectedImage && normalizePath(selectedImage.path) === imagePathFromUrl) {
            return;
        }

        const found = imageFiles.find((f) => normalizePath(f.path) === imagePathFromUrl);
        if (found) {
            setSelectedImage(found);
            return;
        }

        // wait for the listing (and folder prefix) before deciding the image doesn't exist
        if (listLoading || listIsError || !listData || !currentAlbum) return;

        if (canLoadMore) {
            void loadMore();
        } else {
            navigate({ pathname: `/album/${identifier}`, search }, { replace: true });
        }
    }, [
        imagePathFromUrl,
        imageFiles,
        selectedImage,
        canLoadMore,
        listData,
        listLoading,
        listIsError,
        currentAlbum,
        loadMore,
        navigate,
        identifier,
        search,
        normalizePath,
    ]);

    const showSentinel = canLoadMore && imageFiles.length > 0 && !isFetchNextPageError;
    useEffect(() => {
        if (!showSentinel || !sentinelRef.current) return;
        const el = sentinelRef.current;
        const observer = new IntersectionObserver(
            (entries) => {
                const entry = entries[0];
                if (entry.isIntersecting && hasUserScrolledRef.current) {
                    void loadMoreRef.current();
                }
            },
            { rootMargin: '0px 0px 300px 0px', threshold: 0.01 },
        );
        observer.observe(el);
        return () => observer.disconnect();
    }, [showSentinel]);

    // if page content doesn't fill the viewport, prefetch at most once without requiring scroll.
    // Runs after the grid reports a committed layout so scrollHeight is meaningful.
    useEffect(() => {
        if (layoutVersion === 0 || !canLoadMore) return;
        const docEl = document.documentElement;
        const viewportH = window.innerHeight || 0;
        const contentH = docEl.scrollHeight || 0;
        if (contentH <= viewportH + 40 && prefillCountRef.current < 1) {
            prefillCountRef.current += 1;
            void loadMoreRef.current();
        }
    }, [layoutVersion, canLoadMore]);

    const handleLayoutComplete = useCallback(() => setLayoutVersion((v) => v + 1), []);

    const handleImageClick = useCallback(
        (image: FileInfo) => {
            setSelectedImage(image);
            navigate({ pathname: `/album/${identifier}/image/${encodeImagePath(image.path)}`, search });
        },
        [navigate, identifier, encodeImagePath, search],
    );

    const handleCloseLightbox = useCallback(() => {
        pendingAdvanceRef.current = false;
        setSelectedImage(null);
        navigate({ pathname: `/album/${identifier}`, search }, { replace: true });
    }, [navigate, identifier, search]);

    const selectedIndex = useMemo(() => {
        if (!selectedImage) return -1;
        return imageFiles.findIndex((f) => f.path === selectedImage.path);
    }, [selectedImage, imageFiles]);

    // server total also counts entries the client filters out, so only trust it while more pages remain
    const totalImageCount = canLoadMore ? Math.max(listMeta?.total ?? 0, imageFiles.length) : imageFiles.length;
    const canPrev = selectedIndex > 0;
    const canNext = selectedIndex >= 0 && (selectedIndex < imageFiles.length - 1 || canLoadMore);

    const goToImage = useCallback(
        (image: FileInfo) => {
            setSelectedImage(image);
            navigate(
                { pathname: `/album/${identifier}/image/${encodeImagePath(image.path)}`, search },
                { replace: true },
            );
        },
        [navigate, identifier, encodeImagePath, search],
    );

    const handlePrevImage = useCallback(() => {
        pendingAdvanceRef.current = false;
        if (!canPrev) return;
        const prev = imageFiles[selectedIndex - 1];
        if (prev) goToImage(prev);
    }, [canPrev, imageFiles, selectedIndex, goToImage]);

    const handleNextImage = useCallback(() => {
        if (selectedIndex < imageFiles.length - 1) {
            goToImage(imageFiles[selectedIndex + 1]);
        } else if (canLoadMore) {
            pendingAdvanceRef.current = true;
            void loadMore(true).then((ok) => {
                if (!ok) pendingAdvanceRef.current = false;
            });
        }
    }, [selectedIndex, imageFiles, canLoadMore, loadMore, goToImage]);

    // Proactive preload: fetch next page when within 20 images of end
    useEffect(() => {
        if (!selectedImage) return;
        if (selectedIndex >= imageFiles.length - 20 && canLoadMore) {
            void loadMore();
        }
    }, [selectedIndex, imageFiles.length, canLoadMore, selectedImage, loadMore]);

    // Flush pending advance when new images arrive
    useEffect(() => {
        if (pendingAdvanceRef.current && selectedImage && imageFiles.length > selectedIndex + 1) {
            pendingAdvanceRef.current = false;
            goToImage(imageFiles[selectedIndex + 1]);
        }
    }, [imageFiles, selectedIndex, selectedImage, goToImage]);

    const handleDownloadZip = () => {
        const zipId = currentAlbum?.slug || identifier;
        if (!currentAlbum?.zip_ready || !zipId) return;
        const link = document.createElement('a');
        link.href = getAlbumDownloadUrl(zipId);
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
        if (!currentAlbum || imageFiles.length === 0) return;
        const chunks = chunkImagesBySize(imageFiles, MAX_SHARE_CHUNK_BYTES);
        if (chunks.length === 0) return;
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

    const error = albumError
        ? (albumError as Error).message
        : listIsError && !isFetchNextPageError
          ? ((listError as Error | null)?.message ?? 'Failed to load photos')
          : null;
    const loadMoreError = isFetchNextPageError ? 'Failed to load more photos.' : null;
    const photoCountLabel = `${totalImageCount} ${totalImageCount === 1 ? 'photo' : 'photos'}`;
    const artistNames = (currentAlbum?.artists ?? [])
        .map((u) => (u.first_name || u.last_name ? `${u.first_name ?? ''} ${u.last_name ?? ''}`.trim() : u.username))
        .filter(Boolean)
        .join(', ');

    const albumMetadata = (
        <>
            <div className='flex items-center gap-1.5'>
                <PhotoIcon className='size-4 text-gray-950/40' />
                {photoCountLabel}
                {highlightsMode && <span className='text-yellow-500'> (highlights)</span>}
            </div>
            {artistNames && (
                <>
                    <span className='hidden text-gray-950/25 sm:inline dark:text-white/25'>&middot;</span>
                    <div className='flex items-center gap-1.5'>
                        <CameraIcon className='size-4 text-gray-950/40' />
                        {artistNames}
                    </div>
                </>
            )}
            {currentAlbum?.location && (
                <>
                    <span className='hidden text-gray-950/25 sm:inline dark:text-white/25'>&middot;</span>
                    <div className='flex items-center gap-1.5'>
                        <MapPinIcon className='size-4 text-gray-950/40' />
                        {currentAlbum.location}
                    </div>
                </>
            )}
        </>
    );

    const albumActions = (
        <>
            <button
                onClick={handleToggleHighlights}
                className={`inline-flex items-center gap-x-2 rounded-full px-3 py-0.5 text-sm/7 font-semibold transition-colors ${
                    highlightsMode
                        ? 'bg-yellow-400 text-gray-950 hover:bg-yellow-300'
                        : 'bg-gray-950/10 text-gray-950 hover:bg-gray-950/20 dark:bg-white/10 dark:text-white dark:hover:bg-white/20'
                }`}
            >
                <SparklesIcon className='size-2' />
                Highlights
            </button>
            {currentAlbum?.zip_ready && (
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
                        : 'Share'}
                </button>
            )}
        </>
    );

    return (
        <>
            <FlashMessageRender byKey={'album'} />
            <PhotoPageLayout
                title={currentAlbum?.name ?? ''}
                description={currentAlbum?.description}
                bannerUrls={currentAlbum?.banners?.map(getBannerUrl)}
                metadata={albumMetadata}
                actions={albumActions}
                images={imageFiles}
                isLoading={albumLoading || listLoading}
                error={error}
                loadMoreError={loadMoreError}
                onRetryLoadMore={() => void loadMore(true)}
                onLayoutComplete={handleLayoutComplete}
                hasMore={canLoadMore}
                sentinelRef={sentinelRef}
                selectedImage={selectedImage}
                selectedIndex={selectedIndex}
                totalImages={totalImageCount}
                onImageClick={handleImageClick}
                onClose={handleCloseLightbox}
                onPrev={handlePrevImage}
                onNext={handleNextImage}
                canPrev={canPrev}
                canNext={canNext}
            />
        </>
    );
};

export default AlbumView;
