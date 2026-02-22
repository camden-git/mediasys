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
        await new Promise<void>((resolve) => {
            fileEntry.file((f) => {
                out.push({ file: f, relativePath: pathPrefix + f.name });
                resolve();
            });
        });
    } else if (entry.isDirectory) {
        const dirEntry = entry as FileSystemDirectoryEntry;
        const reader = dirEntry.createReader();
        const entries = await new Promise<FileSystemEntry[]>((resolve) => {
            reader.readEntries((results) => resolve(Array.from(results)));
        });
        for (const child of entries) {
            await traverseEntry(child, pathPrefix + entry.name + '/', out);
        }
    }
}

const UploadZone: React.FC<UploadZoneProps> = ({ disabled, onFiles }) => {
    const [isDragOver, setIsDragOver] = React.useState(false);
    const fileInputRef = React.useRef<HTMLInputElement>(null);
    const folderInputRef = React.useRef<HTMLInputElement>(null);

    const handleDragOver = (e: React.DragEvent) => {
        e.preventDefault();
        if (!disabled) setIsDragOver(true);
    };

    const handleDragLeave = (e: React.DragEvent) => {
        e.preventDefault();
        setIsDragOver(false);
    };

    const handleDrop = async (e: React.DragEvent) => {
        e.preventDefault();
        setIsDragOver(false);
        if (disabled) return;

        const collected: Array<{ file: File; relativePath: string }> = [];
        const items = Array.from(e.dataTransfer.items);

        for (const item of items) {
            if (item.kind !== 'file') continue;
            const entry = item.webkitGetAsEntry?.();
            if (entry) {
                await traverseEntry(entry, '', collected);
            } else {
                const f = item.getAsFile();
                if (f) collected.push({ file: f, relativePath: f.name });
            }
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
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            className={`flex flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed px-6 py-10 text-center transition-colors ${
                disabled
                    ? 'cursor-not-allowed border-gray-200 bg-gray-50 opacity-60'
                    : isDragOver
                      ? 'border-blue-500 bg-blue-50'
                      : 'border-gray-300 bg-white hover:border-gray-400'
            }`}
        >
            <PhotoIcon className={`h-10 w-10 ${isDragOver ? 'text-blue-400' : 'text-gray-300'}`} />
            <div className='text-sm text-gray-600'>
                <span className='font-medium'>Drag &amp; drop files or folders here</span>
                <br />
                <span className='text-gray-400'>or use the buttons below</span>
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
