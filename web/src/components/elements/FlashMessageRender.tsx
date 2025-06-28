import React from 'react';
import { useStoreState } from '../../store/hooks';
import { FlashMessage } from '../../store/uiStore';

interface FlashMessageRenderProps {
    byKey: string;
    className?: string;
}

const FlashMessageRender: React.FC<FlashMessageRenderProps> = ({ byKey, className = '' }) => {
    const flashes = useStoreState((state) => state.ui.flashes);
    const filteredFlashes = flashes.filter((flash) => flash.key === byKey);

    if (filteredFlashes.length === 0) {
        return null;
    }

    return (
        <div className={className}>
            {filteredFlashes.map((flash: FlashMessage, index: number) => (
                <div
                    key={`${flash.key}-${index}`}
                    className={`mb-4 rounded-md p-4 ${
                        flash.type === 'error'
                            ? 'border border-red-200 bg-red-50 text-red-800'
                            : flash.type === 'success'
                              ? 'border border-green-200 bg-green-50 text-green-800'
                              : flash.type === 'warning'
                                ? 'border border-yellow-200 bg-yellow-50 text-yellow-800'
                                : 'border border-blue-200 bg-blue-50 text-blue-800'
                    }`}
                >
                    {flash.message}
                </div>
            ))}
        </div>
    );
};

export default FlashMessageRender;
