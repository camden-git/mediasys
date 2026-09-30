import React from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { getAlbums } from '../../api/albums';
import { getBannerUrl } from '../../api/media';
import { useGroups } from '../../hooks/useGroups.ts';
import { useDocumentTitle } from '../../hooks/useDocumentTitle.ts';
import { useAuthStore } from '../../store/useAuthStore.ts';
import { Can } from '../elements/Can.tsx';
import { queryKeys } from '../../lib/queryKeys.ts';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';

const browseLinks = [
    { label: 'Groups', to: '/groups' },
    { label: 'Collections', to: '/collections' },
    { label: 'People', to: '/people' },
];

const navLinkClass =
    'text-zinc-700 underline decoration-zinc-400 hover:decoration-zinc-700 dark:text-zinc-300 dark:decoration-zinc-600 dark:hover:decoration-zinc-300';

const AlbumList: React.FC = () => {
    useDocumentTitle('Albums');
    const albumsQuery = useQuery({ queryKey: queryKeys.albums.list(), queryFn: ({ signal }) => getAlbums(signal) });
    const { groups, isLoading: groupsLoading, error: groupsError } = useGroups();
    const isAuthenticated = useAuthStore((s) => s.isAuthenticated());
    const isInitializing = useAuthStore((s) => s.isInitializing);
    const albums = albumsQuery.data ?? [];
    const isLoading = albumsQuery.isLoading || groupsLoading;
    const fetchError = albumsQuery.error ?? groupsError;
    const error = fetchError ? fetchError.message || 'Failed to fetch albums' : null;

    return (
        <div className='container mx-auto p-4'>
            <h1 className='mb-4 text-3xl font-bold text-gray-950 dark:text-white'>Mediasys</h1>
            <nav aria-label='Browse' className='mb-8 flex gap-4 text-sm font-medium'>
                {browseLinks.map(({ label, to }) => (
                    <Link key={to} to={to} className={navLinkClass}>
                        {label}
                    </Link>
                ))}
                {!isInitializing && (
                    <Link to={isAuthenticated ? '/admin' : '/auth/login'} className={`${navLinkClass} ml-auto`}>
                        {isAuthenticated ? 'Admin' : 'Sign in'}
                    </Link>
                )}
            </nav>
            {error && (
                <>
                    <div className='mx-auto flex justify-center'>
                        <ExclamationTriangleIcon className='mr-3 h-6 w-6 text-red-500' />
                        <p className='my-auto font-light text-red-400'>Failed to get albums</p>
                    </div>
                    <p className='m-auto ml-3 justify-center text-center font-light text-gray-600 dark:text-gray-400'>
                        {error}
                    </p>
                </>
            )}
            {isLoading && (
                <div className='mx-auto flex justify-center'>
                    <span className='mr-3'>
                        <LoadingSpinner />
                    </span>
                    <p className='my-auto font-light text-gray-600 dark:text-gray-400'>Loading albums</p>
                </div>
            )}

            {!isLoading && !error && (
                <>
                    {groups.length > 0 && (
                        <div className='mb-10'>
                            <h2 className='mb-4 text-2xl font-bold text-gray-950 dark:text-white'>Groups</h2>
                            <div className='grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4'>
                                {groups.map((group) => (
                                    <Link
                                        key={group.id}
                                        to={`/groups/${group.slug}`}
                                        className='group relative overflow-hidden rounded-lg bg-white shadow transition-shadow duration-200 hover:shadow-md dark:bg-zinc-800'
                                    >
                                        {group.banner_image_path ? (
                                            <img
                                                src={getBannerUrl(group.banner_image_path)}
                                                alt=''
                                                className='h-32 w-full object-cover opacity-80 transition-opacity group-hover:opacity-100'
                                            />
                                        ) : (
                                            <div className='h-32 w-full bg-gradient-to-br from-gray-200 to-gray-300 dark:from-gray-700 dark:to-gray-600' />
                                        )}
                                        <div className='p-4'>
                                            <h3 className='mb-1 truncate font-semibold text-gray-950 dark:text-white'>
                                                {group.name}
                                            </h3>
                                            {group.description && (
                                                <p className='line-clamp-2 text-xs text-gray-500 italic dark:text-gray-400'>
                                                    {group.description}
                                                </p>
                                            )}
                                            {group.albums && group.albums.length > 0 && (
                                                <p className='mt-1 text-xs text-gray-400'>
                                                    {group.albums.length} album{group.albums.length !== 1 ? 's' : ''}
                                                </p>
                                            )}
                                        </div>
                                    </Link>
                                ))}
                            </div>
                        </div>
                    )}

                    <h2 className='mb-4 text-2xl font-bold text-gray-950 dark:text-white'>Albums</h2>
                    <div className='grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4'>
                        {albums.length === 0 && (
                            <p className='col-span-full text-center text-gray-500 dark:text-gray-400'>
                                No albums found.
                                <Can permission='album.create'>
                                    {' '}
                                    Create one in the{' '}
                                    <Link to='/admin/albums/create' className='underline'>
                                        admin area
                                    </Link>
                                    .
                                </Can>
                            </p>
                        )}
                        {albums.map((album) => (
                            <Link
                                key={album.id}
                                to={`/album/${album.slug}`}
                                className='block overflow-hidden rounded-lg bg-white shadow transition-shadow duration-200 hover:shadow-md dark:bg-zinc-800'
                            >
                                <div className='p-4'>
                                    <h3 className='mb-2 truncate text-xl font-semibold text-gray-950 dark:text-white'>
                                        {album.name}
                                    </h3>
                                    {album.description && (
                                        <p className='truncate text-xs text-gray-500 italic dark:text-gray-400'>
                                            {album.description}
                                        </p>
                                    )}
                                </div>
                            </Link>
                        ))}
                    </div>
                </>
            )}
        </div>
    );
};

export default AlbumList;
