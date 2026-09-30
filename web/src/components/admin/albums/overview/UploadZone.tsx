import React from 'react';
import { PhotoIcon } from '@heroicons/react/24/outline';
import { Button } from '../../../elements/Button.tsx';

export interface UploadZoneProps {
    disabled?: boolean;
    onFiles: (files: Array<{ file: File; relativePath: string }>) => void;
}

async function traverseEntry(
    entry: FileSystemEntry,
    pathPrefix: string,
    out: Array<{ file: File; relativePath: string }>,
): Promise<void> {
    if (entry.isFile) {
        const fileEntry = entry as FileSystemFileEntry;
        await new Promise<void>((resolve, reject) => {
            fileEntry.file((f) => {
                out.push({ file: f, relativePath: pathPrefix + f.name });
                resolve();
            }, reject);
        });
    } else if (entry.isDirectory) {
        const dirEntry = entry as FileSystemDirectoryEntry;
        const reader = dirEntry.createReader();
        // readEntries returns results in batches (~100 in Chrome); keep reading until an empty batch
        const entries: FileSystemEntry[] = [];
        for (;;) {
            const batch = await new Promise<FileSystemEntry[]>((resolve, reject) => {
                reader.readEntries((results) => resolve(Array.from(results)), reject);
            });
            if (batch.length === 0) break;
            entries.push(...batch);
        }
        for (const child of entries) {
            await traverseEntry(child, pathPrefix + entry.name + '/', out);
        }
    }
}

const UploadZone: React.FC<UploadZoneProps> = ({ disabled, onFiles }) => {
    const [isDragOver, setIsDragOver] = React.useState(false);
    const fileInputRef = React.useRef<HTMLInputElement>(null);
    const folderInputRef = React.useRef<HTMLInputElement>(null);

    // dragenter/dragleave fire for every child element crossed, so count them to avoid flicker
    const dragDepth = React.useRef(0);

    const handleDragEnter = (e: React.DragEvent) => {
        e.preventDefault();
        dragDepth.current += 1;
        if (!disabled) setIsDragOver(true);
    };

    const handleDragOver = (e: React.DragEvent) => {
        e.preventDefault();
        if (!disabled) setIsDragOver(true);
    };

    const handleDragLeave = (e: React.DragEvent) => {
        e.preventDefault();
        dragDepth.current = Math.max(0, dragDepth.current - 1);
        if (dragDepth.current === 0) setIsDragOver(false);
    };

    const handleDrop = async (e: React.DragEvent) => {
        e.preventDefault();
        dragDepth.current = 0;
        setIsDragOver(false);
        if (disabled) return;

        const collected: Array<{ file: File; relativePath: string }> = [];

        // DataTransfer items are invalidated once the handler yields, so grab everything synchronously first
        const entries: FileSystemEntry[] = [];
        for (const item of Array.from(e.dataTransfer.items)) {
            if (item.kind !== 'file') continue;
            const entry = item.webkitGetAsEntry?.();
            if (entry) {
                entries.push(entry);
            } else {
                const f = item.getAsFile();
                if (f) collected.push({ file: f, relativePath: f.name });
            }
        }

        try {
            for (const entry of entries) {
                await traverseEntry(entry, '', collected);
            }
        } catch (err) {
            console.error('Failed to read dropped files', err);
        }

        if (collected.length > 0) onFiles(collected);
    };

    const handleFileInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const files = e.target.files;
        if (!files || files.length === 0) return;
        const collected = Array.from(files).map((f) => ({
            file: f,
            relativePath: (f as any).webkitRelativePath || f.name,
        }));
        onFiles(collected);
        e.target.value = '';
    };

    return (
        <div
            onDragEnter={handleDragEnter}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            className={`flex flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed px-6 py-10 text-center transition-colors ${
                disabled
                    ? 'cursor-not-allowed border-gray-200 bg-gray-50 opacity-60 dark:border-zinc-700 dark:bg-zinc-900'
                    : isDragOver
                      ? 'border-blue-500 bg-blue-50 dark:border-blue-400 dark:bg-blue-500/10'
                      : 'border-gray-300 bg-white hover:border-gray-400 dark:border-zinc-700 dark:bg-zinc-900 dark:hover:border-zinc-500'
            }`}
        >
            <PhotoIcon className={`h-10 w-10 ${isDragOver ? 'text-blue-400' : 'text-gray-300 dark:text-zinc-600'}`} />
            <div className='text-sm text-gray-600 dark:text-zinc-300'>
                <span className='font-medium'>Drag &amp; drop files or folders here</span>
                <br />
                <span className='text-gray-400 dark:text-zinc-500'>or use the buttons below</span>
            </div>
            <div className='flex gap-2'>
                <Button color={'dark/zinc'} disabled={disabled} onClick={() => fileInputRef.current?.click()}>
                    Select Files
                </Button>
                <Button outline disabled={disabled} onClick={() => folderInputRef.current?.click()}>
                    Select Folder
                </Button>
            </div>

            {/* Individual files */}
            <input ref={fileInputRef} type='file' multiple className='hidden' onChange={handleFileInputChange} />
            {/* Folder selection */}
            <input
                ref={folderInputRef}
                type='file'
                multiple
                className='hidden'
                // @ts-ignore - nonstandard directory selection for Chromium-based browsers
                webkitdirectory=''
                onChange={handleFileInputChange}
            />
        </div>
    );
};

export default UploadZone;
