import React from 'react';
import { format } from 'date-fns';
import { FileInfo } from '../../types.ts';
import { Subheading } from '../elements/Heading.tsx';
import { XMarkIcon } from '@heroicons/react/24/solid';
import { Text } from '../elements/Text.tsx';

interface MetadataPanelProps {
    isOpen: boolean;
    onClose: () => void;
    image: FileInfo | null;
}

const MetadataPanel: React.FC<MetadataPanelProps> = ({ isOpen, onClose, image }) => {
    const formatBytes = (bytes: number, decimals = 2): string => {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const dm = decimals < 0 ? 0 : decimals;
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
    };

    const formatShutterSpeed = (speed?: string): string | null => {
        if (!speed) return null;

        if (speed.includes('/') || speed.includes('"') || speed.includes('s')) return speed;

        return speed;
    };

    const formatAperture = (aperture?: number): string | null => {
        if (!aperture) return null;
        return `ƒ/${aperture.toFixed(1)}`; // e.g., ƒ/2.8
    };

    const formatFocalLength = (length?: number): string | null => {
        if (!length) return null;
        return `${length.toFixed(0)} mm`; // e.g., 50 mm
    };

    const formatDate = (timestamp?: number): string | null => {
        if (!timestamp) return null;
        try {
            return format(new Date(timestamp * 1000), "MMMM d, yyyy 'at' h:mm a");
        } catch (e) {
            console.error('Error formatting date:', e);
            return 'Invalid Date';
        }
    };

    return (
        <>
            {isOpen && image && (
                <>
                    <div
                        className='h-full w-sm overflow-y-auto bg-gray-100 shadow-lg'
                        aria-modal='true'
                        role='dialog'
                        aria-labelledby='metadata-panel-title'
                    >
                        <div className='p-6'>
                            <div className='mb-6 flex items-center justify-between'>
                                <Subheading huge>this is my really long image name</Subheading>
                                <button
                                    onClick={onClose}
                                    className='text-gray-500 transition-colors hover:text-gray-800'
                                    aria-label='Close metadata panel'
                                >
                                    <XMarkIcon className='h-6 w-6' />
                                </button>
                            </div>
                            <Text className={'mb-4'}>
                                With my even longer description. testWith my even longer description. testWith my longer
                                description. testWith my even longer description. testWith my even longer description.
                                testWith my even longer description. test
                            </Text>

                            <div className='space-y-3 text-sm'>
                                {/* File Info */}
                                <div className='mb-3 border-b pb-3'>
                                    <p className='mb-1 text-base font-semibold break-words'>{image.name}</p>
                                    <p className='break-words text-gray-500'>Path: {image.path}</p>
                                    {image.width && image.height && (
                                        <p className='text-gray-500'>
                                            {image.width} x {image.height} px
                                        </p>
                                    )}
                                    <p className='text-gray-500'>{formatBytes(image.size)}</p>
                                </div>

                                {/* Date/Time */}
                                {(image.taken_at || image.mod_time) && (
                                    <div className='mb-3 border-b pb-3'>
                                        {image.taken_at && (
                                            <p>
                                                <span className='font-medium text-gray-700'>Taken:</span>{' '}
                                                {formatDate(image.taken_at)}
                                            </p>
                                        )}
                                        <p>
                                            <span className='font-medium text-gray-700'>Modified:</span>{' '}
                                            {formatDate(image.mod_time)}
                                        </p>
                                    </div>
                                )}

                                {/* Camera & Exposure */}
                                {(image.camera_make ||
                                    image.camera_model ||
                                    image.aperture ||
                                    image.shutter_speed ||
                                    image.iso ||
                                    image.focal_length) && (
                                    <div className='mb-3 border-b pb-3'>
                                        {image.camera_make && image.camera_model && (
                                            <p className='font-medium text-gray-700'>
                                                {image.camera_make} {image.camera_model}
                                            </p>
                                        )}
                                        {(image.aperture || image.shutter_speed || image.iso || image.focal_length) && (
                                            <p className='text-gray-500'>
                                                {formatFocalLength(image.focal_length)}
                                                {formatAperture(image.aperture) &&
                                                    ` • ${formatAperture(image.aperture)}`}
                                                {formatShutterSpeed(image.shutter_speed) &&
                                                    ` • ${formatShutterSpeed(image.shutter_speed)}`}
                                                {image.iso && ` • ISO ${image.iso}`}
                                            </p>
                                        )}
                                        {/* Lens Info */}
                                        {image.lens_make && image.lens_model && (
                                            <p className='mt-1 text-gray-500'>
                                                {image.lens_make} {image.lens_model}
                                            </p>
                                        )}
                                        {!image.lens_model && image.lens_make && (
                                            <p className='mt-1 text-gray-500'>{image.lens_make}</p>
                                        )}
                                    </div>
                                )}

                                {/* Add other sections like Location, Tags, People later */}
                            </div>
                        </div>
                    </div>
                </>
            )}
        </>
    );
};

export default MetadataPanel;
