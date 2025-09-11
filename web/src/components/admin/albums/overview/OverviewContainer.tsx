import React from 'react';
import { useStoreState } from '../../../../store/hooks.ts';
import { getBannerUrl } from '../../../../api.ts';
import { Heading } from '../../../elements/Heading.tsx';
import { CameraIcon, MapPinIcon, PhotoIcon } from '@heroicons/react/16/solid';
import { Button } from '../../../elements/Button';
import { uploadAlbumImages } from '../../../../api/admin/albums';

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
    const apiUrl = (import.meta as any).env.VITE_API_URL as string;
    const authToken = localStorage.getItem('authToken');

    React.useEffect(() => {
        try {
            const httpUrl = new URL(apiUrl);
            const wsProtocol = httpUrl.protocol === 'https:' ? 'wss:' : 'ws:';
            const tokenQuery = authToken ? `?token=${encodeURIComponent(authToken)}` : '';
            const wsUrl = `${wsProtocol}//${httpUrl.host}/api/ws${tokenQuery}`;
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
                } catch {}
            };
            return () => ws.close();
        } catch {
            // ignore bad URL
        }
    }, [apiUrl]);

    const handleFolderChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const files = e.target.files;
        if (!files || files.length === 0) return;
        setIsUploading(true);
        try {
            const items = Array.from(files).map((f) => ({ file: f, relativePath: (f as any).webkitRelativePath || f.name }));
            await uploadAlbumImages(album.id, items);
        } finally {
            setIsUploading(false);
            if (fileInputRef.current) fileInputRef.current.value = '';
        }
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
                                    {album.artists.map((u) =>
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
                                const currentLabel = it.currentTask ? labelMap[it.currentTask] || it.currentTask : doneCount === totalTasks ? 'Completed' : it.uploaded ? 'Queued for processing' : '';
                                return (
                                    <div key={it.path} className='mb-3 last:mb-0'>
                                        <div className='mb-1 flex items-center justify-between gap-2'>
                                            <div className='truncate font-mono'>{it.path.replace(album.folder_path + '/', '')}</div>
                                            <div className={`text-[10px] ${hadError ? 'text-red-600' : 'text-gray-500'}`}>{hadError ? 'Error' : currentLabel}</div>
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
                </div>
            </div>
        </div>
    );
};

export default OverviewContainer;
