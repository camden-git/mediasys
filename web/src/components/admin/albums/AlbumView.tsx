import React from 'react';
import { useStoreState } from '../../../store/hooks';
import { formatDistanceToNow } from 'date-fns';
import { getBannerUrl } from '../../../api.ts';

const AlbumView: React.FC = () => {
    const album = useStoreState((state) => state.albumContext.data!);

    return (
        <div className='space-y-6'>
            <div className='flex items-center justify-between'>
                <div className='flex items-center space-x-4'>
                    <h1 className='text-2xl font-bold text-gray-900'>{album.name}</h1>
                </div>
            </div>

            <div className='rounded-lg bg-white shadow'>
                <div className='border-b border-gray-200 px-6 py-4'>
                    <h2 className='text-lg font-medium text-gray-900'>Album Details</h2>
                </div>
                <div className='px-6 py-4'>
                    <dl className='grid grid-cols-1 gap-x-4 gap-y-6 sm:grid-cols-2'>
                        <div>
                            <dt className='text-sm font-medium text-gray-500'>Name</dt>
                            <dd className='mt-1 text-sm text-gray-900'>{album.name}</dd>
                        </div>
                        <div>
                            <dt className='text-sm font-medium text-gray-500'>Slug</dt>
                            <dd className='mt-1 text-sm text-gray-900'>
                                <code className='rounded bg-gray-100 px-2 py-1'>{album.slug}</code>
                            </dd>
                        </div>
                        <div>
                            <dt className='text-sm font-medium text-gray-500'>Folder Path</dt>
                            <dd className='mt-1 text-sm text-gray-900'>{album.folder_path}</dd>
                        </div>
                        <div>
                            <dt className='text-sm font-medium text-gray-500'>Status</dt>
                            <dd className='mt-1'>
                                <span
                                    className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                                        album.is_hidden
                                            ? 'bg-yellow-100 text-yellow-800'
                                            : 'bg-green-100 text-green-800'
                                    }`}
                                >
                                    {album.is_hidden ? 'Hidden' : 'Visible'}
                                </span>
                            </dd>
                        </div>
                        {album.description && (
                            <div className='sm:col-span-2'>
                                <dt className='text-sm font-medium text-gray-500'>Description</dt>
                                <dd className='mt-1 text-sm text-gray-900'>{album.description}</dd>
                            </div>
                        )}
                        {album.location && (
                            <div>
                                <dt className='text-sm font-medium text-gray-500'>Location</dt>
                                <dd className='mt-1 text-sm text-gray-900'>{album.location}</dd>
                            </div>
                        )}
                        <div>
                            <dt className='text-sm font-medium text-gray-500'>Sort Order</dt>
                            <dd className='mt-1 text-sm text-gray-900 capitalize'>{album.sort_order}</dd>
                        </div>
                        <div>
                            <dt className='text-sm font-medium text-gray-500'>Created</dt>
                            <dd className='mt-1 text-sm text-gray-900'>
                                {formatDistanceToNow(new Date(album.created_at * 1000), { addSuffix: true })}
                            </dd>
                        </div>
                        <div>
                            <dt className='text-sm font-medium text-gray-500'>Last Updated</dt>
                            <dd className='mt-1 text-sm text-gray-900'>
                                {formatDistanceToNow(new Date(album.updated_at * 1000), { addSuffix: true })}
                            </dd>
                        </div>
                    </dl>
                </div>
            </div>

            {album.banner_image_path && (
                <div className='rounded-lg bg-white shadow'>
                    <div className='border-b border-gray-200 px-6 py-4'>
                        <h2 className='text-lg font-medium text-gray-900'>Banner Image</h2>
                    </div>
                    <div className='px-6 py-4'>
                        <img
                            src={getBannerUrl(album.banner_image_path)}
                            alt={`Banner for ${album.name}`}
                            className='h-auto max-w-md rounded-lg'
                        />
                    </div>
                </div>
            )}

            <div className='rounded-lg bg-white shadow'>
                <div className='border-b border-gray-200 px-6 py-4'>
                    <h2 className='text-lg font-medium text-gray-900'>Zip Archive</h2>
                </div>
                <div className='px-6 py-4'>
                    <dl className='grid grid-cols-1 gap-x-4 gap-y-6 sm:grid-cols-2'>
                        <div>
                            <dt className='text-sm font-medium text-gray-500'>Status</dt>
                            <dd className='mt-1'>
                                <span
                                    className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                                        album.zip_status === 'ready'
                                            ? 'bg-green-100 text-green-800'
                                            : album.zip_status === 'generating'
                                              ? 'bg-yellow-100 text-yellow-800'
                                              : 'bg-gray-100 text-gray-800'
                                    }`}
                                >
                                    {album.zip_status || 'Not generated'}
                                </span>
                            </dd>
                        </div>
                        {album.zip_size && (
                            <div>
                                <dt className='text-sm font-medium text-gray-500'>Size</dt>
                                <dd className='mt-1 text-sm text-gray-900'>
                                    {(album.zip_size / 1024 / 1024).toFixed(2)} MB
                                </dd>
                            </div>
                        )}
                        {album.zip_last_generated_at && (
                            <div>
                                <dt className='text-sm font-medium text-gray-500'>Last Generated</dt>
                                <dd className='mt-1 text-sm text-gray-900'>
                                    {formatDistanceToNow(new Date(album.zip_last_generated_at * 1000), {
                                        addSuffix: true,
                                    })}
                                </dd>
                            </div>
                        )}
                        {album.zip_error && (
                            <div className='sm:col-span-2'>
                                <dt className='text-sm font-medium text-gray-500'>Error</dt>
                                <dd className='mt-1 text-sm text-red-600'>{album.zip_error}</dd>
                            </div>
                        )}
                    </dl>
                </div>
            </div>
        </div>
    );
};

export default AlbumView;
