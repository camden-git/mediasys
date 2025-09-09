import React, { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { FileInfo } from '../../types.ts';
import { getOriginalImageUrl } from '../../api.ts';
import MetadataPanel from './MetadataPanel';
import {
    DocumentArrowDownIcon,
    InformationCircleIcon,
    XMarkIcon,
    ShareIcon,
    ChevronLeftIcon,
    ChevronRightIcon,
} from '@heroicons/react/24/outline';

interface ImageLightboxProps {
    image: FileInfo | null;
    onClose: () => void;
    onPrev?: () => void;
    onNext?: () => void;
    canPrev?: boolean;
    canNext?: boolean;
}

const PANEL_WIDTH_NUMERIC = 384;

const transitionSettings = {
    type: 'tween',
    duration: 0.3,
    ease: 'easeInOut',
};

interface IconButtonProps {
    label: string;
    onClick: () => void;
    children: React.ReactNode;
    disabled?: boolean;
}

const IconButton: React.FC<IconButtonProps> = ({ label, onClick, children, disabled }) => (
    <button
        onClick={onClick}
        disabled={disabled}
        className='rounded-full bg-gray-700/60 p-2 text-white transition-colors hover:bg-gray-600/80 focus:ring-2 focus:ring-white focus:outline-none disabled:cursor-not-allowed disabled:opacity-40'
        aria-label={label}
    >
        {children}
    </button>
);

const ImageLightbox: React.FC<ImageLightboxProps> = ({
    image,
    onClose,
    onPrev,
    onNext,
    canPrev = false,
    canNext = false,
}) => {
    const [isPanelOpen, setIsPanelOpen] = useState(false);
    const [isImageLoaded, setIsImageLoaded] = useState(false);

    const previousImageRef = useRef<FileInfo | null>(null);
    const [shouldZoomOnImage, setShouldZoomOnImage] = useState(false);

    useEffect(() => {
        const wasImageNull = previousImageRef.current === null;
        const isImageNow = image !== null;
        setShouldZoomOnImage(wasImageNull && isImageNow);
        previousImageRef.current = image;
        if (!image) setIsPanelOpen(false);
        setIsImageLoaded(false);
    }, [image]);
    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                onClose();
                return;
            }
            if (event.key === 'ArrowLeft') {
                if (canPrev && onPrev) {
                    event.preventDefault();
                    onPrev();
                }
                return;
            }
            if (event.key === 'ArrowRight') {
                if (canNext && onNext) {
                    event.preventDefault();
                    onNext();
                }
                return;
            }
        };
        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [onClose, onPrev, onNext, canPrev, canNext]);

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

    const handleShareImage = async (path: string) => {
        try {
            if (!('share' in navigator)) {
                await handleDownloadImage(path);
                return;
            }
            const response = await fetch(path, { mode: 'cors' });
            const blob = await response.blob();
            const fileName = image?.name || 'photo.jpg';
            const file = new File([blob], fileName, { type: blob.type || 'image/jpeg' });
            await navigator.share({
                files: [file],
                title: fileName,
                text: image?.name ? `Check out this photo: ${image.name}` : 'Check out this photo',
            });
        } catch (error) {
            console.error('Error sharing image:', error);
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
                        <motion.div
                            className='relative flex flex-1 items-center justify-center overflow-hidden'
                            layout
                            transition={transitionSettings}
                        >
                            <div className='pointer-events-none absolute inset-y-0 right-0 left-0 z-60 flex items-center justify-between px-3'>
                                <div className='pointer-events-auto'>
                                    <IconButton
                                        label='Previous image'
                                        onClick={() => onPrev && onPrev()}
                                        disabled={!canPrev}
                                    >
                                        <ChevronLeftIcon className='h-7 w-7' />
                                    </IconButton>
                                </div>
                                <div className='pointer-events-auto'>
                                    <IconButton
                                        label='Next image'
                                        onClick={() => onNext && onNext()}
                                        disabled={!canNext}
                                    >
                                        <ChevronRightIcon className='h-7 w-7' />
                                    </IconButton>
                                </div>
                            </div>
                            <div className='absolute top-3 right-3 z-60 flex space-x-3'>
                                <IconButton label='Download Image' onClick={() => handleDownloadImage(imageUrl)}>
                                    <DocumentArrowDownIcon className='h-6 w-6' />
                                </IconButton>
                                <IconButton label='Share Image' onClick={() => handleShareImage(imageUrl)}>
                                    <ShareIcon className='h-6 w-6' />
                                </IconButton>
                                <IconButton
                                    label={isPanelOpen ? 'Hide image information' : 'Show image information'}
                                    onClick={togglePanel}
                                >
                                    <InformationCircleIcon className='h-6 w-6' />
                                </IconButton>
                                <IconButton label='Close image view' onClick={onClose}>
                                    <XMarkIcon className='h-6 w-6' />
                                </IconButton>
                            </div>
                            <div className='relative inline-block max-h-screen max-w-full p-4'>
                                {!isImageLoaded && (
                                    <div className='absolute inset-0 flex items-center justify-center rounded bg-black/5'>
                                        <svg
                                            className='mr-3 h-6 w-6 animate-spin text-white'
                                            xmlns='http://www.w3.org/2000/svg'
                                            fill='none'
                                            viewBox='0 0 24 24'
                                        >
                                            <circle
                                                className='text-gray-400'
                                                cx='12'
                                                cy='12'
                                                r='10'
                                                stroke='currentColor'
                                                strokeWidth='4'
                                            ></circle>
                                            <path
                                                className='opacity-50'
                                                fill='currentColor'
                                                d='M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z'
                                            ></path>
                                        </svg>
                                    </div>
                                )}
                                <motion.img
                                    key={imageUrl}
                                    initial={shouldZoomOnImage ? { scale: 0.8, opacity: 0 } : { opacity: 1 }}
                                    animate={
                                        shouldZoomOnImage
                                            ? { scale: isImageLoaded ? 1 : 0.95, opacity: isImageLoaded ? 1 : 0 }
                                            : { opacity: 1 }
                                    }
                                    exit={{ opacity: 0 }}
                                    transition={transitionSettings}
                                    onLoad={() => setIsImageLoaded(true)}
                                    src={imageUrl}
                                    alt={image.name}
                                    width={image?.width}
                                    height={image?.height}
                                    className='block max-h-screen max-w-full object-contain'
                                />
                            </div>
                        </motion.div>

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
