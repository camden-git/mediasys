import React from 'react';
import { useAlbumData } from '../../../../store/albumContextHooks';
import { getBannerUrl } from '../../../../api.ts';
import { Heading } from '../../../elements/Heading.tsx';
import { CameraIcon, MapPinIcon, PhotoIcon } from '@heroicons/react/16/solid';
import { deleteAlbumImage, listAlbumImages, uploadAlbumImagesBatched } from '../../../../api/admin/albums';
import AdvancedImageGrid from '../../../album/AdvancedImageGrid.tsx';
import { FileInfo } from '../../../../types.ts';
import { ListBulletIcon, RectangleStackIcon, Squares2X2Icon, TrashIcon } from '@heroicons/react/20/solid';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../elements/Table';
import { getThumbnailUrl } from '../../../../api.ts';
import UploadZone from './UploadZone.tsx';

const REQUIRED_TASKS = ['thumbnail', 'metadata', 'detection'];
const TOTAL_TASKS = REQUIRED_TASKS.length;

const OverviewContainer: React.FC = () => {
    const album = useAlbumData();
    type ItemState = {
        path: string;
        uploading?: boolean;
        uploaded?: boolean;
        tasks: Record<string, 'processing' | 'done' | 'error'>;
        currentTask?: string;
        error?: string;
    };
    const [items, setItems] = React.useState<Record<string, ItemState>>({});
    const [isUploading, setIsUploading] = React.useState(false);
    const apiUrl = (import.meta as any).env.VITE_API_URL as string | undefined;
    const authToken = localStorage.getItem('authToken');

    // Upload size / ETA tracking
    const fileSizesRef = React.useRef<Record<string, number>>({});
    const uploadStartRef = React.useRef<number | null>(null);
    const processingStartRef = React.useRef<number | null>(null);
    const abortControllerRef = React.useRef<AbortController | null>(null);

    React.useEffect(() => {
        return () => {
            abortControllerRef.current?.abort();
        };
    }, []);
    const [, setTick] = React.useState(0);
    // Always tick every second so ETA stays live during both upload and processing
    React.useEffect(() => {
        const id = setInterval(() => setTick((t) => t + 1), 1000);
        return () => clearInterval(id);
    }, []);

    // Debounced gallery refresh after file completions
    const refreshTimerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);
    const pendingRefreshRef = React.useRef(false);

    const scheduleRefresh = React.useCallback((fetchFn: () => void) => {
        pendingRefreshRef.current = true;
        if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
        refreshTimerRef.current = setTimeout(() => {
            pendingRefreshRef.current = false;
            fetchFn();
        }, 1000);
    }, []);

    React.useEffect(() => {
        try {
            const base = apiUrl && apiUrl.startsWith('http') ? new URL(apiUrl) : new URL(window.location.href);
            const wsProtocol = base.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${wsProtocol}//${base.host}/api/ws`;
            // Send the auth token as a WebSocket subprotocol rather than a query
            // parameter so it never ends up in server access logs.
            const ws = authToken ? new WebSocket(wsUrl, ['bearer', authToken]) : new WebSocket(wsUrl);
            ws.onmessage = (e) => {
                try {
                    const data = JSON.parse(e.data);
                    if (!data || (data.type !== 'upload' && data.type !== 'task')) return;
                    const rel = String(data.path || '');
                    setItems((prev) => {
                        const next = { ...prev };
                        const current = { ...(next[rel] || { path: rel, tasks: {} }) };
                        current.tasks = { ...current.tasks };
                        if (data.type === 'upload') {
                            if (data.status === 'uploading') {
                                current.uploading = true;
                                current.currentTask = 'upload';
                            }
                            if (data.status === 'uploaded') {
                                current.uploading = false;
                                current.uploaded = true;
                                if (current.currentTask === 'upload') current.currentTask = undefined;
                            }
                            if (data.status === 'error') current.error = data.error || 'upload error';
                        } else if (data.type === 'task') {
                            const task = String(data.task || '');
                            // Ignore tasks that aren't part of the tracked set (e.g. preview)
                            if (!REQUIRED_TASKS.includes(task)) return prev;
                            // Stamp processing start on the first task event
                            if (processingStartRef.current === null) {
                                processingStartRef.current = Date.now();
                            }
                            if (data.status === 'processing') {
                                current.tasks[task] = 'processing';
                                current.currentTask = task;
                            }
                            if (data.status === 'done') {
                                current.tasks[task] = 'done';
                                if (current.currentTask === task) current.currentTask = undefined;
                            }
                            if (data.status === 'error') {
                                current.tasks[task] = 'error';
                                if (current.currentTask === task) current.currentTask = undefined;
                            }
                        }
                        next[rel] = current;
                        return next;
                    });
                } catch (err) {
                    if ((import.meta as any).env.DEV) console.warn('Failed to parse websocket message', err);
                }
            };
            return () => ws.close();
        } catch (err) {
            if ((import.meta as any).env.DEV) console.warn('Invalid websocket base URL', apiUrl, err);
        }
    }, [apiUrl, authToken]);

    const [listing, setListing] = React.useState<{ path: string; files: FileInfo[] } | null>(null);
    const [isLoadingImages, setIsLoadingImages] = React.useState(false);
    type ViewMode = 'cascading' | 'grid' | 'table';
    const [viewMode, setViewMode] = React.useState<ViewMode>('cascading');
    const [scale, setScale] = React.useState<number>(180);

    const fetchImages = React.useCallback(async () => {
        setIsLoadingImages(true);
        try {
            const res = await listAlbumImages(album.id);
            setListing({ path: res.path, files: res.files.filter((f) => !f.is_dir) });
        } finally {
            setIsLoadingImages(false);
        }
    }, [album.id]);

    React.useEffect(() => {
        fetchImages();
    }, [fetchImages]);

    // Auto-refresh when items reach completion
    React.useEffect(() => {
        const allDone = Object.values(items).every(
            (it) =>
                it.error ||
                Object.values(it.tasks).filter((s) => s === 'done').length === TOTAL_TASKS ||
                Object.values(it.tasks).includes('error'),
        );
        if (Object.keys(items).length > 0 && allDone) {
            scheduleRefresh(fetchImages);
        }
    }, [items, scheduleRefresh, fetchImages]);

    const handleFiles = async (files: Array<{ file: File; relativePath: string }>) => {
        fileSizesRef.current = {};
        uploadStartRef.current = Date.now();
        processingStartRef.current = null;
        for (const { file, relativePath } of files) {
            let rel = relativePath.replace(/\\/g, '/').replace(/^\.\//, '').replace(/^\//, '');
            const slash = rel.indexOf('/');
            if (slash >= 0) rel = rel.slice(slash + 1);
            fileSizesRef.current[album.folder_path + '/' + rel] = file.size;
        }

        const controller = new AbortController();
        abortControllerRef.current = controller;
        setIsUploading(true);
        try {
            const result = await uploadAlbumImagesBatched(album.id, files, {
                concurrency: 3,
                maxRetries: 3,
                signal: controller.signal,
            });
            if (result.failed.length > 0) {
                setItems((prev) => {
                    const next = { ...prev };
                    for (const f of result.failed) {
                        const key = album.folder_path + '/' + (f.path || '');
                        next[key] = { ...(next[key] || { path: key, tasks: {} }), error: f.error };
                    }
                    return next;
                });
            }
        } catch {
            // aborted or unrecoverable — per-batch failures already surfaced above
        } finally {
            setIsUploading(false);
            abortControllerRef.current = null;
        }
    };

    const handleDeleteImage = async (image: FileInfo) => {
        const fullPath = image.path.startsWith('/') ? image.path.slice(1) : image.path;
        await deleteAlbumImage(album.id, fullPath);
        await fetchImages();
    };

    const handleClearCompleted = () => {
        setItems((prev) => {
            const next: Record<string, ItemState> = {};
            for (const [key, it] of Object.entries(prev)) {
                const doneCount = Object.values(it.tasks).filter((s) => s === 'done').length;
                const isFullyDone = doneCount === TOTAL_TASKS && !it.error;
                if (!isFullyDone) next[key] = it;
            }
            return next;
        });
    };

    const Toolbar = () => (
        <div className='flex flex-wrap items-center justify-between gap-3 rounded border bg-white px-3 py-2'>
            <div className='flex items-center gap-1'>
                <button
                    type='button'
                    onClick={() => setViewMode('cascading')}
                    className={`inline-flex items-center gap-1 rounded px-2 py-1 text-sm ${viewMode === 'cascading' ? 'bg-blue-600 text-white' : 'text-gray-700 hover:bg-gray-100'}`}
                    title='Cascading layout'
                >
                    <RectangleStackIcon className='h-4 w-4' />
                    Cascading
                </button>
                <button
                    type='button'
                    onClick={() => setViewMode('grid')}
                    className={`inline-flex items-center gap-1 rounded px-2 py-1 text-sm ${viewMode === 'grid' ? 'bg-blue-600 text-white' : 'text-gray-700 hover:bg-gray-100'}`}
                    title='Grid layout'
                >
                    <Squares2X2Icon className='h-4 w-4' />
                    Grid
                </button>
                <button
                    type='button'
                    onClick={() => setViewMode('table')}
                    className={`inline-flex items-center gap-1 rounded px-2 py-1 text-sm ${viewMode === 'table' ? 'bg-blue-600 text-white' : 'text-gray-700 hover:bg-gray-100'}`}
                    title='Table layout'
                >
                    <ListBulletIcon className='h-4 w-4' />
                    Table
                </button>
            </div>
            <div className='flex items-center gap-2'>
                <span className='text-xs text-gray-500'>Scale</span>
                <input
                    type='range'
                    min={100}
                    max={320}
                    step={10}
                    value={scale}
                    onChange={(e) => setScale(parseInt(e.target.value, 10))}
                    className='h-2 w-44 cursor-pointer appearance-none rounded-lg bg-gray-200'
                />
                <span className='w-10 text-right text-xs text-gray-600'>{scale}px</span>
            </div>
        </div>
    );

    const renderContent = () => {
        if (isLoadingImages) return <div className='py-6 text-sm text-gray-500'>Loading images…</div>;
        const images = listing?.files ?? [];
        if (images.length === 0) return <div className='py-6 text-sm text-gray-500'>No photos in this album yet.</div>;

        if (viewMode === 'cascading') {
            return <AdvancedImageGrid images={images} targetRowHeight={scale} boxSpacing={6} onImageClick={() => {}} />;
        }

        if (viewMode === 'grid') {
            const tile = Math.max(80, Math.min(480, scale));
            return (
                <div
                    className='grid gap-3'
                    style={{ gridTemplateColumns: `repeat(auto-fill, minmax(${Math.round(tile)}px, 1fr))` }}
                >
                    {images.map((img) => {
                        const backgroundImage = img.thumbnail_path
                            ? `url(${getThumbnailUrl(img.thumbnail_path)})`
                            : undefined;
                        return (
                            <div key={img.path} className='group relative overflow-hidden rounded border bg-gray-100'>
                                <div
                                    className='h-full w-full bg-cover bg-center'
                                    style={{ height: `${tile}px`, backgroundImage }}
                                />
                                <div className='pointer-events-none absolute inset-0 bg-black/0 transition group-hover:bg-black/20' />
                                <button
                                    onClick={() => handleDeleteImage(img)}
                                    className='absolute top-2 right-2 hidden rounded bg-white/90 p-1 text-red-600 shadow group-hover:block'
                                    title={`Delete ${img.name}`}
                                >
                                    <TrashIcon className='h-4 w-4' />
                                </button>
                                <div className='truncate px-2 py-1 text-xs text-gray-700'>{img.name}</div>
                            </div>
                        );
                    })}
                </div>
            );
        }

        // table view
        return (
            <Table striped bleed className='rounded-lg'>
                <TableHead>
                    <TableRow>
                        <TableHeader className='w-16'>Preview</TableHeader>
                        <TableHeader>Name</TableHeader>
                        <TableHeader className='w-24'>Dimensions</TableHeader>
                        <TableHeader className='w-28'>Size</TableHeader>
                        <TableHeader className='w-32'>Modified</TableHeader>
                        <TableHeader className='w-24'>Actions</TableHeader>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {images.map((img) => {
                        const thumb = img.thumbnail_path ? getThumbnailUrl(img.thumbnail_path) : undefined;
                        return (
                            <TableRow key={img.path}>
                                <TableCell>
                                    <div className='flex h-14 w-20 items-center justify-center rounded border bg-gray-100'>
                                        {thumb && (
                                            <img
                                                src={thumb}
                                                alt={img.name}
                                                className='max-h-full max-w-full object-contain'
                                            />
                                        )}
                                    </div>
                                </TableCell>
                                <TableCell className='max-w-[28rem] truncate'>{img.name}</TableCell>
                                <TableCell>{img.width && img.height ? `${img.width}×${img.height}` : '—'}</TableCell>
                                <TableCell>{(img.size / 1024).toFixed(0)} KB</TableCell>
                                <TableCell>{new Date(img.mod_time * 1000).toLocaleString()}</TableCell>
                                <TableCell>
                                    <button
                                        onClick={() => handleDeleteImage(img)}
                                        className='inline-flex items-center gap-1 rounded border px-2 py-1 text-xs text-red-600 hover:bg-red-50'
                                    >
                                        <TrashIcon className='h-4 w-4' /> Delete
                                    </button>
                                </TableCell>
                            </TableRow>
                        );
                    })}
                </TableBody>
            </Table>
        );
    };

    // Format helpers
    const formatBytes = (bytes: number): string => {
        if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
        if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
        return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
    };
    const formatEta = (sec: number): string => {
        if (sec < 60) return `${sec} second${sec === 1 ? '' : 's'} remaining`;
        if (sec < 3600) {
            const m = Math.ceil(sec / 60);
            return `${m} minute${m === 1 ? '' : 's'} remaining`;
        }
        const h = Math.floor(sec / 3600);
        const m = Math.ceil((sec % 3600) / 60);
        return `${h}h ${m}m remaining`;
    };

    // Progress panel helpers
    const itemList = Object.values(items);
    const hasItems = itemList.length > 0;
    const totalFiles = itemList.length;
    const processedFiles = itemList.filter((it) => {
        const doneCount = Object.values(it.tasks).filter((s) => s === 'done').length;
        return doneCount === TOTAL_TASKS || it.error;
    }).length;
    const hasCompleted = itemList.some((it) => {
        const doneCount = Object.values(it.tasks).filter((s) => s === 'done').length;
        return doneCount === TOTAL_TASKS && !it.error;
    });
    const overallPct = totalFiles > 0 ? Math.round((processedFiles / totalFiles) * 100) : 0;

    // Byte-level upload stats (derived from fileSizesRef + uploaded flags)
    const totalBytes = Object.values(fileSizesRef.current).reduce((a, b) => a + b, 0);
    const uploadedBytes = Object.entries(items)
        .filter(([, it]) => it.uploaded === true)
        .reduce((sum, [path]) => sum + (fileSizesRef.current[path] ?? 0), 0);
    const elapsedSec = uploadStartRef.current ? (Date.now() - uploadStartRef.current) / 1000 : 0;
    const uploadRate = elapsedSec > 1 ? uploadedBytes / elapsedSec : 0; // bytes/sec
    const remainingBytes = Math.max(0, totalBytes - uploadedBytes);
    const etaSec = isUploading && uploadRate > 0 && remainingBytes > 0 ? Math.ceil(remainingBytes / uploadRate) : null;

    // Processing ETA — task-based, active when upload is done but processing is ongoing
    const completedTasks = itemList.reduce(
        (sum, it) => sum + Object.values(it.tasks).filter((s) => s === 'done').length,
        0,
    );
    const totalTasks = totalFiles * TOTAL_TASKS;
    const processingElapsedSec = processingStartRef.current ? (Date.now() - processingStartRef.current) / 1000 : 0;
    const processingRate = processingElapsedSec > 2 ? completedTasks / processingElapsedSec : 0; // tasks/sec
    const remainingTasks = Math.max(0, totalTasks - completedTasks);
    const processingEtaSec =
        !isUploading && processingRate > 0 && remainingTasks > 0 ? Math.ceil(remainingTasks / processingRate) : null;

    const getBadge = (it: ItemState) => {
        const doneCount = Object.values(it.tasks).filter((s) => s === 'done').length;
        const hasError = Boolean(it.error) || Object.values(it.tasks).includes('error');
        if (hasError) {
            return (
                <span className='inline-flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700'>
                    Error
                </span>
            );
        }
        if (doneCount === TOTAL_TASKS) {
            return (
                <span className='inline-flex items-center gap-1 rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700'>
                    Done
                </span>
            );
        }
        if (it.currentTask && it.currentTask !== 'upload') {
            return (
                <span className='inline-flex items-center gap-1 rounded-full bg-yellow-100 px-2 py-0.5 text-xs font-medium text-yellow-700'>
                    Processing
                </span>
            );
        }
        if (it.uploading) {
            return (
                <span className='inline-flex items-center gap-1 rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700'>
                    Uploading
                </span>
            );
        }
        return (
            <span className='inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600'>
                Queued
            </span>
        );
    };

    const getFilePct = (it: ItemState) => {
        const doneCount = Object.values(it.tasks).filter((s) => s === 'done').length;
        if (it.uploading) return 10;
        if (it.uploaded && doneCount === 0) return 30;
        return Math.max(30, 30 + Math.round((doneCount / TOTAL_TASKS) * 70));
    };

    return (
        <div className='relative mx-auto'>
            <div className='absolute inset-x-0 top-0 -z-10 h-80 overflow-hidden rounded-t-2xl mask-b-from-60% sm:h-88 md:h-112 lg:h-128'>
                {album.banners?.[0] && (
                    <img
                        alt=''
                        src={getBannerUrl(album.banners[0].image_path)}
                        className='absolute inset-0 h-full w-full mask-l-from-60% object-cover object-center opacity-40'
                    />
                )}
                <div className='absolute inset-0 rounded-t-2xl outline-1 -outline-offset-1 outline-gray-950/10 dark:outline-white/10' />
            </div>
            <div className='mx-auto'>
                <div className='relative'>
                    <div className='px-8 pt-48 pb-12 lg:py-24'>
                        <h1 className='sr-only'>{album.name} overview</h1>
                        <Heading className={'truncate font-bold'} huge>
                            {album.name}
                        </Heading>
                        <p className='mt-7 max-w-lg text-base/7 text-pretty text-gray-600 dark:text-gray-400'>
                            {album.description}
                        </p>
                        <div className='mt-6 flex flex-wrap items-center gap-x-4 gap-y-3 text-sm/7 font-semibold text-gray-950 sm:gap-3'>
                            <div className='flex items-center gap-1.5'>
                                <PhotoIcon className='size-4 text-gray-950/40' />
                            </div>
                            <span className='hidden text-gray-950/25 sm:inline dark:text-white/25'>&middot;</span>
                            {album.artists && album.artists.length > 0 && (
                                <div className='flex items-center gap-1.5'>
                                    <CameraIcon className='size-4 text-gray-950/40' />
                                    {album.artists
                                        .map((u) =>
                                            u.first_name || u.last_name
                                                ? `${u.first_name ?? ''} ${u.last_name ?? ''}`.trim()
                                                : u.username,
                                        )
                                        .join(', ')}
                                </div>
                            )}
                            {album.location && (
                                <>
                                    <span className='hidden text-gray-950/25 sm:inline dark:text-white/25'>
                                        &middot;
                                    </span>
                                    <div className='flex items-center gap-1.5'>
                                        <MapPinIcon className='size-4 text-gray-950/40' />
                                        {album.location}
                                    </div>
                                </>
                            )}
                        </div>
                    </div>

                    {/* Upload zone */}
                    <div className='mt-4'>
                        <UploadZone onFiles={handleFiles} disabled={isUploading} />
                    </div>

                    {/* Progress panel — only shown when items exist */}
                    {hasItems && (
                        <div className='mt-4 rounded-lg border bg-white shadow-sm'>
                            {/* Header */}
                            <div className='flex items-center justify-between border-b px-4 py-3'>
                                <div className='flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1'>
                                    <span className='shrink-0 text-sm font-medium text-gray-800'>
                                        {processedFiles} / {totalFiles} files processed
                                    </span>
                                    {totalBytes > 0 && (
                                        <span className='text-xs text-gray-500'>
                                            {formatBytes(uploadedBytes)} of {formatBytes(totalBytes)} uploaded
                                            {etaSec !== null && <> &middot; {formatEta(etaSec)}</>}
                                            {processingEtaSec !== null && <> &middot; {formatEta(processingEtaSec)}</>}
                                        </span>
                                    )}
                                    <div className='h-2 w-32 shrink-0 overflow-hidden rounded-full bg-gray-200'>
                                        <div
                                            className='h-full rounded-full bg-blue-500 transition-all'
                                            style={{ width: `${overallPct}%` }}
                                        />
                                    </div>
                                </div>
                                <div className='flex shrink-0 items-center gap-3'>
                                    {isUploading && (
                                        <button
                                            type='button'
                                            onClick={() => abortControllerRef.current?.abort()}
                                            className='text-xs text-red-500 underline hover:text-red-700'
                                        >
                                            Cancel
                                        </button>
                                    )}
                                    {hasCompleted && (
                                        <button
                                            type='button'
                                            onClick={handleClearCompleted}
                                            className='text-xs text-gray-500 underline hover:text-gray-700'
                                        >
                                            Clear completed
                                        </button>
                                    )}
                                </div>
                            </div>

                            {/* Per-file rows */}
                            <div className='max-h-64 divide-y overflow-auto'>
                                {itemList
                                    .sort((a, b) => {
                                        const aDone =
                                            Object.values(a.tasks).filter((s) => s === 'done').length === TOTAL_TASKS;
                                        const bDone =
                                            Object.values(b.tasks).filter((s) => s === 'done').length === TOTAL_TASKS;
                                        if (aDone !== bDone) return Number(aDone) - Number(bDone);
                                        return a.path.localeCompare(b.path);
                                    })
                                    .map((it) => {
                                        const pct = getFilePct(it);
                                        const hasError = Boolean(it.error) || Object.values(it.tasks).includes('error');
                                        const displayName = it.path.replace(album.folder_path + '/', '');
                                        return (
                                            <div key={it.path} className='flex items-center gap-3 px-4 py-2.5'>
                                                {getBadge(it)}
                                                <div className='min-w-0 flex-1'>
                                                    <div
                                                        className='mb-1 truncate text-xs text-gray-700'
                                                        title={displayName}
                                                    >
                                                        {displayName}
                                                    </div>
                                                    <div className='h-1.5 w-full overflow-hidden rounded-full bg-gray-100'>
                                                        <div
                                                            className={`h-full rounded-full transition-all ${hasError ? 'bg-red-400' : 'bg-blue-500'}`}
                                                            style={{ width: `${pct}%` }}
                                                        />
                                                    </div>
                                                </div>
                                            </div>
                                        );
                                    })}
                            </div>
                        </div>
                    )}

                    <div className='mt-8 rounded-lg bg-white shadow'>
                        <div className='border-b border-gray-200 px-6 py-4'>
                            <h2 className='text-lg font-medium text-gray-900'>Photos</h2>
                        </div>
                        <div className='space-y-4 px-6 py-4'>
                            <Toolbar />
                            {renderContent()}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default OverviewContainer;
