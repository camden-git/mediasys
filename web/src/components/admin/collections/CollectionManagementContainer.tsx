import React, { useCallback, useEffect, useState } from 'react';
import { Heading } from '../../elements/Heading';
import PageContentBlock from '../../elements/PageContentBlock.tsx';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../elements/Table';
import { Collection } from '../../../types';
import { Button } from '../../elements/Button';
import CreateCollectionForm from './CreateCollectionForm';
import EditCollectionForm from './EditCollectionForm';
import { deleteCollection, listCollections } from '../../../api/admin/collections';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import { Text } from '../../elements/Text.tsx';
import { Dropdown, DropdownButton, DropdownItem, DropdownMenu, DropdownSeparator } from '../../elements/Dropdown';
import { EllipsisHorizontalIcon } from '@heroicons/react/20/solid';
import { Dialog, DialogActions, DialogDescription, DialogTitle } from '../../elements/Dialog';

const CollectionManagementContainer: React.FC = () => {
    const [collections, setCollections] = useState<Collection[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const { clearFlashes, clearAndAddHttpError, addFlash } = useFlash();
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [collectionForEdit, setCollectionForEdit] = useState<Collection | null>(null);
    const [collectionForDelete, setCollectionForDelete] = useState<Collection | null>(null);

    const fetchCollections = useCallback(async () => {
        setIsLoading(true);
        clearFlashes('collections');
        try {
            const data = await listCollections();
            setCollections(data);
        } catch (error: any) {
            clearAndAddHttpError({ error, key: 'collections' });
        } finally {
            setIsLoading(false);
        }
    }, [clearFlashes, clearAndAddHttpError]);

    useEffect(() => {
        fetchCollections();
    }, [fetchCollections]);

    const confirmDelete = useCallback(async () => {
        if (!collectionForDelete) return;
        try {
            await deleteCollection(collectionForDelete.id);
            setCollections((prev) => prev.filter((c) => c.id !== collectionForDelete.id));
            addFlash({
                key: 'collections',
                type: 'success',
                title: 'Collection Removed',
                message: `Collection "${collectionForDelete.name}" has been deleted.`,
            });
        } catch (error: any) {
            clearAndAddHttpError({ error, key: 'collections' });
        } finally {
            setCollectionForDelete(null);
        }
    }, [collectionForDelete, addFlash, clearAndAddHttpError]);

    if (isLoading) {
        return <LoadingSpinner />;
    }

    return (
        <PageContentBlock title='Collection Management'>
            <div className='mb-6 flex w-full flex-wrap items-end justify-between gap-4'>
                <div>
                    <Heading>Collections</Heading>
                    <Text className='mt-1 text-sm text-gray-600 dark:text-gray-300'>
                        Collections are virtual, tag-filter-based photo groupings that work across albums.
                    </Text>
                </div>
                <Button onClick={() => setShowCreateModal(true)}>Create Collection</Button>
            </div>

            <FlashMessageRender byKey='collections' className='mb-4' />

            <CreateCollectionForm
                isOpen={showCreateModal}
                onClose={() => setShowCreateModal(false)}
                onCreated={fetchCollections}
            />
            <EditCollectionForm
                key={collectionForEdit?.id ?? 'edit-collection'}
                isOpen={!!collectionForEdit}
                onClose={() => setCollectionForEdit(null)}
                onUpdated={fetchCollections}
                collection={collectionForEdit ?? undefined}
            />

            {collections.length === 0 ? (
                <div className='flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 p-10 text-center'>
                    <Heading level={4} className='text-lg font-semibold'>
                        No collections yet
                    </Heading>
                    <Text className='mt-2 text-sm text-gray-600'>
                        Create a collection to group photos by tag filters across albums.
                    </Text>
                </div>
            ) : (
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableHeader>Name</TableHeader>
                            <TableHeader>Slug</TableHeader>
                            <TableHeader>Public</TableHeader>
                            <TableHeader>Filters</TableHeader>
                            <TableHeader>Actions</TableHeader>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {collections.map((c) => (
                            <TableRow key={c.id}>
                                <TableCell className='font-medium'>{c.name}</TableCell>
                                <TableCell>
                                    <span className='font-mono text-sm text-gray-500'>{c.slug}</span>
                                </TableCell>
                                <TableCell>
                                    <span className='text-sm'>{c.is_public ? 'Yes' : 'No'}</span>
                                </TableCell>
                                <TableCell>
                                    <span className='text-sm'>{c.filters?.length ?? 0}</span>
                                </TableCell>
                                <TableCell>
                                    <Dropdown>
                                        <DropdownButton plain aria-label='Actions'>
                                            <EllipsisHorizontalIcon className='size-5' />
                                        </DropdownButton>
                                        <DropdownMenu>
                                            <DropdownItem onClick={() => setCollectionForEdit(c)}>Edit</DropdownItem>
                                            <DropdownSeparator />
                                            <DropdownItem
                                                onClick={() => setCollectionForDelete(c)}
                                                className='text-red-600 data-[focus]:bg-red-500'
                                            >
                                                Delete
                                            </DropdownItem>
                                        </DropdownMenu>
                                    </Dropdown>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            )}

            <Dialog open={!!collectionForDelete} onClose={() => setCollectionForDelete(null)} size='sm'>
                <DialogTitle>Delete Collection</DialogTitle>
                <DialogDescription>
                    Are you sure you want to delete &ldquo;{collectionForDelete?.name}&rdquo;? This action cannot be
                    undone. Images will not be deleted or modified.
                </DialogDescription>
                <DialogActions>
                    <Button plain onClick={() => setCollectionForDelete(null)}>
                        Cancel
                    </Button>
                    <Button color='red' onClick={confirmDelete}>
                        Delete
                    </Button>
                </DialogActions>
            </Dialog>
        </PageContentBlock>
    );
};

export default CollectionManagementContainer;
