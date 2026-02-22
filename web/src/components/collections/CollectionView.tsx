import React, { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { Collection } from '../../types.ts';
import { getPublicCollection } from '../../api/collections.ts';
import { getBannerUrl } from '../../api.ts';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import { PhotoIcon } from '@heroicons/react/16/solid';

const CollectionView: React.FC = () => {
    const { slug } = useParams<{ slug: string }>();
    const [collection, setCollection] = useState<Collection | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (!slug) return;
        setIsLoading(true);
        setError(null);
        getPublicCollection(slug)
            .then(setCollection)
            .catch((err) => setError(err.message || 'Failed to load collection'))
            .finally(() => setIsLoading(false));
    }, [slug]);

    if (isLoading) return <LoadingSpinner />;
    if (error) return <p className='p-8 text-red-500'>{error}</p>;
    if (!collection) return null;

    return (
        <div className='mx-auto max-w-6xl'>
            {/* Hero banner */}
            <div className='relative h-64 overflow-hidden rounded-2xl'>
                {collection.banner_image_path ? (
                    <img
                        src={getBannerUrl(collection.banner_image_path)}
                        alt=''
                        className='h-full w-full object-cover opacity-60'
                    />
                ) : (
                    <div className='h-full w-full bg-gradient-to-br from-gray-200 to-gray-300 dark:from-gray-700 dark:to-gray-600' />
                )}
                <div className='absolute inset-0 flex flex-col justify-end p-8'>
                    <h1 className='text-4xl font-bold text-white drop-shadow-md'>{collection.name}</h1>
                    {collection.description && (
                        <p className='mt-2 max-w-lg text-white/80 drop-shadow'>{collection.description}</p>
                    )}
                </div>
            </div>

            {/* Filter tags */}
            {collection.filters && collection.filters.length > 0 && (
                <div className='mt-4 flex flex-wrap gap-2 px-1'>
                    {collection.filters.map((f) => (
                        <span
                            key={f.id}
                            className='rounded-full bg-gray-100 px-3 py-1 font-mono text-xs text-gray-600 dark:bg-gray-800 dark:text-gray-300'
                        >
                            {f.tag_key} = {f.tag_value}
                        </span>
                    ))}
                </div>
            )}

            {/* Action button */}
            <div className='mt-6 flex gap-3 px-1'>
                <Link
                    to={`/collections/${slug}/photos`}
                    className='inline-flex items-center gap-x-2 rounded-full bg-gray-950 px-4 py-1.5 text-sm font-semibold text-white hover:bg-gray-800 dark:bg-gray-700 dark:hover:bg-gray-600'
                >
                    <PhotoIcon className='size-3.5' />
                    View all photos
                </Link>
            </div>
        </div>
    );
};

export default CollectionView;
