import React, { useEffect, useRef, useState, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { XMarkIcon, ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/24/outline';
import { UntaggedFaceResult } from '../../../../types';
import { getPreviewImageUrl, getOriginalImageUrl } from '../../../../api';
import FaceTaggingPanel from './FaceTaggingPanel';

interface ContainLayout {
    x: number;
    y: number;
    w: number;
    h: number;
}

/**
 * Computes the rendered rect of an object-contain image inside a container.
 * Updates on ResizeObserver and when naturalWidth/naturalHeight change.
 */
function useContainLayout(
    containerRef: React.RefObject<HTMLDivElement | null>,
    naturalWidth: number,
    naturalHeight: number,
): ContainLayout {
    const [layout, setLayout] = useState<ContainLayout>({ x: 0, y: 0, w: 0, h: 0 });

    const compute = useCallback(() => {
        const el = containerRef.current;
        if (!el || !naturalWidth || !naturalHeight) return;
        const cw = el.clientWidth;
        const ch = el.clientHeight;
        const scale = Math.min(cw / naturalWidth, ch / naturalHeight);
        const w = naturalWidth * scale;
        const h = naturalHeight * scale;
        setLayout({ x: (cw - w) / 2, y: (ch - h) / 2, w, h });
    }, [containerRef, naturalWidth, naturalHeight]);

    useEffect(() => {
        compute();
        const el = containerRef.current;
        if (!el) return;
        const ro = new ResizeObserver(compute);
        ro.observe(el);
        return () => ro.disconnect();
    }, [compute]);

    return layout;
}

// How much context to show around the face (1.0 = face fills container edge-to-edge,
// 1.5 = face takes up ~2/3 of the view with room around it)
const ZOOM_PADDING = 8;

interface FaceLightboxModalProps {
    faces: UntaggedFaceResult[];
    currentIndex: number;
    onClose: () => void;
    onNavigate: (index: number) => void;
    onTagged: (faceId: number) => void;
    onDeleted: (faceId: number) => void;
    onSuggestionUpdated: (
        faceId: number,
        personId: number | null,
        personName: string | null,
        suggestionCount: number,
    ) => void;
}

const FaceLightboxModal: React.FC<FaceLightboxModalProps> = ({
    faces,
    currentIndex,
    onClose,
    onNavigate,
    onTagged,
    onDeleted,
    onSuggestionUpdated,
}) => {
    const face = faces[currentIndex];
    const containerRef = useRef<HTMLDivElement>(null);

    const [previewSrc, setPreviewSrc] = useState<string>('');
    const [previewLoaded, setPreviewLoaded] = useState(false);
    const [naturalWidth, setNaturalWidth] = useState(0);
    const [naturalHeight, setNaturalHeight] = useState(0);

    const layout = useContainLayout(containerRef, naturalWidth, naturalHeight);

    // Load image on face change
    useEffect(() => {
        if (!face) return;
        setPreviewSrc('');
        setPreviewLoaded(false);
        setNaturalWidth(0);
        setNaturalHeight(0);

        const url = getPreviewImageUrl(face.image_path);
        const img = new Image();
        img.onload = () => {
            setPreviewSrc(url);
            setNaturalWidth(img.naturalWidth);
            setNaturalHeight(img.naturalHeight);
            setPreviewLoaded(true);
        };
        img.onerror = () => {
            const fallback = getOriginalImageUrl(face.image_path);
            const img2 = new Image();
            img2.onload = () => {
                setPreviewSrc(fallback);
                setNaturalWidth(img2.naturalWidth);
                setNaturalHeight(img2.naturalHeight);
                setPreviewLoaded(true);
            };
            img2.onerror = () => setPreviewLoaded(true);
            img2.src = fallback;
        };
        img.src = url;
    }, [face?.face_id]);

    // Keyboard navigation
    useEffect(() => {
        const handleKey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onClose();
            else if (e.key === 'ArrowLeft' && currentIndex > 0) {
                e.preventDefault();
                onNavigate(currentIndex - 1);
            } else if (e.key === 'ArrowRight' && currentIndex < faces.length - 1) {
                e.preventDefault();
                onNavigate(currentIndex + 1);
            }
        };
        window.addEventListener('keydown', handleKey);
        return () => window.removeEventListener('keydown', handleKey);
    }, [onClose, onNavigate, currentIndex, faces.length]);

    if (!face) return null;

    const canPrev = currentIndex > 0;
    const canNext = currentIndex < faces.length - 1;

    // Bounding box in screen space (relative to the container div)
    const origW = face.image_width;
    const origH = face.image_height;
    const hasDims = previewLoaded && layout.w > 0 && origW && origH;

    const boxLeft = hasDims ? layout.x + (face.x1 / origW!) * layout.w : 0;
    const boxTop = hasDims ? layout.y + (face.y1 / origH!) * layout.h : 0;
    const boxWidth = hasDims ? ((face.x2 - face.x1) / origW!) * layout.w : 0;
    const boxHeight = hasDims ? ((face.y2 - face.y1) / origH!) * layout.h : 0;

    // Zoom transform: scale the inner wrapper so the face (+ ZOOM_PADDING) fills the container.
    // transform-origin is fixed at container centre (50% 50%).
    // Formula: tx = zoom * (containerCX - faceCX), ty = zoom * (containerCY - faceCY)
    const containerW = layout.x * 2 + layout.w; // same as clientWidth
    const containerH = layout.y * 2 + layout.h;

    let zoom = 1;
    let tx = 0;
    let ty = 0;

    if (hasDims && boxWidth > 0 && boxHeight > 0 && containerW > 0 && containerH > 0) {
        zoom = Math.min(
            containerW / (boxWidth * ZOOM_PADDING),
            containerH / (boxHeight * ZOOM_PADDING),
            20, // sanity cap
        );
        const faceCX = boxLeft + boxWidth / 2;
        const faceCY = boxTop + boxHeight / 2;
        tx = zoom * (containerW / 2 - faceCX);
        ty = zoom * (containerH / 2 - faceCY);
    }

    return (
        <AnimatePresence>
            <motion.div
                key='face-lightbox-backdrop'
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                className='fixed inset-0 z-50 flex items-stretch bg-zinc-950/25 backdrop-blur-xl dark:bg-zinc-950/50'
                aria-modal='true'
                role='dialog'
                aria-label='Face tagging viewer'
            >
                {/* Left: image area */}
                <div className='relative flex flex-1 flex-col items-stretch'>
                    {/* Top bar */}
                    <div className='flex shrink-0 items-center justify-between px-4 py-3'>
                        <div className='flex items-center gap-3'>
                            <button
                                onClick={onClose}
                                className='rounded-full p-2 text-white/80 transition-colors hover:bg-white/10 hover:text-white focus:outline-none'
                                aria-label='Close'
                            >
                                <XMarkIcon className='h-6 w-6' />
                            </button>
                            <span className='text-sm text-white/60'>
                                {currentIndex + 1} / {faces.length}
                            </span>
                        </div>
                        <div className='flex items-center gap-2'>
                            <button
                                onClick={() => canPrev && onNavigate(currentIndex - 1)}
                                disabled={!canPrev}
                                className='rounded-full p-2 text-white/80 transition-colors hover:bg-white/10 hover:text-white focus:outline-none disabled:opacity-30'
                            >
                                <ChevronLeftIcon className='h-6 w-6' />
                            </button>
                            <button
                                onClick={() => canNext && onNavigate(currentIndex + 1)}
                                disabled={!canNext}
                                className='rounded-full p-2 text-white/80 transition-colors hover:bg-white/10 hover:text-white focus:outline-none disabled:opacity-30'
                            >
                                <ChevronRightIcon className='h-6 w-6' />
                            </button>
                        </div>
                    </div>

                    {/* Image container — clips the zoomed inner wrapper */}
                    <div ref={containerRef} className='relative min-h-0 flex-1 overflow-hidden'>
                        {/* Side nav arrows sit above the zoomed content */}
                        <div className='pointer-events-none absolute inset-y-0 right-0 left-0 z-10 flex items-center justify-between px-4'>
                            <button
                                onClick={(e) => {
                                    e.stopPropagation();
                                    canPrev && onNavigate(currentIndex - 1);
                                }}
                                disabled={!canPrev}
                                className='pointer-events-auto flex h-12 w-10 items-center justify-center rounded-full bg-black/40 text-white backdrop-blur-sm transition-opacity hover:bg-black/60 disabled:opacity-0'
                            >
                                <ChevronLeftIcon className='h-7 w-7' />
                            </button>
                            <button
                                onClick={(e) => {
                                    e.stopPropagation();
                                    canNext && onNavigate(currentIndex + 1);
                                }}
                                disabled={!canNext}
                                className='pointer-events-auto flex h-12 w-10 items-center justify-center rounded-full bg-black/40 text-white backdrop-blur-sm transition-opacity hover:bg-black/60 disabled:opacity-0'
                            >
                                <ChevronRightIcon className='h-7 w-7' />
                            </button>
                        </div>

                        {/* Inner wrapper — receives the zoom transform */}
                        <div
                            style={{
                                position: 'absolute',
                                inset: 0,
                                transformOrigin: '50% 50%',
                                transform: `translate(${tx}px, ${ty}px) scale(${zoom})`,
                                transition: previewLoaded ? 'transform 0.35s ease' : 'none',
                            }}
                        >
                            {/* Image */}
                            {previewSrc && (
                                <motion.img
                                    key={previewSrc}
                                    src={previewSrc}
                                    alt='Face source image'
                                    initial={{ opacity: 0 }}
                                    animate={{ opacity: previewLoaded ? 1 : 0 }}
                                    transition={{ duration: 0.3, ease: 'easeInOut' }}
                                    className='absolute inset-0 h-full w-full object-contain'
                                />
                            )}

                            {/* Bounding box */}
                            {hasDims && boxWidth > 0 && (
                                <div
                                    style={{
                                        position: 'absolute',
                                        left: boxLeft,
                                        top: boxTop,
                                        width: boxWidth,
                                        height: boxHeight,
                                        border: '2px solid rgba(99,102,241,0.9)',
                                        boxShadow: '0 0 0 9999px rgba(0,0,0,0.45), 0 0 12px 2px rgba(99,102,241,0.5)',
                                        pointerEvents: 'none',
                                    }}
                                />
                            )}
                        </div>
                    </div>
                </div>

                {/* Right panel */}
                <div className='flex h-full w-96 shrink-0 flex-col overflow-y-auto border-l border-zinc-200 bg-white dark:border-zinc-700 dark:bg-zinc-900'>
                    <FaceTaggingPanel
                        key={face.face_id}
                        face={face}
                        onTagged={onTagged}
                        onDeleted={onDeleted}
                        onSuggestionUpdated={onSuggestionUpdated}
                    />
                </div>
            </motion.div>
        </AnimatePresence>
    );
};

export default FaceLightboxModal;
