import React, { Fragment } from 'react';
import { useStoreState } from '../../store/hooks';
import Notification from './Notification.tsx';

interface FlashMessageRenderProps {
    byKey: string;
    className?: string;
}

const FlashMessageRender: React.FC<FlashMessageRenderProps> = ({ byKey }) => {
    const flashes = useStoreState((state) => state.ui.flashes);
    const filteredFlashes = flashes.filter((flash) => flash.key === byKey);

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
                    <Fragment key={`${flash.key}-${index}`}>
                        {index > 0 && <div className='mt-2'></div>}
                        <Notification type={flash.type} title={'flash message'}>
                            {flash.message}
                        </Notification>
                    </Fragment>
                ))}
            </div>
        </div>
    );
};

export default FlashMessageRender;
