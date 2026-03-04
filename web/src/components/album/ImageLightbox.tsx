import React, { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { format } from 'date-fns';
import { FileInfo, FaceData } from '../../types.ts';
import {
    getOriginalImageUrl,
    getThumbnailUrl,
    getPreviewImageUrl,
    getFacesForImage,
    getPersonKeyPhotoUrl,
} from '../../api.ts';
import MetadataPanel from './MetadataPanel';
import LoadingSpinner from '../elements/LoadingSpinner';
import {
    DocumentArrowDownIcon,
    InformationCircleIcon,
    XMarkIcon,
    ChevronLeftIcon,
    ChevronRightIcon,
    CheckIcon,
    ShareIcon,
} from '@heroicons/react/24/outline';
import { Heading } from '../elements/Heading.tsx';
import { Text } from '../elements/Text.tsx';

interface ImageLightboxProps {
    image: FileInfo | null;
    imageIndex?: number;
    totalImages?: number;
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

// Generate a consistent hue from a numeric ID
function hueFromId(id: number): number {
    return (id * 137.508) % 360;
}

function personInitials(name: string): string {
    return name
        .split(' ')
        .slice(0, 2)
        .map((w) => w[0]?.toUpperCase() ?? '')
        .join('');
}

interface FaceBubbleProps {
    face: FaceData;
}

const FaceBubble: React.FC<FaceBubbleProps> = ({ face }) => {
    if (!face.person || !face.confirmed) return null;
    const hue = hueFromId(face.person.id);
    const bg = `hsl(${hue}, 55%, 40%)`;
    const initials = personInitials(face.person.primary_name);
    const hasKeyPhoto = !!face.person.key_photo_face_id;
    return (
        <div className='flex flex-col items-center gap-0.5'>
            <div
                className='flex h-9 w-9 items-center justify-center overflow-hidden rounded text-sm font-semibold text-white ring-2 ring-white/20'
                style={{ backgroundColor: bg }}
                title={face.person.primary_name}
            >
                {hasKeyPhoto ? (
                    <img
                        src={getPersonKeyPhotoUrl(face.person.id)}
                        alt={face.person.primary_name}
                        className='h-9 w-9 rounded object-cover'
                        onError={(e) => {
                            (e.currentTarget as HTMLImageElement).style.display = 'none';
                        }}
                    />
                ) : (
                    initials
                )}
            </div>
            <span className='max-w-[52px] truncate text-center text-[10px] leading-tight text-white/70'>
                {face.person.primary_name.split(' ')[0]}
            </span>
        </div>
    );
};

const ImageLightbox: React.FC<ImageLightboxProps> = ({
    image,
    imageIndex,
    totalImages,
    onClose,
    onPrev,
    onNext,
    canPrev = false,
    canNext = false,
}) => {
    const [isPanelOpen, setIsPanelOpen] = useState(false);
    const [thumbnailSrc, setThumbnailSrc] = useState<string>('');
    const [previewSrc, setPreviewSrc] = useState<string>('');
    const [previewLoaded, setPreviewLoaded] = useState(false);
    const [textVisible, setTextVisible] = useState(false);
    const [badgeHovered, setBadgeHovered] = useState(false);
    const [faces, setFaces] = useState<FaceData[]>([]);
    const [isSharing, setIsSharing] = useState(false);
    const previewImgRef = useRef<HTMLImageElement | null>(null);

    // Reset and start loading on image change
    useEffect(() => {
        if (!image) {
            setIsPanelOpen(false);
            setThumbnailSrc('');
            setPreviewSrc('');
            setPreviewLoaded(false);
            setFaces([]);
            return;
        }

        setPreviewSrc('');
        setPreviewLoaded(false);
        setFaces([]);

        // Stage 1: show thumbnail immediately (likely cached from grid)
        if (image.thumbnail_path) {
            setThumbnailSrc(getThumbnailUrl(image.thumbnail_path));
        } else {
            setThumbnailSrc('');
        }

        // Stage 2: load preview in background; fade in on top when ready
        const previewUrl = getPreviewImageUrl(image.path);
        const previewImg = new Image();
        previewImgRef.current = previewImg;
        previewImg.onload = () => {
            setPreviewSrc(previewUrl);
            setPreviewLoaded(true);
        };
        previewImg.onerror = () => {
            setPreviewSrc(getOriginalImageUrl(image.path));
            setPreviewLoaded(true);
        };
        previewImg.src = previewUrl;

        // Fetch face data
        getFacesForImage(image.path)
            .then((data) => setFaces(data))
            .catch(() => setFaces([]));

        return () => {
            if (previewImgRef.current) {
                previewImgRef.current.onload = null;
                previewImgRef.current.onerror = null;
            }
        };
    }, [image]);

    useEffect(() => {
        if (!image) {
            setTextVisible(false);
            return;
        }

        if (previewLoaded) {
            // Don't force visible — if slow load already showed text, just schedule collapse
            const hideTimer = setTimeout(() => setTextVisible(false), 3000);
            return () => clearTimeout(hideTimer);
        } else {
            // Reset for new image, only expand if loading takes longer than 250ms
            setTextVisible(false);
            const slowTimer = setTimeout(() => setTextVisible(true), 250);
            return () => clearTimeout(slowTimer);
        }
    }, [previewLoaded, image]);

    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                onClose();
                return;
            }
            if (event.key === 'ArrowLeft' && canPrev && onPrev) {
                event.preventDefault();
                onPrev();
                return;
            }
            if (event.key === 'ArrowRight' && canNext && onNext) {
                event.preventDefault();
                onNext();
                return;
            }
            if (event.key.toLowerCase() === 'i') {
                event.preventDefault();
                setIsPanelOpen((prev) => !prev);
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

    const handleDownloadImage = async () => {
        if (!image) return;
        try {
            const fullUrl = getOriginalImageUrl(image.path);
            const response = await fetch(fullUrl, { mode: 'cors' });
            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;
            link.setAttribute('download', image.name || 'download.jpg');
            document.body.appendChild(link);
            link.click();
            link.remove();
            window.URL.revokeObjectURL(url);
        } catch (error) {
            console.error('Error downloading image:', error);
        }
    };

    const handleShareImage = async () => {
        if (!image || isSharing) return;
        setIsSharing(true);
        try {
            const fullUrl = getOriginalImageUrl(image.path);
            const response = await fetch(fullUrl, { mode: 'cors' });
            const blob = await response.blob();
            const ext = image.name.split('.').pop() ?? 'jpg';
            const file = new File([blob], image.name, { type: blob.type || `image/${ext}` });
            if (navigator.canShare && navigator.canShare({ files: [file] })) {
                await navigator.share({ files: [file], title: image.name });
            }
        } catch (error) {
            if ((error as Error).name !== 'AbortError') {
                console.error('Error sharing image:', error);
            }
        } finally {
            setIsSharing(false);
        }
    };

    const formatDate = (timestamp?: number): string | null => {
        if (!timestamp) return null;
        try {
            return format(new Date(timestamp * 1000), 'MMM d, yyyy');
        } catch {
            return null;
        }
    };

    const taggedFaces = faces.filter((f) => f.person != null);
    const maxVisibleFaces = 5;
    const visibleFaces = taggedFaces.slice(0, maxVisibleFaces);
    const overflowCount = taggedFaces.length - visibleFaces.length;

    const cameraInfo = image
        ? [
              image.camera_model,
              image.focal_length ? `${Math.round(image.focal_length)}mm` : null,
              image.aperture ? `ƒ/${image.aperture.toFixed(1)}` : null,
              image.iso ? `ISO ${image.iso}` : null,
          ]
              .filter(Boolean)
              .join(' · ')
        : '';

    return (
        <AnimatePresence>
            {image && (
                <motion.div
                    key='lightbox-backdrop'
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    className='fixed inset-0 z-50 flex items-stretch bg-zinc-950/25 backdrop-blur-xl dark:bg-zinc-950/50'
                    onClick={handleBackdropClick}
                    aria-modal='true'
                    role='dialog'
                    aria-label='Image viewer'
                >
                    <motion.div
                        className='relative flex flex-1 flex-col items-stretch'
                        layout
                        transition={transitionSettings}
                    >
                        {/* Top bar */}
                        <div className='flex shrink-0 items-center justify-between px-4 py-3'>
                            <div className='flex items-center gap-3'>
                                <button
                                    onClick={onClose}
                                    className='rounded-full p-2 text-white/80 transition-colors hover:bg-white/10 hover:text-white focus:outline-none'
                                    aria-label='Close image view'
                                >
                                    <XMarkIcon className='h-6 w-6' />
                                </button>
                                {imageIndex != null && totalImages != null && (
                                    <span className='text-sm text-white/60'>
                                        {imageIndex + 1} / {totalImages}
                                    </span>
                                )}
                            </div>
                            <div className='flex items-center gap-2'>
                                <button
                                    onClick={handleDownloadImage}
                                    className='flex items-center gap-1.5 rounded-full border border-white/20 bg-white/10 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-white/20 focus:outline-none'
                                    aria-label='Download original image'
                                >
                                    <DocumentArrowDownIcon className='h-5 w-5 shrink-0' />
                                    <span className='hidden sm:inline'>Download</span>
                                </button>
                                {typeof navigator !== 'undefined' && !!navigator.share && (
                                    <button
                                        onClick={handleShareImage}
                                        disabled={isSharing}
                                        className='flex items-center gap-1.5 rounded-full border border-white/20 bg-white/10 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-white/20 focus:outline-none disabled:opacity-50'
                                        aria-label='Share image'
                                    >
                                        {isSharing ? (
                                            <LoadingSpinner />
                                        ) : (
                                            <ShareIcon className='h-5 w-5 shrink-0' />
                                        )}
                                        <span className='hidden sm:inline'>Share</span>
                                    </button>
                                )}
                                <button
                                    onClick={togglePanel}
                                    className={`rounded-full p-2 transition-colors focus:outline-none ${isPanelOpen ? 'bg-white/20 text-white' : 'text-white/80 hover:bg-white/10 hover:text-white'}`}
                                    aria-label={isPanelOpen ? 'Hide image information' : 'Show image information'}
                                >
                                    <InformationCircleIcon className='h-6 w-6' />
                                </button>
                            </div>
                        </div>

                        {/* Main image area */}
                        <div className='relative min-h-0 flex-1 overflow-hidden' onClick={handleBackdropClick}>
                            {/* Nav arrows */}
                            <div className='pointer-events-none absolute inset-y-0 right-0 left-0 z-10 flex items-center justify-between px-4'>
                                <button
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        onPrev && onPrev();
                                    }}
                                    disabled={!canPrev}
                                    className='pointer-events-auto flex h-12 w-10 items-center justify-center rounded-full bg-black/40 text-white backdrop-blur-sm transition-opacity hover:bg-black/60 disabled:opacity-0'
                                    aria-label='Previous image'
                                >
                                    <ChevronLeftIcon className='h-7 w-7' />
                                </button>
                                <button
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        onNext && onNext();
                                    }}
                                    disabled={!canNext}
                                    className='pointer-events-auto flex h-12 w-10 items-center justify-center rounded-full bg-black/40 text-white backdrop-blur-sm transition-opacity hover:bg-black/60 disabled:opacity-0'
                                    aria-label='Next image'
                                >
                                    <ChevronRightIcon className='h-7 w-7' />
                                </button>
                            </div>

                            {/* Image stack — explicitly inset so it has a definite padded box to center within */}
                            <div className='absolute inset-x-0 top-0 bottom-20'>
                                <div className='relative h-full w-full'>
                                    {/* Thumbnail layer — always visible as base */}
                                    {thumbnailSrc && (
                                        <img
                                            src={thumbnailSrc}
                                            alt={image.name}
                                            className='absolute inset-0 h-full w-full object-contain'
                                            onContextMenu={(e) => e.preventDefault()}
                                            onDragStart={(e) => e.preventDefault()}
                                            style={{ WebkitTouchCallout: 'none', userSelect: 'none' }}
                                        />
                                    )}
                                    {/* Preview layer — fades in on top, no exit */}
                                    {previewSrc && (
                                        <motion.img
                                            key={previewSrc}
                                            src={previewSrc}
                                            alt={image.name}
                                            initial={{ opacity: 0 }}
                                            animate={{ opacity: previewLoaded ? 1 : 0 }}
                                            transition={{ duration: 0.35, ease: 'easeInOut' }}
                                            className='absolute inset-0 h-full w-full object-contain'
                                            onContextMenu={(e) => e.preventDefault()}
                                            onDragStart={(e) => e.preventDefault()}
                                            style={{ WebkitTouchCallout: 'none', userSelect: 'none' }}
                                        />
                                    )}
                                </div>
                            </div>
                        </div>

                        {/* Bottom info bar */}
                        <div
                            className='absolute right-0 bottom-0 left-0 z-10 flex items-end justify-between px-5 py-4'
                            style={{ background: 'linear-gradient(to top, rgba(9,9,11,0.85) 0%, rgba(9,9,11,0) 100%)' }}
                        >
                            {/* Left: filename + date + camera + spinner */}
                            <div className='flex items-end gap-2'>
                                <div className='flex flex-col'>
                                    <div className='flex gap-2'>
                                        <motion.div
                                            layout
                                            onHoverStart={() => setBadgeHovered(true)}
                                            onHoverEnd={() => setBadgeHovered(false)}
                                            className='flex items-center overflow-hidden rounded-lg border border-white/10 bg-gray-600/10 px-1 py-1 text-white/70 backdrop-blur-sm'
                                        >
                                            <AnimatePresence mode='wait'>
                                                {!previewLoaded ? (
                                                    <motion.span
                                                        key='spinner-icon'
                                                        initial={{ opacity: 0 }}
                                                        animate={{ opacity: 1 }}
                                                        exit={{ opacity: 0 }}
                                                        transition={{ duration: 0.15 }}
                                                        className='flex shrink-0 items-center justify-center'
                                                    >
                                                        <LoadingSpinner />
                                                    </motion.span>
                                                ) : (
                                                    <motion.span
                                                        key='check-icon'
                                                        initial={{ opacity: 0 }}
                                                        animate={{ opacity: 1 }}
                                                        exit={{ opacity: 0 }}
                                                        transition={{ duration: 0.15 }}
                                                        className='flex shrink-0 items-center justify-center text-green-400'
                                                    >
                                                        <CheckIcon className='h-6 w-6' />
                                                    </motion.span>
                                                )}
                                            </AnimatePresence>
                                            <motion.span
                                                initial={false}
                                                animate={{
                                                    width: !textVisible && !badgeHovered ? 0 : 'auto',
                                                    opacity: !textVisible && !badgeHovered ? 0 : 1,
                                                    paddingLeft: !textVisible && !badgeHovered ? 0 : 6,
                                                }}
                                                transition={{ duration: 0.3, ease: 'easeInOut' }}
                                                className='overflow-hidden text-xs whitespace-nowrap'
                                            >
                                                {previewLoaded ? 'Loaded Preview' : 'Loading Preview...'}
                                            </motion.span>
                                        </motion.div>
                                        <Heading invert>{image.name}</Heading>
                                    </div>

                                    <Text>
                                        {[formatDate(image.taken_at), cameraInfo].filter(Boolean).join('  ·  ')}
                                    </Text>
                                </div>
                            </div>

                            {/* Right: face bubbles */}
                            <div className='flex items-end gap-3'>
                                {visibleFaces.length > 0 && (
                                    <div className='flex items-end gap-2'>
                                        {visibleFaces.map((face) => (
                                            <FaceBubble key={face.id} face={face} />
                                        ))}
                                        {overflowCount > 0 && (
                                            <div className='flex flex-col items-center gap-0.5'>
                                                <div className='flex h-9 w-9 items-center justify-center rounded-full bg-white/20 text-xs font-semibold text-white ring-2 ring-white/20'>
                                                    +{overflowCount}
                                                </div>
                                            </div>
                                        )}
                                    </div>
                                )}
                            </div>
                        </div>
                    </motion.div>

                    {/* Metadata panel */}
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
                                <MetadataPanel isOpen={true} onClose={togglePanel} image={image} faces={faces} />
                            </motion.div>
                        )}
                    </AnimatePresence>
                </motion.div>
            )}
        </AnimatePresence>
    );
};

export default ImageLightbox;
