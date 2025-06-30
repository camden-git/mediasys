import React, { useEffect, useRef, useState } from 'react';
import { useStoreActions, useStoreState } from '../../../../store/hooks';
import { Button } from '../../../elements/Button';
import { PhotoIcon } from '@heroicons/react/24/solid';
import HeaderedContent from '../../../elements/HeaderedContent.tsx';

const allowedTypes: readonly string[] = ['image/png', 'image/jpeg', 'image/webp'] as const;

export function BannerUpload() {
    const albumId = useStoreState((state) => state.albumContext.data!.id);
    const { uploadAlbumBanner } = useStoreActions((actions) => actions.adminAlbums);
    const { addFlash } = useStoreActions((actions) => actions.ui);
    const { setAlbum } = useStoreActions((actions) => actions.albumContext);

    const [bannerFile, setBannerFile] = useState<File | null>(null);
    const [bannerBlobUrl, setBannerBlobUrl] = useState<string | null>(null);
    const [bannerUploading, setBannerUploading] = useState(false);
    const [bannerError, setBannerError] = useState('');
    const bannerInputRef = useRef<HTMLInputElement>(null);

    const handleBannerDrop: React.DragEventHandler<HTMLButtonElement> = (event) => {
        event.preventDefault();
        if (event.dataTransfer.files.length > 0) {
            const file = event.dataTransfer.files[0];
            if (!allowedTypes.includes(file.type)) {
                setBannerError('Invalid file type. Please select a PNG, JPEG, or WebP file.');
                return;
            }
            setBannerFile(file);
            setBannerError('');
        }
    };

    const handleBannerFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        if (event.target.files && event.target.files.length > 0) {
            const file = event.target.files[0];
            if (!allowedTypes.includes(file.type)) {
                setBannerError('Invalid file type. Please select a PNG, JPEG, or WebP file.');
                return;
            }
            setBannerFile(file);
            setBannerError('');
        }
    };

    const handleBannerUpload = () => {
        if (!bannerFile) return;
        setBannerUploading(true);
        setBannerError('');

        uploadAlbumBanner({
            id: albumId,
            file: bannerFile,
            addFlash,
            setAlbum,
            onSuccess: () => {
                setBannerFile(null);
                setBannerBlobUrl(null);
                setBannerUploading(false);
            },
        });
    };

    // cleanup blob URL when component unmounts or file changes
    useEffect(() => {
        if (bannerBlobUrl) {
            URL.revokeObjectURL(bannerBlobUrl);
        }
        if (!bannerFile) {
            setBannerBlobUrl(null);
            return;
        }
        setBannerBlobUrl(URL.createObjectURL(bannerFile));
    }, [bannerFile]);

    return (
        <HeaderedContent
            title={'Album Banner'}
            description={'Upload a banner image for this album.'}
            className={'mt-8 mb-4'}
        >
            {bannerError && (
                <div className='mb-4 rounded-lg border border-red-200 bg-red-50 p-3'>
                    <p className='sm text-red-700'>{bannerError}</p>
                </div>
            )}

            <input
                type='file'
                className='hidden'
                aria-hidden
                ref={bannerInputRef}
                accept={allowedTypes.join(',')}
                onChange={handleBannerFileChange}
            />

            <button
                className='flex w-full flex-col items-center justify-center rounded-xl border-2 border-dashed border-gray-600 p-8 transition-opacity duration-100 disabled:opacity-50'
                onDragOver={(e) => e.preventDefault()}
                onDrop={handleBannerDrop}
                onClick={() => bannerInputRef.current?.click()}
                disabled={bannerUploading}
            >
                {bannerBlobUrl ? (
                    <img src={bannerBlobUrl} className='h-48 max-w-full rounded-lg' alt='Banner preview' />
                ) : (
                    <>
                        <PhotoIcon className='size-16 text-gray-500' />
                        <p className='text-gray-500'>Drag and drop a banner image here</p>
                        <p className='xs text-gray-500'>or click to select a file</p>
                        <p className='mt-2 text-xs text-gray-500'>Supported formats: .jpg, .png, .webp</p>
                    </>
                )}
            </button>

            <div className='mt-4 flex justify-end space-x-3'>
                <Button
                    type='button'
                    outline
                    onClick={() => {
                        setBannerFile(null);
                        setBannerError('');
                    }}
                    disabled={bannerUploading || !bannerFile}
                >
                    Clear
                </Button>
                <Button onClick={handleBannerUpload} disabled={bannerUploading || !bannerFile}>
                    {bannerUploading ? 'Uploading...' : 'Upload Banner'}
                </Button>
            </div>
        </HeaderedContent>
    );
}
