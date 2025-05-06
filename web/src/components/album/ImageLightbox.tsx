import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { FileInfo } from '../../types.ts';
import { getOriginalImageUrl } from '../../api.ts';
import MetadataPanel from './MetadataPanel';
import { DocumentArrowDownIcon, InformationCircleIcon, XMarkIcon } from '@heroicons/react/24/outline';

interface ImageLightboxProps {
    image: FileInfo | null;
    onClose: () => void;
}

const PANEL_WIDTH_NUMERIC = 384;

const transitionSettings = {
    type: 'tween',
    duration: 0.3,
    ease: 'easeInOut',
};

const ImageLightbox: React.FC<ImageLightboxProps> = ({ image, onClose }) => {
    const [isPanelOpen, setIsPanelOpen] = useState(false);
    const [isImageLoaded, setIsImageLoaded] = useState(false);

    useEffect(() => {
        if (!image) setIsPanelOpen(false);
    }, [image]);
    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') onClose();
        };
        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [onClose]);

    const handleBackdropClick = (event: React.MouseEvent<HTMLDivElement>) => {
        if (event.target === event.currentTarget) onClose();
    };

    const togglePanel = () => setIsPanelOpen((prev) => !prev);

    const imageUrl = image ? getOriginalImageUrl(image.path) : '';

    const handleDownloadImage = async (path: string) => {
        try {
            const response = await fetch(path, { mode: 'cors' });
            const blob = await response.blob();

            const url = window.URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;
            link.setAttribute('download', image?.name || 'download.jpg');
            document.body.appendChild(link);
            link.click();
            link.remove();
            window.URL.revokeObjectURL(url);
        } catch (error) {
            console.error('Error downloading image:', error);
        }
    };

    return (
        <AnimatePresence>
            {image && (
                <motion.div
                    key='lightbox-backdrop'
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    className='fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/25 backdrop-blur-xl'
                    onClick={handleBackdropClick}
                    aria-modal='true'
                    role='dialog'
                >
                    <motion.div
                        className='relative flex h-full w-full items-stretch justify-center'
                        layout
                        transition={transitionSettings}
                    >
                        {/* Image Container */}
                        <motion.div
                            className='relative flex flex-1 items-center justify-center overflow-hidden'
                            layout
                            transition={transitionSettings}
                        >
                            <div className='absolute top-3 right-3 z-60 flex space-x-3'>
                                <button
                                    onClick={() => handleDownloadImage(imageUrl)}
                                    className='rounded-full bg-gray-700/60 p-2 text-white transition-colors hover:bg-gray-600/80 focus:ring-2 focus:ring-white focus:outline-none'
                                    aria-label={'Download Image'}
                                >
                                    <DocumentArrowDownIcon className='h-6 w-6' />
                                </button>
                                <button
                                    onClick={togglePanel}
                                    className='rounded-full bg-gray-700/60 p-2 text-white transition-colors hover:bg-gray-600/80 focus:ring-2 focus:ring-white focus:outline-none'
                                    aria-label={isPanelOpen ? 'Hide image information' : 'Show image information'}
                                >
                                    <InformationCircleIcon className='h-6 w-6' />
                                </button>
                                <button
                                    onClick={onClose}
                                    className='rounded-full bg-gray-700/60 p-2 text-white transition-colors hover:bg-gray-600/80 focus:ring-2 focus:ring-white focus:outline-none'
                                    aria-label='Close image view'
                                >
                                    <XMarkIcon className='h-6 w-6' />
                                </button>
                            </div>
                            <div className='relative max-h-full max-w-full'>
                                {!isImageLoaded && image?.width && image?.height && (
                                    <div
                                        style={{ width: image.width, height: image.height }}
                                        className='animate-pulse rounded bg-gray-800/40 p-4 shadow-inner'
                                    />
                                )}
                                <motion.img
                                    key={imageUrl}
                                    initial={{ scale: 0.8, opacity: 0 }}
                                    animate={{ scale: isImageLoaded ? 1 : 0.95, opacity: isImageLoaded ? 1 : 0 }}
                                    exit={{ scale: 0.8, opacity: 0 }}
                                    transition={transitionSettings}
                                    onLoad={() => setIsImageLoaded(true)}
                                    src={imageUrl}
                                    alt={image.name}
                                    className='block max-h-screen max-w-full object-contain p-4'
                                />
                            </div>
                        </motion.div>

                        {/* Metadata Panel */}
                        <AnimatePresence>
                            {isPanelOpen && (
                                <motion.div
                                    key='metadata-panel-wrapper'
                                    initial={{ width: 0 }}
                                    animate={{ width: PANEL_WIDTH_NUMERIC }}
                                    exit={{ width: 0 }}
                                    transition={transitionSettings}
                                    className='h-full overflow-hidden'
                                >
                                    <MetadataPanel isOpen={true} onClose={togglePanel} image={image} />
                                </motion.div>
                            )}
                        </AnimatePresence>
                    </motion.div>
                </motion.div>
            )}
        </AnimatePresence>
    );
};

export default ImageLightbox;
