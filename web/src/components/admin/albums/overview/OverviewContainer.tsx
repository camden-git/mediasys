import React from 'react';
import { useStoreState } from '../../../../store/hooks.ts';
import { getBannerUrl } from '../../../../api.ts';
import { Heading } from '../../../elements/Heading.tsx';
import { CameraIcon, MapPinIcon, PhotoIcon } from '@heroicons/react/16/solid';
import { Button } from '../../../elements/Button';
import { deleteAlbumImage, listAlbumImages, uploadAlbumImagesBatched } from '../../../../api/admin/albums';
import AdvancedImageGrid from '../../../album/AdvancedImageGrid.tsx';
import { FileInfo } from '../../../../types.ts';
import { ListBulletIcon, RectangleStackIcon, Squares2X2Icon, TrashIcon } from '@heroicons/react/20/solid';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../elements/Table';
import { getThumbnailUrl } from '../../../../api.ts';

const OverviewContainer: React.FC = () => {
    const album = useStoreState((state) => state.albumContext.data!);
    type ItemState = {
        path: string;
        uploading?: boolean;
        uploaded?: boolean;
        tasks: Record<string, 'processing' | 'done' | 'error'>;
        currentTask?: string; // 'upload' | 'thumbnail' | 'metadata' | 'detection'
        error?: string;
    };
    const [items, setItems] = React.useState<Record<string, ItemState>>({});
    const [isUploading, setIsUploading] = React.useState(false);
    const fileInputRef = React.useRef<HTMLInputElement>(null);
    const apiUrl = (import.meta as any).env.VITE_API_URL as string | undefined;
    const authToken = localStorage.getItem('authToken');

    React.useEffect(() => {
        try {
            const base = apiUrl && apiUrl.startsWith('http') ? new URL(apiUrl) : new URL(window.location.href);
            const wsProtocol = base.protocol === 'https:' ? 'wss:' : 'ws:';
            const tokenQuery = authToken ? `?token=${encodeURIComponent(authToken)}` : '';
            const wsUrl = `${wsProtocol}//${base.host}/api/ws${tokenQuery}`;
            const ws = new WebSocket(wsUrl);
            ws.onmessage = (e) => {
                try {
                    const data = JSON.parse(e.data);
                    if (!data || (data.type !== 'upload' && data.type !== 'task')) return;
                    const rel = String(data.path || '');
                    setItems((prev) => {
                        const next = { ...prev };
                        const current = next[rel] || { path: rel, tasks: {} };
                        if (data.type === 'upload') {
                            if (data.status === 'uploading') {
                                current.uploading = true;
                                current.currentTask = 'upload';
                            }
                            if (data.status === 'uploaded') {
                                current.uploading = false;
                                current.uploaded = true;
                                // Clear upload task indicator when upload finishes
                                if (current.currentTask === 'upload') current.currentTask = undefined;
                            }
                            if (data.status === 'error') current.error = data.error || 'upload error';
                        } else if (data.type === 'task') {
                            const task = String(data.task || '');
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

    const handleFolderChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const files = e.target.files;
        if (!files || files.length === 0) return;
        setIsUploading(true);
        try {
            const items = Array.from(files).map((f) => ({
                file: f,
                relativePath: (f as any).webkitRelativePath || f.name,
            }));
            await uploadAlbumImagesBatched(album.id, items, { batchSize:    5, concurrency: 3, requestTimeoutMs: 0 });
        } finally {
            setIsUploading(false);
            if (fileInputRef.current) fileInputRef.current.value = '';
        }
    };

    const [listing, setListing] = React.useState<{ path: string; files: FileInfo[] } | null>(null);
    const [isLoadingImages, setIsLoadingImages] = React.useState(false);
    type ViewMode = 'cascading' | 'grid' | 'table';
    const [viewMode, setViewMode] = React.useState<ViewMode>('cascading');
    const [scale, setScale] = React.useState<number>(180); // affects row height or tile size

    const fetchImages = React.useCallback(async () => {
        setIsLoadingImages(true);
        try {
            const res = await listAlbumImages(album.id);
            // Only keep raster images
            setListing({ path: res.path, files: res.files.filter((f) => !f.is_dir) });
        } finally {
            setIsLoadingImages(false);
        }
    }, [album.id]);

    React.useEffect(() => {
        fetchImages();
    }, [fetchImages]);

    const handleDeleteImage = async (image: FileInfo) => {
        const fullPath = image.path.startsWith('/') ? image.path.slice(1) : image.path;
        await deleteAlbumImage(album.id, fullPath);
        await fetchImages();
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
            return (
                <AdvancedImageGrid images={images} targetRowHeight={scale} boxSpacing={6} onImageClick={() => {}} />
            );
        }

        if (viewMode === 'grid') {
            const tile = Math.max(80, Math.min(480, scale));
            return (
                <div
                    className='grid gap-3'
                    style={{ gridTemplateColumns: `repeat(auto-fill, minmax(${Math.round(tile)}px, 1fr))` }}
                >
                    {images.map((img) => {
                        const backgroundImage = img.thumbnail_path ? `url(${getThumbnailUrl(img.thumbnail_path)})` : undefined;
                        return (
                            <div key={img.path} className='group relative overflow-hidden rounded border bg-gray-100'>
                                <div
                                    className='h-full w-full bg-cover bg-center'
                                    style={{ height: `${tile}px`, backgroundImage }}
                                />
                                <div className='pointer-events-none absolute inset-0 bg-black/0 transition group-hover:bg-black/20' />
                                <button
                                    onClick={() => handleDeleteImage(img)}
                                    className='absolute right-2 top-2 hidden rounded bg-white/90 p-1 text-red-600 shadow group-hover:block'
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
                                <TableCell>
                                    {img.width && img.height ? `${img.width}×${img.height}` : '—'}
                                </TableCell>
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

    return (
        <div className='relative mx-auto'>
            <div className='absolute inset-x-0 top-0 -z-10 h-80 overflow-hidden rounded-t-2xl mask-b-from-60% sm:h-88 md:h-112 lg:h-128'>
                {album.banner_image_path && (
                    <img
                        alt=''
                        src={getBannerUrl(album.banner_image_path)}
                        className='absolute inset-0 h-full w-full mask-l-from-60% object-cover object-center opacity-40'
                    />
                )}
                <div className='absolute inset-0 rounded-t-2xl outline-1 -outline-offset-1 outline-gray-950/10 dark:outline-white/10' />
            </div>
            <div className='mx-auto'>
                <div className='relative'>
                    <div className='px-8 pt-48 pb-12 lg:py-24'>
                        {/*<Logo className="h-8 fill-gray-950 dark:fill-white" />*/}
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

                    <div className='mt-4'>
                        <input
                            ref={fileInputRef}
                            type='file'
                            className='hidden'
                            multiple
    // @ts-ignore - nonstandard directory selection for Chromium-based browsers
                            webkitdirectory=''
                            onChange={handleFolderChange}
                        />
                        <Button onClick={() => fileInputRef.current?.click()} disabled={isUploading}>
                            {isUploading ? 'Uploading…' : 'Upload Folder'}
                        </Button>
                    </div>
                    <div className='mt-4 max-h-80 overflow-auto rounded border p-3 text-xs'>
                        {Object.values(items).length === 0 && (
                            <div className='text-gray-500'>No active uploads or processing.</div>
                        )}
                        {Object.values(items)
                            .sort((a, b) => {
                                const totalTasks = 3;
                                const aDone = Object.values(a.tasks).filter((s) => s === 'done').length === totalTasks;
                                const bDone = Object.values(b.tasks).filter((s) => s === 'done').length === totalTasks;
                                if (aDone !== bDone) return Number(aDone) - Number(bDone); // incomplete first, completed last
                                return a.path.localeCompare(b.path);
                            })
                            .map((it) => {
                                const totalTasks = 3; // thumbnail, metadata, detection
                                const doneCount = Object.values(it.tasks).filter((s) => s === 'done').length;
                                const hadError = Boolean(it.error) || Object.values(it.tasks).includes('error');
                                let pct = 0;
                                if (it.uploading) pct = 10;
                                if (it.uploaded) pct = 30;
                                pct = Math.max(pct, 30 + Math.round((doneCount / totalTasks) * 70));
                                if (doneCount === totalTasks) pct = 100;

                                const labelMap: Record<string, string> = {
                                    upload: 'Uploading',
                                    thumbnail: 'Generating thumbnail',
                                    metadata: 'Extracting metadata',
                                    detection: 'Detecting faces',
                                };
                                let currentLabel = '';
                                if (it.currentTask) {
                                    currentLabel = labelMap[it.currentTask] || it.currentTask;
                                } else if (doneCount === totalTasks) {
                                    currentLabel = 'Completed';
                                } else if (it.uploaded) {
                                    currentLabel = 'Queued for processing';
                                }
                                return (
                                    <div key={it.path} className='mb-3 last:mb-0'>
                                        <div className='mb-1 flex items-center justify-between gap-2'>
                                            <div className='truncate font-mono'>
                                                {it.path.replace(album.folder_path + '/', '')}
                                            </div>
                                            <div
                                                className={`text-[10px] ${hadError ? 'text-red-600' : 'text-gray-500'}`}
                                            >
                                                {hadError ? 'Error' : currentLabel}
                                            </div>
                                        </div>
                                        <div className='h-2 w-full overflow-hidden rounded bg-gray-200'>
                                            <div
                                                className={`h-full ${hadError ? 'bg-red-500' : 'bg-blue-500'}`}
                                                style={{ width: `${pct}%` }}
                                            />
                                        </div>
                                    </div>
                                );
                            })}
                    </div>

                    <div className='mt-8 rounded-lg bg-white shadow'>
                        <div className='border-b border-gray-200 px-6 py-4'>
                            <h2 className='text-lg font-medium text-gray-900'>Photos</h2>
                        </div>
                        <div className='px-6 py-4 space-y-4'>
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
