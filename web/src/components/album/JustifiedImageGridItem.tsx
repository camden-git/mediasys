import React, { useState, useEffect } from 'react';
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
        
        // Detect if device is mobile/touch
        useEffect(() => {
            const checkMobile = () => {
                setIsMobile('ontouchstart' in window || navigator.maxTouchPoints > 0);
            };
            checkMobile();
            window.addEventListener('resize', checkMobile);
            return () => window.removeEventListener('resize', checkMobile);
        }, []);
        
        if (!image.thumbnail_path) {
            return null;
        }
        const thumbnailUrl = getThumbnailUrl(image.thumbnail_path);

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
            // Disable long-press save on mobile
            WebkitTouchCallout: 'none',
            WebkitUserSelect: 'none',
            userSelect: 'none',
            pointerEvents: 'auto',
        };

        // Prevent context menu and long-press save
        const handleContextMenu = (e: React.MouseEvent) => {
            e.preventDefault();
            return false;
        };

        const handleTouchStart = (e: React.TouchEvent) => {
            // Prevent long-press context menu
            e.preventDefault();
        };

        const handleTouchEnd = (e: React.TouchEvent) => {
            // Show tooltip briefly on mobile devices only
            if (isMobile) {
                setShowTooltip(true);
                setTimeout(() => setShowTooltip(false), 2000);
            }
        };

        const handleDragStart = (e: React.DragEvent) => {
            e.preventDefault();
            return false;
        };

        return (
            <div 
                style={style} 
                onClick={onClick}
                onContextMenu={handleContextMenu}
                onTouchStart={handleTouchStart}
                onTouchEnd={handleTouchEnd}
                onDragStart={handleDragStart}
            >
                <img
                    src={thumbnailUrl}
                    alt={image.name}
                    className='absolute top-0 left-0 h-full w-full object-cover'
                    loading='lazy'
                    style={{
                        width: '100%',
                        height: '100%',
                        // Additional CSS to prevent save
                        WebkitTouchCallout: 'none',
                        WebkitUserSelect: 'none',
                        userSelect: 'none',
                        pointerEvents: 'none', // Prevent direct interaction with img
                    }}
                    onError={(e) => {
                        (e.target as HTMLImageElement).style.display = 'none';
                        console.error(`Failed to load thumbnail: ${thumbnailUrl}`);
                    }}
                    onContextMenu={handleContextMenu}
                    onDragStart={handleDragStart}
                />
                
                {/* Tooltip */}
                {showTooltip && (
                    <div 
                        className="absolute bottom-2 left-1/2 transform -translate-x-1/2 bg-black bg-opacity-75 text-white text-xs px-2 py-1 rounded pointer-events-none z-10"
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
