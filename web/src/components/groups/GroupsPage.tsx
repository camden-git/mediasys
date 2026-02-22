import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { AlbumGroup } from '../../types.ts';
import { getBannerUrl, getGroups } from '../../api.ts';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';

const GroupsPage: React.FC = () => {
    const [groups, setGroups] = useState<AlbumGroup[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();
        (async () => {
            try {
                const data = await getGroups(controller.signal);
                setGroups(data);
            } catch (err: any) {
                if (err.name !== 'AbortError') {
                    setError(err.message || 'Failed to load groups');
                }
            } finally {
                setIsLoading(false);
            }
        })();
        return () => controller.abort();
    }, []);

    if (isLoading) return <LoadingSpinner />;
    if (error) return <p className='p-8 text-red-500'>{error}</p>;

    return (
        <div className='mx-auto max-w-6xl px-4 py-12'>
            <h1 className='mb-8 text-3xl font-bold text-gray-950 dark:text-white'>Event Collections</h1>
            {groups.length === 0 && <p className='text-gray-500 dark:text-gray-400'>No collections yet.</p>}
            <div className='grid gap-6 sm:grid-cols-2 lg:grid-cols-3'>
                {groups.map((group) => (
                    <Link
                        key={group.id}
                        to={`/groups/${group.slug}`}
                        className='group relative overflow-hidden rounded-xl bg-gray-100 transition-shadow hover:shadow-lg dark:bg-gray-800'
                    >
                        {group.banner_image_path && (
                            <img
                                src={getBannerUrl(group.banner_image_path)}
                                alt=''
                                className='h-40 w-full object-cover opacity-70 transition-opacity group-hover:opacity-90'
                            />
                        )}
                        {!group.banner_image_path && (
                            <div className='h-40 w-full bg-gradient-to-br from-gray-200 to-gray-300 dark:from-gray-700 dark:to-gray-600' />
                        )}
                        <div className='p-4'>
                            <h2 className='text-lg font-semibold text-gray-950 dark:text-white'>{group.name}</h2>
                            {group.description && (
                                <p className='mt-1 line-clamp-2 text-sm text-gray-500 dark:text-gray-400'>
                                    {group.description}
                                </p>
                            )}
                            {group.albums && group.albums.length > 0 && (
                                <p className='mt-2 text-xs text-gray-400 dark:text-gray-500'>
                                    {group.albums.length} album{group.albums.length !== 1 ? 's' : ''}
                                </p>
                            )}
                        </div>
                    </Link>
                ))}
            </div>
        </div>
    );
};

export default GroupsPage;
