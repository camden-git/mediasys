import React from 'react';
import { Link } from 'react-router-dom';
import { getBannerUrl } from '../../api.ts';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import { useCollections } from '../../hooks/useCollections.ts';
import { useDocumentTitle } from '../../hooks/useDocumentTitle.ts';

const CollectionsPage: React.FC = () => {
    useDocumentTitle('Collections');
    const { collections, isLoading, error } = useCollections();

    if (isLoading) return <LoadingSpinner />;
    if (error) return <p className='p-8 text-red-500'>{error.message || 'Failed to load collections'}</p>;

    return (
        <div className='mx-auto max-w-6xl px-4 py-12'>
            <h1 className='mb-8 text-3xl font-bold text-gray-950 dark:text-white'>Collections</h1>
            {collections.length === 0 && <p className='text-gray-500 dark:text-gray-400'>No collections available.</p>}
            <div className='grid gap-6 sm:grid-cols-2 lg:grid-cols-3'>
                {collections.map((c) => (
                    <Link
                        key={c.id}
                        to={`/collections/${c.slug}`}
                        className='group relative overflow-hidden rounded-xl bg-gray-100 transition-shadow hover:shadow-lg dark:bg-gray-800'
                    >
                        {c.banners?.[0] ? (
                            <img
                                src={getBannerUrl(c.banners[0])}
                                alt=''
                                loading='lazy'
                                decoding='async'
                                className='h-40 w-full object-cover opacity-70 transition-opacity group-hover:opacity-90'
                            />
                        ) : (
                            <div className='h-40 w-full bg-gradient-to-br from-gray-200 to-gray-300 dark:from-gray-700 dark:to-gray-600' />
                        )}
                        <div className='p-4'>
                            <h2 className='text-lg font-semibold text-gray-950 dark:text-white'>{c.name}</h2>
                            {c.description && (
                                <p className='mt-1 line-clamp-2 text-sm text-gray-500 dark:text-gray-400'>
                                    {c.description}
                                </p>
                            )}
                            {c.filters && c.filters.length > 0 && (
                                <p className='mt-2 text-xs text-gray-400 dark:text-gray-500'>
                                    {c.filters.length} filter{c.filters.length !== 1 ? 's' : ''}
                                </p>
                            )}
                        </div>
                    </Link>
                ))}
            </div>
        </div>
    );
};

export default CollectionsPage;
