import React, { Fragment, useEffect, useRef } from 'react';
import { useUIStore } from '../../store/useUIStore';
import Notification from './Notification.tsx';

interface FlashMessageRenderProps {
    byKey: string;
    className?: string;
}

const FlashMessageRender: React.FC<FlashMessageRenderProps> = ({ byKey }) => {
    const flashes = useUIStore((s) => s.flashes);
    const clearFlashes = useUIStore((s) => s.clearFlashes);
    const removeFlash = useUIStore((s) => s.removeFlash);
    const filteredFlashes = flashes.filter((flash) => flash.key === byKey);
    const clearTimer = useRef<number | undefined>(undefined);

    // Drop this key's flashes once the renderer goes away (e.g. on route change) so they don't reappear later.
    // The clear is deferred a tick so StrictMode's simulated unmount/remount doesn't wipe them.
    useEffect(() => {
        window.clearTimeout(clearTimer.current);
        return () => {
            clearTimer.current = window.setTimeout(() => clearFlashes(byKey), 0);
        };
    }, [byKey, clearFlashes]);

    if (filteredFlashes.length === 0) {
        return null;
    }

    return (
        <div
            aria-live='assertive'
            className='pointer-events-none fixed inset-0 z-100 flex items-end px-4 py-6 sm:items-start sm:p-6'
        >
            <div className='flex w-full flex-col items-center space-y-4 sm:items-end'>
                {filteredFlashes.map((flash, index) => (
                    <Fragment key={flash.id ?? `${flash.key}-${index}`}>
                        {index > 0 && <div className='mt-2'></div>}
                        <Notification
                            type={flash.type}
                            title={flash.title}
                            onClose={() => flash.id !== undefined && removeFlash(flash.id)}
                        >
                            {flash.message}
                        </Notification>
                    </Fragment>
                ))}
            </div>
        </div>
    );
};

export default FlashMessageRender;
