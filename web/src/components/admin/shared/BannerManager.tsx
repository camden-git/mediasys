import React, { useRef } from 'react';
import { PhotoIcon } from '@heroicons/react/24/outline';
import { ChevronDownIcon, ChevronUpIcon, TrashIcon } from '@heroicons/react/24/solid';
import { Button } from '../../elements/Button';
import { getBannerUrl } from '../../../api/media';

const ALLOWED_TYPES = ['image/png', 'image/jpeg', 'image/webp'];

export interface BannerItem {
    id: number;
    image_path: string;
    sort_order: number;
}

export interface BannerManagerProps {
    banners: BannerItem[];
    onAdd: (file: File) => Promise<void>;
    onDelete: (id: number) => Promise<void>;
    onReorder: (ids: number[]) => Promise<void>;
}

export const BannerManager: React.FC<BannerManagerProps> = ({ banners, onAdd, onDelete, onReorder }) => {
    const [isDragOver, setIsDragOver] = React.useState(false);
    const [isUploading, setIsUploading] = React.useState(false);
    const inputRef = useRef<HTMLInputElement>(null);

    const processFile = async (file: File) => {
        if (!ALLOWED_TYPES.includes(file.type)) return;
        setIsUploading(true);
        try {
            await onAdd(file);
        } finally {
            setIsUploading(false);
        }
    };

    const handleDragOver = (e: React.DragEvent) => {
        e.preventDefault();
        if (!isUploading) setIsDragOver(true);
    };

    const handleDragLeave = (e: React.DragEvent) => {
        e.preventDefault();
        // ignore leaves into child elements
        if (e.currentTarget.contains(e.relatedTarget as Node | null)) return;
        setIsDragOver(false);
    };

    const handleDrop = async (e: React.DragEvent) => {
        e.preventDefault();
        setIsDragOver(false);
        if (isUploading) return;
        for (const item of Array.from(e.dataTransfer.items)) {
            if (item.kind !== 'file') continue;
            const file = item.getAsFile();
            if (file && ALLOWED_TYPES.includes(file.type)) {
                await processFile(file);
                return;
            }
        }
    };

    const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        e.target.value = '';
        if (!file) return;
        await processFile(file);
    };

    const handleMove = async (index: number, direction: -1 | 1) => {
        const newOrder = [...banners];
        const swapIndex = index + direction;
        if (swapIndex < 0 || swapIndex >= newOrder.length) return;
        [newOrder[index], newOrder[swapIndex]] = [newOrder[swapIndex], newOrder[index]];
        await onReorder(newOrder.map((b) => b.id));
    };

    return (
        <div className='space-y-3'>
            {banners.length > 0 && (
                <ul className='space-y-2'>
                    {banners.map((banner, idx) => (
                        <li
                            key={banner.id}
                            className='flex items-center gap-3 rounded-lg border border-zinc-200 bg-zinc-50 p-2 dark:border-zinc-700 dark:bg-zinc-800'
                        >
                            <div className='relative shrink-0'>
                                <img
                                    src={getBannerUrl(banner.image_path)}
                                    alt=''
                                    className='h-20 w-32 rounded object-cover object-center'
                                    loading='lazy'
                                    decoding='async'
                                />
                                <span className='absolute top-1 left-1 flex h-5 w-5 items-center justify-center rounded-full bg-black/50 text-[10px] font-medium text-white'>
                                    {idx + 1}
                                </span>
                            </div>
                            <div className='flex flex-1 items-center justify-end gap-1'>
                                <Button
                                    plain
                                    type='button'
                                    onClick={() => handleMove(idx, -1)}
                                    disabled={idx === 0 || isUploading}
                                    title='Move up'
                                    aria-label={`Move banner ${idx + 1} up`}
                                >
                                    <ChevronUpIcon className='size-4' />
                                </Button>
                                <Button
                                    plain
                                    type='button'
                                    onClick={() => handleMove(idx, 1)}
                                    disabled={idx === banners.length - 1 || isUploading}
                                    title='Move down'
                                    aria-label={`Move banner ${idx + 1} down`}
                                >
                                    <ChevronDownIcon className='size-4' />
                                </Button>
                                <Button
                                    plain
                                    type='button'
                                    onClick={() => onDelete(banner.id)}
                                    disabled={isUploading}
                                    title='Remove banner'
                                    aria-label={`Remove banner ${idx + 1}`}
                                >
                                    <TrashIcon className='size-4 text-red-500' />
                                </Button>
                            </div>
                        </li>
                    ))}
                </ul>
            )}

            <div
                onDragOver={handleDragOver}
                onDragLeave={handleDragLeave}
                onDrop={handleDrop}
                className={`flex flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed px-6 py-8 text-center transition-colors ${
                    isUploading
                        ? 'cursor-not-allowed border-gray-200 bg-gray-50 opacity-60'
                        : isDragOver
                          ? 'border-blue-500 bg-blue-50'
                          : 'border-gray-300 bg-white hover:border-gray-400'
                }`}
            >
                <PhotoIcon className={`h-10 w-10 ${isDragOver ? 'text-blue-400' : 'text-gray-300'}`} />
                <div className='text-sm text-gray-600'>
                    <span className='font-medium'>{isUploading ? 'Uploading…' : 'Drag & drop a banner image'}</span>
                    {!isUploading && (
                        <>
                            <br />
                            <span className='text-gray-400'>PNG, JPEG, or WebP</span>
                        </>
                    )}
                </div>
                {!isUploading && (
                    <Button color='dark/zinc' disabled={isUploading} onClick={() => inputRef.current?.click()}>
                        Select Image
                    </Button>
                )}
            </div>

            <input
                ref={inputRef}
                type='file'
                className='hidden'
                aria-hidden
                accept={ALLOWED_TYPES.join(',')}
                onChange={handleFileChange}
            />
        </div>
    );
};
