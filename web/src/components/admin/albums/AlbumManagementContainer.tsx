import React from 'react';
import { Link } from 'react-router-dom';
import { useAlbums } from '../../../api/swr/useAlbums';
import { Button } from '../../elements/Button';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../elements/Table';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { Can } from '../../elements/Can';
import { PlusIcon } from '@heroicons/react/20/solid';
import { formatDistanceToNow } from 'date-fns';
import PageContentBlock from '../../elements/PageContentBlock.tsx';
import { Heading } from '../../elements/Heading';
import { Text } from '../../elements/Text';

const AlbumManagementContainer: React.FC = () => {
    const { albums, isLoading, error } = useAlbums();

    if (isLoading) {
        return (
            <div className='flex h-64 items-center justify-center'>
                <LoadingSpinner />
            </div>
        );
    }

    if (error) {
        return <div className='text-center text-red-600'>Error loading albums: {error.message}</div>;
    }

    return (
        <PageContentBlock title={'Albums'} className='space-y-6'>
            <div className='mb-6 flex w-full flex-wrap items-end justify-between gap-4'>
                <div>
                    <Heading>Albums</Heading>
                    <Text className='mt-1'>Manage photo albums, their settings, and access control.</Text>
                </div>
                <Can permission='album.create'>
                    <Link to='/admin/albums/create'>
                        <Button>
                            <PlusIcon className='mr-2 h-4 w-4' />
                            Create Album
                        </Button>
                    </Link>
                </Can>
            </div>

            {albums.length === 0 ? (
                <div className='flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 p-10 text-center'>
                    <Heading level={4} className='text-lg font-semibold'>
                        No albums yet
                    </Heading>
                    <Text className='mt-2 text-sm text-gray-600'>
                        Create your first album to get started.
                    </Text>
                </div>
            ) : (
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableHeader>Name</TableHeader>
                            <TableHeader>Status</TableHeader>
                            <TableHeader>Created</TableHeader>
                            <TableHeader>Actions</TableHeader>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {albums.map((album) => (
                            <TableRow key={album.id}>
                                <TableCell>
                                    <div>
                                        <Link
                                            to={`/admin/albums/view/${album.id}`}
                                            className='font-medium hover:underline'
                                        >
                                            {album.name}
                                        </Link>
                                        {album.description && (
                                            <div className='text-sm text-gray-500'>{album.description}</div>
                                        )}
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <span
                                        className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                                            album.is_hidden
                                                ? 'bg-yellow-100 text-yellow-800'
                                                : 'bg-green-100 text-green-800'
                                        }`}
                                    >
                                        {album.is_hidden ? 'Hidden' : 'Visible'}
                                    </span>
                                </TableCell>
                                <TableCell>
                                    <div className='text-sm text-gray-500'>
                                        {formatDistanceToNow(new Date(album.created_at * 1000), { addSuffix: true })}
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <Can permission='album.edit.general'>
                                        <Button plain to={`/admin/albums/view/${album.id}`}>
                                            Edit
                                        </Button>
                                    </Can>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            )}

        </PageContentBlock>
    );
};

export default AlbumManagementContainer;
