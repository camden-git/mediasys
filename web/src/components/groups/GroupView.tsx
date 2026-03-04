import React from 'react';
import { Link, useParams } from 'react-router-dom';
import { getBannerUrl } from '../../api.ts';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import { SparklesIcon, PhotoIcon } from '@heroicons/react/16/solid';
import { useGroup } from '../../hooks/useGroups.ts';

const GroupView: React.FC = () => {
    const { slug } = useParams<{ slug: string }>();
    const { group, isLoading, error } = useGroup(slug);

    if (isLoading) return <LoadingSpinner />;
    if (error) return <p className='p-8 text-red-500'>{error.message || 'Failed to load group'}</p>;
    if (!group) return null;

    return (
        <div className='mx-auto max-w-6xl'>
            {/* Hero banner */}
            <div className='relative h-64 overflow-hidden rounded-2xl'>
                {group.banner_image_path ? (
                    <img
                        src={getBannerUrl(group.banner_image_path)}
                        alt=''
                        decoding='async'
                        className='h-full w-full object-cover opacity-60'
                    />
                ) : (
                    <div className='h-full w-full bg-gradient-to-br from-gray-200 to-gray-300 dark:from-gray-700 dark:to-gray-600' />
                )}
                <div className='absolute inset-0 flex flex-col justify-end p-8'>
                    <h1 className='text-4xl font-bold text-white drop-shadow-md'>{group.name}</h1>
                    {group.description && (
                        <p className='mt-2 max-w-lg text-white/80 drop-shadow'>{group.description}</p>
                    )}
                </div>
            </div>

            {/* Action buttons */}
            <div className='mt-6 flex gap-3 px-1'>
                <Link
                    to={`/groups/${slug}/photos`}
                    className='inline-flex items-center gap-x-2 rounded-full bg-gray-950 px-4 py-1.5 text-sm font-semibold text-white hover:bg-gray-800 dark:bg-gray-700 dark:hover:bg-gray-600'
                >
                    <PhotoIcon className='size-3.5' />
                    View all photos
                </Link>
                <Link
                    to={`/groups/${slug}/photos?highlights=1`}
                    className='inline-flex items-center gap-x-2 rounded-full bg-yellow-400 px-4 py-1.5 text-sm font-semibold text-gray-950 hover:bg-yellow-300'
                >
                    <SparklesIcon className='size-3.5' />
                    Highlights only
                </Link>
            </div>

            {/* Album cards */}
            {group.albums && group.albums.length > 0 && (
                <div className='mt-10 px-1'>
                    <h2 className='mb-4 text-xl font-semibold text-gray-950 dark:text-white'>Albums</h2>
                    <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
                        {group.albums.map((album) => (
                            <Link
                                key={album.id}
                                to={`/album/${album.slug}`}
                                className='group overflow-hidden rounded-xl bg-gray-100 transition-shadow hover:shadow-md dark:bg-gray-800'
                            >
                                {album.banners?.[0] ? (
                                    <img
                                        src={getBannerUrl(album.banners[0])}
                                        alt=''
                                        loading='lazy'
                                        decoding='async'
                                        className='h-32 w-full object-cover opacity-70 transition-opacity group-hover:opacity-90'
                                    />
                                ) : (
                                    <div className='h-32 w-full bg-gradient-to-br from-gray-200 to-gray-300 dark:from-gray-700 dark:to-gray-600' />
                                )}
                                <div className='p-3'>
                                    <p className='font-medium text-gray-950 dark:text-white'>{album.name}</p>
                                    {album.location && (
                                        <p className='mt-0.5 text-xs text-gray-400 dark:text-gray-500'>
                                            {album.location}
                                        </p>
                                    )}
                                </div>
                            </Link>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
};

export default GroupView;
