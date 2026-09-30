import React, { useState } from 'react';

interface AvatarProps {
    src?: string;
    initials?: string;
    className?: string;
    alt?: string;
    style?: React.CSSProperties;
}

export const Avatar: React.FC<AvatarProps> = ({ src, initials, className, alt = '', style }) => {
    // remember which src failed so a new src gets a fresh attempt
    const [failedSrc, setFailedSrc] = useState<string | undefined>(undefined);
    const showImg = src && src !== failedSrc;

    return (
        <span
            data-slot='avatar'
            className={`inline-flex items-center justify-center overflow-hidden ${className ?? ''}`}
            style={style}
        >
            {showImg ? (
                <img src={src} alt={alt} className='h-full w-full object-cover' onError={() => setFailedSrc(src)} />
            ) : (
                <span className='text-xs leading-none font-semibold text-white'>{initials}</span>
            )}
        </span>
    );
};
