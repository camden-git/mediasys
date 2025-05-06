import React, { useMemo, useState } from 'react';
import { useStoreState, State } from 'easy-peasy';
import { StoreModel } from '../../store';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import ErrorMessage from '../elements/ErrorMessage.tsx';
import { Heading } from '../elements/Heading.tsx';
import { Text } from '../elements/Text.tsx';
import { Button } from '../elements/Button.tsx';
import { FolderArrowDownIcon } from '@heroicons/react/24/outline';
import AdvancedImageGrid from './AdvancedImageGrid.tsx';
import { getAlbumDownloadUrl, getBannerUrl } from '../../api.ts';
import { FileInfo } from '../../types.ts';
import ImageLightbox from './ImageLightbox.tsx';
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../elements/Dialog.tsx';
import { bytesToString } from '../../lib/formatters.ts';

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
            <div>
                <div className={'bg-gray-200'}>
                    {currentAlbum?.banner_image_path && (
                        <img
                            className='h-32 w-full rounded-t-lg object-cover lg:h-80'
                            src={getBannerUrl(currentAlbum?.banner_image_path)}
                            alt={`${currentAlbum?.name} banner image`}
                        />
                    )}
                </div>
                <div className='mb-8 px-4 sm:px-6 lg:px-8'>
                    <div className='-mt-30 sm:flex sm:items-end sm:space-x-5'>
                        <div className='mt-6 sm:flex sm:min-w-0 sm:flex-1 sm:items-center sm:justify-end sm:space-x-6 sm:pb-1'>
                            <div className='mt-6 min-w-0 flex-1 sm:hidden md:block'>
                                <Heading className={'truncate font-bold !text-gray-100'} huge>
                                    {currentAlbum?.name}
                                </Heading>
                                <Text className={'truncate !text-zinc-200'}>{currentAlbum?.description}</Text>
                            </div>
                            <div className='mt-6 flex flex-col justify-stretch space-y-3 sm:flex-row sm:space-y-0 sm:space-x-4'>
                                {/*<Button color={'white'}>*/}
                                {/*    Share <ArrowTopRightOnSquareIcon />*/}
                                {/*</Button>*/}
                                {currentAlbum?.zip_size && (
                                    <>
                                        <Button color={'white'} onClick={() => setDownloadModalOpen(true)}>
                                            Download <FolderArrowDownIcon />
                                        </Button>
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
                    </div>
                    <div className='-mt-6 hidden min-w-0 flex-1 sm:block md:hidden'>
                        <Heading className={'truncate font-bold !text-gray-100'} huge>
                            {currentAlbum?.name}
                        </Heading>
                    </div>
                </div>
            </div>
            <div className='p-4'>
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
        </>
    );
};

export default AlbumView;
