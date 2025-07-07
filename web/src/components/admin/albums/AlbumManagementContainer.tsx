import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { useAlbums } from '../../../api/swr/useAlbums';
import { useStoreActions } from '../../../store/hooks';
import { AdminAlbumResponse } from '../../../api/admin/albums';
import { Button } from '../../elements/Button';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../elements/Table';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { Dialog, DialogActions, DialogDescription, DialogTitle } from '../../elements/Dialog';
import { Can } from '../../elements/Can';
import { PlusIcon, PencilIcon, TrashIcon, EyeIcon } from '@heroicons/react/20/solid';
import { formatDistanceToNow } from 'date-fns';
import PageContentBlock from '../../elements/PageContentBlock.tsx';

const AlbumManagementContainer: React.FC = () => {
    const { albums, isLoading, error, mutate } = useAlbums();
    const { deleteAlbum } = useStoreActions((actions) => actions.adminAlbums);
    const { addFlash } = useStoreActions((actions) => actions.ui);
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
    const [albumToDelete, setAlbumToDelete] = useState<AdminAlbumResponse | null>(null);

    const handleDeleteClick = (album: AdminAlbumResponse) => {
        setAlbumToDelete(album);
        setDeleteDialogOpen(true);
    };

    const handleDeleteConfirm = () => {
        if (albumToDelete) {
            deleteAlbum({
                id: albumToDelete.id,
                addFlash,
                onSuccess: () => {
                    mutate();
                    setDeleteDialogOpen(false);
                    setAlbumToDelete(null);
                },
            });
        }
    };

    const handleDeleteCancel = () => {
        setDeleteDialogOpen(false);
        setAlbumToDelete(null);
    };

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
            <div className='flex items-center justify-between'>
                <h1 className='text-2xl font-bold text-gray-900'>Album Management</h1>
                <Can permission='album.create'>
                    <Link to='/admin/albums/create'>
                        <Button>
                            <PlusIcon className='mr-2 h-4 w-4' />
                            Create Album
                        </Button>
                    </Link>
                </Can>
            </div>

            <Table>
                <TableHead>
                    <TableRow>
                        <TableHeader>Name</TableHeader>
                        <TableHeader>Slug</TableHeader>
                        <TableHeader>Folder Path</TableHeader>
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
                                    <div className='font-medium text-gray-900'>{album.name}</div>
                                    {album.description && (
                                        <div className='text-sm text-gray-500'>{album.description}</div>
                                    )}
                                </div>
                            </TableCell>
                            <TableCell>
                                <code className='rounded bg-gray-100 px-2 py-1 text-sm'>{album.slug}</code>
                            </TableCell>
                            <TableCell>
                                <div className='max-w-xs truncate text-sm text-gray-600'>{album.folder_path}</div>
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
                                <div className='flex space-x-2'>
                                    <Can permission='album.view'>
                                        <Link
                                            to={`/admin/albums/${album.id}`}
                                            className='text-blue-600 hover:text-blue-900'
                                        >
                                            <EyeIcon className='h-4 w-4' />
                                        </Link>
                                    </Can>
                                    <Can permission='album.edit.general'>
                                        <Link
                                            to={`/admin/albums/${album.id}/edit`}
                                            className='text-gray-600 hover:text-gray-900'
                                        >
                                            <PencilIcon className='h-4 w-4' />
                                        </Link>
                                    </Can>
                                    <Can permission='album.delete'>
                                        <button
                                            onClick={() => handleDeleteClick(album)}
                                            className='text-red-600 hover:text-red-900'
                                        >
                                            <TrashIcon className='h-4 w-4' />
                                        </button>
                                    </Can>
                                </div>
                            </TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table>

            <Dialog open={deleteDialogOpen} onClose={handleDeleteCancel}>
                <DialogTitle>Delete Album</DialogTitle>
                <DialogDescription>
                    Are you sure you want to delete "{albumToDelete?.name}"? This action cannot be undone.
                </DialogDescription>
                <DialogActions>
                    <Button plain onClick={handleDeleteCancel}>
                        Cancel
                    </Button>
                    <Button onClick={handleDeleteConfirm} type={'submit'}>
                        Delete
                    </Button>
                </DialogActions>
            </Dialog>
        </PageContentBlock>
    );
};

export default AlbumManagementContainer;
