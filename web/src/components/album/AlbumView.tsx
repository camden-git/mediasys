import React, { useEffect, useMemo, useRef, useState, useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { getAlbumContentsWithRating, getAlbumDownloadUrl, getBannerUrl, getOriginalImageUrl } from '../../api.ts';
import { FileInfo } from '../../types.ts';
import { CameraIcon, MapPinIcon, PhotoIcon, ArrowDownIcon, ShareIcon, SparklesIcon } from '@heroicons/react/16/solid';
import { useFlash } from '../../hooks/useFlash.ts';
import FlashMessageRender from '../elements/FlashMessageRender.tsx';
import DownloadDialog from './DownloadDialog.tsx';
import ShareChunksDialog from './ShareChunksDialog.tsx';
import PhotoPageLayout from './PhotoPageLayout.tsx';
import { usePublicAlbumDetail, usePublicAlbumContents } from '../../api/query/usePublicAlbum';

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

    const { data: currentAlbum, isLoading, error: albumError } = usePublicAlbumDetail(identifier);
    const { data: contentsData, fetchNextPage, hasNextPage, isFetchingNextPage } = usePublicAlbumContents(identifier);

    const allFiles = useMemo(() => contentsData?.pages.flatMap((p) => p.files ?? []) ?? [], [contentsData]);

    const directoryListingMeta = contentsData?.pages[contentsData.pages.length - 1];

    const { addFlash } = useFlash();

    const [selectedImage, setSelectedImage] = useState<FileInfo | null>(null);
    const [downloadModalOpen, setDownloadModalOpen] = useState(false);
    const [shareChunksDialogOpen, setShareChunksDialogOpen] = useState(false);
    const [shareChunks, setShareChunks] = useState<FileInfo[][]>([]);
    const [currentChunkIndex, setCurrentChunkIndex] = useState(0);
    const [isSharing, setIsSharing] = useState(false);
    const [shareProgress, setShareProgress] = useState<ShareProgress>(null);

    const navigate = useNavigate();

    // Highlights mode: ?highlights=1 in URL enables min_rating=4 filter
    const searchParams = new URLSearchParams(typeof window !== 'undefined' ? window.location.search : '');
    const [highlightsMode, setHighlightsMode] = useState(searchParams.get('highlights') === '1');
    const [highlightsListing, setHighlightsListing] = useState<typeof directoryListingMeta | null>(null);

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

    // Fetch highlights when toggled on
    useEffect(() => {
        if (!highlightsMode || !currentAlbum) {
            setHighlightsListing(null);
            return;
        }
        const controller = new AbortController();
        (async () => {
            try {
                const data = await getAlbumContentsWithRating(
                    currentAlbum.slug ?? String(currentAlbum.id),
                    { offset: 0, limit: 200, min_rating: 4 },
                    controller.signal,
                );
                setHighlightsListing(data);
            } catch {
                // ignore abort
            }
        })();
        return () => controller.abort();
    }, [highlightsMode, currentAlbum]);

    const handleToggleHighlights = () => {
        const next = !highlightsMode;
        setHighlightsMode(next);
        const base = `/album/${identifier}`;
        navigate(next ? `${base}?highlights=1` : base, { replace: true });
    };

    const activeFiles = highlightsMode ? (highlightsListing?.files ?? []) : allFiles;
    const imageFiles = useMemo(() => activeFiles.filter((file) => !file.is_dir && file.thumbnail_path), [activeFiles]);

    const canLoadMore = useMemo(() => {
        if (highlightsMode) return false;
        return !!hasNextPage;
    }, [hasNextPage, highlightsMode]);

    const loadMore = useCallback(async () => {
        if (isFetchingNextPage || !canLoadMore) return;
        await fetchNextPage();
    }, [fetchNextPage, isFetchingNextPage, canLoadMore]);

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
        setSelectedImage(null);
    }, [currentAlbum?.id, currentAlbum?.slug]);

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

        if (canLoadMore) {
            void loadMore();
        } else if (contentsData) {
            navigate(`/album/${identifier}`, { replace: true });
        }
    }, [
        imagePathFromUrl,
        imageFiles,
        selectedImage,
        canLoadMore,
        contentsData,
        loadMore,
        navigate,
        identifier,
        normalizePath,
    ]);

    useEffect(() => {
        if (!sentinelRef.current) return;
        const el = sentinelRef.current;
        const observer = new IntersectionObserver(
            (entries) => {
                const entry = entries[0];
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

    const handleImageClick = useCallback(
        (image: FileInfo) => {
            setSelectedImage(image);
            navigate(`/album/${identifier}/image/${encodeImagePath(image.path)}`, { replace: false });
        },
        [navigate, identifier, encodeImagePath],
    );

    const handleCloseLightbox = useCallback(() => {
        setSelectedImage(null);
        navigate(`/album/${identifier}`, { replace: true });
    }, [navigate, identifier]);

    const selectedIndex = useMemo(() => {
        if (!selectedImage) return -1;
        return imageFiles.findIndex((f) => f.path === selectedImage.path);
    }, [selectedImage, imageFiles]);

    const totalImageCount =
        (highlightsMode ? highlightsListing?.total : directoryListingMeta?.total) ?? imageFiles.length;
    const canPrev = selectedIndex > 0;
    const canNext = selectedIndex >= 0 && selectedIndex < totalImageCount - 1;

    const handlePrevImage = useCallback(() => {
        if (!canPrev) return;
        const prev = imageFiles[selectedIndex - 1];
        if (prev) {
            setSelectedImage(prev);
            navigate(`/album/${identifier}/image/${encodeImagePath(prev.path)}`, { replace: true });
        }
    }, [canPrev, imageFiles, selectedIndex, navigate, identifier, encodeImagePath]);

    const handleNextImage = useCallback(() => {
        if (selectedIndex < imageFiles.length - 1) {
            const next = imageFiles[selectedIndex + 1];
            setSelectedImage(next);
            navigate(`/album/${identifier}/image/${encodeImagePath(next.path)}`, { replace: true });
        } else if (canLoadMore) {
            pendingAdvanceRef.current = true;
            void loadMore();
        }
    }, [selectedIndex, imageFiles, canLoadMore, loadMore, navigate, identifier, encodeImagePath]);

    // Proactive preload: fetch next page when within 20 images of end
    useEffect(() => {
        if (!selectedImage) return;
        if (selectedIndex >= imageFiles.length - 20 && canLoadMore) {
            void loadMore();
        }
    }, [selectedIndex, imageFiles.length, canLoadMore, selectedImage, loadMore]);

    // Flush pending advance when new images arrive
    useEffect(() => {
        if (pendingAdvanceRef.current && imageFiles.length > selectedIndex + 1) {
            pendingAdvanceRef.current = false;
            const nextImage = imageFiles[selectedIndex + 1];
            setSelectedImage(nextImage);
            navigate(`/album/${identifier}/image/${encodeImagePath(nextImage.path)}`, { replace: true });
        }
    }, [imageFiles.length, selectedIndex, imageFiles, navigate, identifier, encodeImagePath]);

    const handleDownloadZip = () => {
        if (!currentAlbum?.zip_path) return;
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

    const error = albumError ? (albumError as Error).message : null;

    const albumMetadata = (
        <>
            <div className='flex items-center gap-1.5'>
                <PhotoIcon className='size-4 text-gray-950/40' />
                {(highlightsMode ? highlightsListing?.total : directoryListingMeta?.total) ?? imageFiles.length} photos
                {highlightsMode && <span className='text-yellow-500'> (highlights)</span>}
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
                isLoading={isLoading}
                error={error}
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
