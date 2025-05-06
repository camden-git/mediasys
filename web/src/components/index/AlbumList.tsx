import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getAlbums } from '../../api.ts';
import { Album } from '../../types.ts';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';

const AlbumList: React.FC = () => {
    const [albums, setAlbums] = useState<Album[]>([]);
    const [isLoading, setIsLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchAlbumData = async () => {
            setIsLoading(true);
            setError(null);
            try {
                const fetchedAlbums = await getAlbums();
                setAlbums(fetchedAlbums);
            } catch (err: any) {
                console.error('Failed to fetch albums:', err);
                setError(err.message || 'Failed to fetch albums');
            } finally {
                setIsLoading(false);
            }
        };

        fetchAlbumData();
    }, []);

    return (
        <div className='container mx-auto p-4'>
            <h1 className='mb-6 text-center text-3xl font-bold'>Albums</h1>
            {error && (
                <>
                    <div className='mx-auto flex justify-center'>
                        <ExclamationTriangleIcon className='mr-3 h-6 w-6 text-red-500' />
                        <p className='font-300 my-auto text-red-400'>Failed to get albums</p>
                    </div>
                    <p className='m-auto ml-3 justify-center text-center font-light text-gray-600'>{error}</p>
                </>
            )}
            {isLoading && (
                <div className='mx-auto flex justify-center'>
                    <svg
                        className='mr-3 h-6 w-6 animate-spin text-white'
                        xmlns='http://www.w3.org/2000/svg'
                        fill='none'
                        viewBox='0 0 24 24'
                    >
                        <circle
                            className='text-gray-400'
                            cx='12'
                            cy='12'
                            r='10'
                            stroke='currentColor'
                            strokeWidth='4'
                        ></circle>
                        <path
                            className='opacity-50'
                            fill='currentColor'
                            d='M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z'
                        ></path>
                    </svg>
                    <p className='my-auto font-light text-gray-600'>Loading albums</p>
                </div>
            )}

            {!isLoading && !error && (
                <div className='grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4'>
                    {albums.length === 0 && <p className='col-span-full text-center text-gray-500'>No albums found.</p>}
                    {albums.map((album) => (
                        <Link
                            key={album.id}
                            to={`/album/${album.slug}`}
                            className='block overflow-hidden rounded-lg bg-white shadow transition-shadow duration-200 hover:shadow-md'
                        >
                            <div className='p-4'>
                                <h2 className='mb-2 truncate text-xl font-semibold'>{album.name}</h2>
                                <p className='mb-1 text-sm text-gray-600'>/{album.folder_path}</p>
                                {album.description && (
                                    <p className='truncate text-xs text-gray-500 italic'>{album.description}</p>
                                )}
                            </div>
                        </Link>
                    ))}
                </div>
            )}
        </div>
    );
};

export default AlbumList;
