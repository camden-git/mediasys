import React from 'react';
import { format } from 'date-fns';
import { FileInfo, FaceData } from '../../types.ts';
import { XMarkIcon } from '@heroicons/react/24/solid';
import { bytesToString } from '../../lib/formatters.ts';

interface MetadataPanelProps {
    isOpen: boolean;
    onClose: () => void;
    image: FileInfo | null;
    faces?: FaceData[];
}

const MetadataPanel: React.FC<MetadataPanelProps> = ({ isOpen, onClose, image, faces = [] }) => {
    const formatShutterSpeed = (speed?: string): string | null => {
        if (!speed) return null;
        return speed;
    };

    const formatAperture = (aperture?: number): string | null => {
        if (!aperture) return null;
        return `ƒ/${aperture.toFixed(1)}`;
    };

    const formatFocalLength = (length?: number): string | null => {
        if (!length) return null;
        return `${length.toFixed(0)} mm`;
    };

    const formatDate = (timestamp?: number): string | null => {
        if (!timestamp) return null;
        try {
            return format(new Date(timestamp * 1000), "MMMM d, yyyy 'at' h:mm a");
        } catch {
            return 'Invalid Date';
        }
    };

    const taggedFaces = faces.filter((f) => f.person != null && f.confirmed);
    const peopleMap = new Map<number, { name: string; count: number }>();
    for (const face of taggedFaces) {
        if (!face.person) continue;
        const existing = peopleMap.get(face.person.id);
        if (existing) {
            existing.count += 1;
        } else {
            peopleMap.set(face.person.id, { name: face.person.primary_name, count: 1 });
        }
    }
    const people = Array.from(peopleMap.values());

    return (
        <>
            {isOpen && image && (
                <div
                    className='h-full w-96 overflow-y-auto bg-zinc-900 text-white shadow-lg'
                    aria-modal='true'
                    role='dialog'
                    aria-labelledby='metadata-panel-title'
                >
                    <div className='p-6'>
                        <div className='mb-6 flex items-start justify-between gap-2'>
                            <h2 id='metadata-panel-title' className='text-base font-semibold break-all text-white'>
                                {image.name}
                            </h2>
                            <button
                                onClick={onClose}
                                className='flex-shrink-0 text-white/50 transition-colors hover:text-white'
                                aria-label='Close metadata panel'
                            >
                                <XMarkIcon className='h-5 w-5' />
                            </button>
                        </div>

                        <div className='space-y-4 text-sm'>
                            {/* File Info */}
                            <div className='border-b border-white/10 pb-4'>
                                <p className='mb-1 text-xs font-semibold tracking-wider text-white/40 uppercase'>
                                    File
                                </p>
                                {image.width && image.height && (
                                    <p className='text-white/70'>
                                        {image.width} × {image.height} px
                                    </p>
                                )}
                                <p className='text-white/70'>{bytesToString(image.size)}</p>
                                <p className='mt-1 text-xs break-all text-white/40'>{image.path}</p>
                            </div>

                            {/* Date/Time */}
                            {(image.taken_at || image.mod_time) && (
                                <div className='border-b border-white/10 pb-4'>
                                    <p className='mb-1 text-xs font-semibold tracking-wider text-white/40 uppercase'>
                                        Date
                                    </p>
                                    {image.taken_at && (
                                        <p className='text-white/70'>
                                            <span className='text-white/50'>Taken</span> {formatDate(image.taken_at)}
                                        </p>
                                    )}
                                    <p className='text-white/70'>
                                        <span className='text-white/50'>Modified</span> {formatDate(image.mod_time)}
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
                                <div className='border-b border-white/10 pb-4'>
                                    <p className='mb-1 text-xs font-semibold tracking-wider text-white/40 uppercase'>
                                        Camera
                                    </p>
                                    {(image.camera_make || image.camera_model) && (
                                        <p className='font-medium text-white'>
                                            {[image.camera_make, image.camera_model].filter(Boolean).join(' ')}
                                        </p>
                                    )}
                                    {(image.aperture || image.shutter_speed || image.iso || image.focal_length) && (
                                        <p className='text-white/70'>
                                            {[
                                                formatFocalLength(image.focal_length),
                                                formatAperture(image.aperture),
                                                formatShutterSpeed(image.shutter_speed),
                                                image.iso ? `ISO ${image.iso}` : null,
                                            ]
                                                .filter(Boolean)
                                                .join(' · ')}
                                        </p>
                                    )}
                                    {(image.lens_make || image.lens_model) && (
                                        <p className='mt-1 text-white/50'>
                                            {[image.lens_make, image.lens_model].filter(Boolean).join(' ')}
                                        </p>
                                    )}
                                </div>
                            )}

                            {/* Star Rating */}
                            {image.rating !== undefined && image.rating !== null && (
                                <div className='border-b border-white/10 pb-4'>
                                    <p className='mb-1 text-xs font-semibold tracking-wider text-white/40 uppercase'>
                                        Rating
                                    </p>
                                    <p className='text-base tracking-wide text-yellow-400'>
                                        {'★'.repeat(image.rating)}
                                        <span className='text-white/20'>{'★'.repeat(5 - image.rating)}</span>
                                    </p>
                                </div>
                            )}

                            {/* Tags */}
                            {image.tags && image.tags.length > 0 && (
                                <div className='border-b border-white/10 pb-4'>
                                    <p className='mb-2 text-xs font-semibold tracking-wider text-white/40 uppercase'>
                                        Tags
                                    </p>
                                    <div className='flex flex-wrap gap-1'>
                                        {image.tags.map((t) => {
                                            const colorClass =
                                                t.source === 'xmp'
                                                    ? 'bg-white/10 text-white/60'
                                                    : t.source === 'album_default'
                                                      ? 'bg-blue-800/40 text-blue-200'
                                                      : 'bg-green-800/40 text-green-200';
                                            return (
                                                <span
                                                    key={t.id}
                                                    className={`rounded px-1.5 py-0.5 font-mono text-xs ${colorClass}`}
                                                    title={`source: ${t.source}`}
                                                >
                                                    {t.tag_key}/{t.tag_value}
                                                </span>
                                            );
                                        })}
                                    </div>
                                </div>
                            )}

                            {/* People */}
                            {people.length > 0 && (
                                <div className='pb-4'>
                                    <p className='mb-2 text-xs font-semibold tracking-wider text-white/40 uppercase'>
                                        People
                                    </p>
                                    <ul className='space-y-1'>
                                        {people.map((p) => (
                                            <li
                                                key={p.name}
                                                className='flex items-center justify-between text-white/80'
                                            >
                                                <span>{p.name}</span>
                                                {p.count > 1 && (
                                                    <span className='text-xs text-white/40'>{p.count} faces</span>
                                                )}
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            )}
        </>
    );
};

export default MetadataPanel;
