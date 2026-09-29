import React from 'react';
import { useAlbumData } from '../../../store/albumContextHooks';
import { formatDistanceToNow } from 'date-fns';
import { getBannerUrl } from '../../../api.ts';
import { Heading } from '../../elements/Heading';
import { DescriptionList, DescriptionItem } from '../../elements/DescriptionList';

const zipStatusLabels: Record<string, string> = {
    notRequired: 'Not generated',
    pending: 'Pending',
    processing: 'Generating',
    done: 'Ready',
    error: 'Error',
};

const AlbumView: React.FC = () => {
    const album = useAlbumData();

    return (
        <div className='space-y-8'>
            <div>
                <Heading level={2}>Album Details</Heading>
                <DescriptionList className='mt-4'>
                    <DescriptionItem term='Name' details={album.name} />
                    <DescriptionItem
                        term='Slug'
                        details={<code className='rounded bg-gray-100 px-2 py-1 dark:bg-zinc-800'>{album.slug}</code>}
                    />
                    <DescriptionItem term='Folder Path' details={album.folder_path} />
                    <DescriptionItem
                        term='Status'
                        details={
                            <span
                                className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                                    album.is_hidden ? 'bg-yellow-100 text-yellow-800' : 'bg-green-100 text-green-800'
                                }`}
                            >
                                {album.is_hidden ? 'Hidden' : 'Visible'}
                            </span>
                        }
                    />
                    {album.description && <DescriptionItem term='Description' details={album.description} />}
                    {album.location && <DescriptionItem term='Location' details={album.location} />}
                    <DescriptionItem
                        term='Sort Order'
                        details={<span className='capitalize'>{album.sort_order}</span>}
                    />
                    <DescriptionItem
                        term='Created'
                        details={formatDistanceToNow(new Date(album.created_at * 1000), { addSuffix: true })}
                    />
                    <DescriptionItem
                        term='Last Updated'
                        details={formatDistanceToNow(new Date(album.updated_at * 1000), { addSuffix: true })}
                    />
                </DescriptionList>
            </div>

            {album.banners && album.banners.length > 0 && (
                <div>
                    <Heading level={2}>Banner Images</Heading>
                    <div className='mt-4 flex flex-wrap gap-3'>
                        {album.banners.map((b) => (
                            <img
                                key={b.id}
                                src={getBannerUrl(b.image_path)}
                                alt={`Banner for ${album.name}`}
                                className='h-auto max-w-xs rounded-lg'
                            />
                        ))}
                    </div>
                </div>
            )}

            <div>
                <Heading level={2}>Zip Archive</Heading>
                <DescriptionList className='mt-4'>
                    <DescriptionItem
                        term='Status'
                        details={
                            <span
                                className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                                    album.zip_status === 'done'
                                        ? 'bg-green-100 text-green-800'
                                        : album.zip_status === 'pending' || album.zip_status === 'processing'
                                          ? 'bg-yellow-100 text-yellow-800'
                                          : album.zip_status === 'error'
                                            ? 'bg-red-100 text-red-800'
                                            : 'bg-gray-100 text-gray-800'
                                }`}
                            >
                                {zipStatusLabels[album.zip_status] ?? 'Not generated'}
                            </span>
                        }
                    />
                    {album.zip_size && (
                        <DescriptionItem term='Size' details={`${(album.zip_size / 1024 / 1024).toFixed(2)} MB`} />
                    )}
                    {album.zip_last_generated_at && (
                        <DescriptionItem
                            term='Last Generated'
                            details={formatDistanceToNow(new Date(album.zip_last_generated_at * 1000), {
                                addSuffix: true,
                            })}
                        />
                    )}
                    {album.zip_error && (
                        <DescriptionItem
                            term='Error'
                            details={<span className='text-red-600'>{album.zip_error}</span>}
                        />
                    )}
                </DescriptionList>
            </div>
        </div>
    );
};

export default AlbumView;
