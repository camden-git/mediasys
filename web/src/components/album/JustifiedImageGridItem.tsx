import React, { useState, useEffect, useRef } from 'react';
import { FileInfo } from '../../types.ts';
import { getThumbnailUrl } from '../../api.ts';

interface JustifiedImageGridItemProps {
    image: FileInfo;
    height: number;
    width: number;
    margin: number;
    onClick: () => void;
}

const JustifiedImageGridItem: React.FC<JustifiedImageGridItemProps> = React.memo(
    ({ image, height, width, margin, onClick }) => {
        const [showTooltip, setShowTooltip] = useState(false);
        const [isMobile, setIsMobile] = useState(false);
        const [isInView, setIsInView] = useState(false);
        const containerRef = useRef<HTMLDivElement | null>(null);
        const longPressTimerRef = useRef<number | null>(null);
        const isLongPressRef = useRef(false);
        const suppressNextClickRef = useRef(false);
        const touchStartPosRef = useRef<{ x: number; y: number } | null>(null);

        // touch-capable detection impacts long-press UX hints
        useEffect(() => {
            const checkMobile = () => {
                setIsMobile('ontouchstart' in window || navigator.maxTouchPoints > 0);
            };
            checkMobile();
            window.addEventListener('resize', checkMobile);
            return () => window.removeEventListener('resize', checkMobile);
        }, []);

        const thumbnailUrl = image.thumbnail_path ? getThumbnailUrl(image.thumbnail_path) : undefined;

        // lazy-load background image
        useEffect(() => {
            if (!containerRef.current) return;
            // safari <12 fallback
            if (!(window as any).IntersectionObserver) {
                setIsInView(true);
                return;
            }
            const observer = new IntersectionObserver(
                (entries) => {
                    for (const entry of entries) {
                        if (entry.isIntersecting) {
                            setIsInView(true);
                            observer.disconnect();
                            break;
                        }
                    }
                },
                { rootMargin: '200px' },
            );
            observer.observe(containerRef.current);
            return () => observer.disconnect();
        }, []);

        const style: React.CSSProperties = {
            width: `${width}px`,
            height: `${height}px`,
            marginRight: `${margin}px`,
            display: 'inline-block',
            verticalAlign: 'top',
            position: 'relative',
            overflow: 'hidden',
            backgroundColor: '#eee',
            cursor: 'pointer',
            // disable long-press save on mobile
            WebkitTouchCallout: 'none',
            WebkitUserSelect: 'none',
            userSelect: 'none',
            pointerEvents: 'auto',
        };

        // prevent context menu and long-press save
        const handleContextMenu = (e: React.MouseEvent) => {
            e.preventDefault();
            return false;
        };

        // long-press detection and click suppression on iOS Safari
        const clearLongPressTimer = () => {
            if (longPressTimerRef.current !== null) {
                window.clearTimeout(longPressTimerRef.current);
                longPressTimerRef.current = null;
            }
        };

        useEffect(() => {
            return () => clearLongPressTimer();
        }, []);

        const handleTouchStart = (e: React.TouchEvent) => {
            const touch = e.touches[0];
            touchStartPosRef.current = { x: touch.clientX, y: touch.clientY };
            isLongPressRef.current = false;
            suppressNextClickRef.current = false;
            clearLongPressTimer();
            longPressTimerRef.current = window.setTimeout(() => {
                isLongPressRef.current = true;
                suppressNextClickRef.current = true;
                if (isMobile) {
                    setShowTooltip(true);
                    setTimeout(() => setShowTooltip(false), 2000);
                }
            }, 500);
        };

        const handleTouchMove = (e: React.TouchEvent) => {
            if (!touchStartPosRef.current) return;
            const touch = e.touches[0];
            const dx = Math.abs(touch.clientX - touchStartPosRef.current.x);
            const dy = Math.abs(touch.clientY - touchStartPosRef.current.y);
            if (dx > 10 || dy > 10) {
                clearLongPressTimer();
            }
        };

        const handleTouchEnd = () => {
            clearLongPressTimer();
            touchStartPosRef.current = null;
            // keep suppression to swallow the synthetic click, then reset shortly
            if (isLongPressRef.current) {
                setTimeout(() => {
                    suppressNextClickRef.current = false;
                    isLongPressRef.current = false;
                }, 400);
            }
        };

        const handleClick = () => {
            if (suppressNextClickRef.current) {
                return;
            }
            onClick();
        };

        const handleDragStart = (e: React.DragEvent) => {
            e.preventDefault();
            return false;
        };

        return (
            <div
                ref={containerRef}
                style={style}
                onClick={handleClick}
                onContextMenu={handleContextMenu}
                onTouchStart={handleTouchStart}
                onTouchMove={handleTouchMove}
                onTouchEnd={handleTouchEnd}
                onDragStart={handleDragStart}
            >
                <div
                    role='img'
                    aria-label={image.name}
                    className='absolute top-0 left-0 h-full w-full'
                    style={{
                        width: '100%',
                        height: '100%',
                        backgroundImage: isInView && thumbnailUrl ? `url(${thumbnailUrl})` : undefined,
                        backgroundSize: 'cover',
                        backgroundPosition: 'center',
                        WebkitTouchCallout: 'none',
                        WebkitUserSelect: 'none',
                        userSelect: 'none',
                        pointerEvents: 'none',
                    }}
                />
                {showTooltip && (
                    <div
                        className='bg-opacity-75 pointer-events-none absolute bottom-2 left-1/2 z-10 -translate-x-1/2 transform rounded bg-black px-2 py-1 text-xs text-white'
                        style={{
                            fontSize: '11px',
                            whiteSpace: 'nowrap',
                            maxWidth: '90%',
                        }}
                    >
                        Tap to view full size & save
                    </div>
                )}
            </div>
        );
    },
);

export default JustifiedImageGridItem;
