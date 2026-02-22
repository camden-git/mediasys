import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Collection } from '../../types';
import { listPublicCollections } from '../../api/collections';

const CollectionsPage: React.FC = () => {
    const [collections, setCollections] = useState<Collection[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        listPublicCollections()
            .then(setCollections)
            .catch(() => setError('Failed to load collections.'))
            .finally(() => setIsLoading(false));
    }, []);

    if (isLoading) {
        return (
            <div className='flex items-center justify-center py-20'>
                <div className='h-8 w-8 animate-spin rounded-full border-4 border-gray-300 border-t-blue-600' />
            </div>
        );
    }

    if (error) {
        return <div className='py-20 text-center text-red-500'>{error}</div>;
    }

    return (
        <div className='mx-auto max-w-7xl px-4 py-8'>
            <h1 className='mb-6 text-2xl font-bold'>Collections</h1>
            {collections.length === 0 ? (
                <p className='text-gray-500'>No collections available.</p>
            ) : (
                <div className='grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3'>
                    {collections.map((c) => (
                        <Link
                            key={c.id}
                            to={`/collections/${c.slug}`}
                            className='group overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700 transition-shadow hover:shadow-md'
                        >
                            {c.banner_image_path && (
                                <div className='aspect-video overflow-hidden bg-gray-100 dark:bg-gray-800'>
                                    <img
                                        src={`/api/banners/${c.banner_image_path.split('/').pop()}`}
                                        alt={c.name}
                                        className='h-full w-full object-cover transition-transform group-hover:scale-105'
                                    />
                                </div>
                            )}
                            <div className='p-4'>
                                <h2 className='font-semibold'>{c.name}</h2>
                                {c.description && (
                                    <p className='mt-1 text-sm text-gray-500 dark:text-gray-400 line-clamp-2'>
                                        {c.description}
                                    </p>
                                )}
                                {c.filters && c.filters.length > 0 && (
                                    <div className='mt-2 flex flex-wrap gap-1'>
                                        {c.filters.slice(0, 3).map((f) => (
                                            <span
                                                key={f.id}
                                                className='rounded bg-gray-100 dark:bg-gray-700 px-1.5 py-0.5 font-mono text-xs text-gray-600 dark:text-gray-300'
                                            >
                                                {f.tag_key}/{f.tag_value}
                                            </span>
                                        ))}
                                        {c.filters.length > 3 && (
                                            <span className='text-xs text-gray-400'>+{c.filters.length - 3} more</span>
                                        )}
                                    </div>
                                )}
                            </div>
                        </Link>
                    ))}
                </div>
            )}
        </div>
    );
};

export default CollectionsPage;
