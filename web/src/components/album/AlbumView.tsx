import React, { useMemo, useState } from 'react';
import { useStoreState, State } from 'easy-peasy';
import { StoreModel } from '../../store';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import ErrorMessage from '../elements/ErrorMessage.tsx';
import { Heading } from '../elements/Heading.tsx';
import { Button } from '../elements/Button.tsx';
import AdvancedImageGrid from './AdvancedImageGrid.tsx';
import { getAlbumDownloadUrl, getBannerUrl } from '../../api.ts';
import { FileInfo } from '../../types.ts';
import ImageLightbox from './ImageLightbox.tsx';
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../elements/Dialog.tsx';
import { bytesToString } from '../../lib/formatters.ts';
import { ArrowDownIcon, CameraIcon, MapPinIcon, PhotoIcon } from '@heroicons/react/16/solid';

const AlbumView: React.FC = () => {
    const { currentAlbum, directoryListing, isLoading, error } = useStoreState(
        (state: State<StoreModel>) => state.contentView,
    );

    const [selectedImage, setSelectedImage] = useState<FileInfo | null>(null);
    const [downloadModalOpen, setDownloadModalOpen] = useState(false);

    const imageFiles = useMemo(() => {
        if (!directoryListing?.files) {
            return [];
        }
        return directoryListing.files.filter((file) => !file.is_dir && file.thumbnail_path);
    }, [directoryListing]);

    const handleImageClick = (image: FileInfo) => {
        setSelectedImage(image);
    };

    const handleCloseLightbox = () => {
        setSelectedImage(null);
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

    return (
        <>
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
                            {/*<Logo className="h-8 fill-gray-950 dark:fill-white" />*/}
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
                                    {directoryListing?.files.length} photos
                                </div>
                                <span className='hidden text-gray-950/25 sm:inline dark:text-white/25'>&middot;</span>
                                <div className='flex items-center gap-1.5'>
                                    <CameraIcon className='size-4 text-gray-950/40' />
                                    Camden Rush
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
                            <div className='mt-10'>
                                {currentAlbum?.zip_size && (
                                    <>
                                        <button
                                            onClick={() => setDownloadModalOpen(true)}
                                            className='inline-flex items-center gap-x-2 rounded-full bg-gray-950 px-3 py-0.5 text-sm/7 font-semibold text-white hover:bg-gray-800 dark:bg-gray-700 dark:hover:bg-gray-600'
                                        >
                                            <ArrowDownIcon className='size-2 fill-white' />
                                            Download
                                        </button>
                                        <Dialog open={downloadModalOpen} onClose={setDownloadModalOpen}>
                                            <DialogTitle>Download {currentAlbum?.name}</DialogTitle>
                                            <DialogDescription>
                                                A {bytesToString(currentAlbum.zip_size)} zip file containing all images
                                                in this album is available for download. This download may take a long
                                                time as the images are in the highest quality. Individual photos can be
                                                downloaded by opening an image and pressing the download icon in the top
                                                right.
                                            </DialogDescription>
                                            <DialogBody></DialogBody>
                                            <DialogActions>
                                                <Button plain onClick={() => setDownloadModalOpen(false)}>
                                                    Cancel
                                                </Button>
                                                <Button onClick={handleDownloadZip}>Download</Button>
                                            </DialogActions>
                                        </Dialog>
                                    </>
                                )}
                            </div>
                        </div>

                        <div className='mt-4'>
                            <ErrorMessage message={error} />

                            {isLoading && <LoadingSpinner />}

                            {!isLoading && !error && directoryListing && (
                                <AdvancedImageGrid
                                    images={imageFiles}
                                    targetRowHeight={260}
                                    boxSpacing={4}
                                    onImageClick={handleImageClick}
                                />
                            )}
                            <ImageLightbox image={selectedImage} onClose={handleCloseLightbox} />
                        </div>
                    </div>
                </div>
            </div>
        </>
    );
};

export default AlbumView;
