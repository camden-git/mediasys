import React, { useCallback, useEffect, useState } from 'react';
import { Heading } from '../../elements/Heading';
import PageContentBlock from '../../elements/PageContentBlock.tsx';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../elements/Table';
import { AlbumGroup } from '../../../types';
import { Button } from '../../elements/Button';
import CreateGroupForm from './CreateGroupForm';
import EditGroupForm from './EditGroupForm';
import { deleteGroup, listGroups } from '../../../api/admin/groups';
import { queryClient } from '../../../lib/queryClient';
import { queryKeys } from '../../../lib/queryKeys';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import { Text } from '../../elements/Text.tsx';
import { Dropdown, DropdownButton, DropdownItem, DropdownMenu, DropdownSeparator } from '../../elements/Dropdown';
import { EllipsisHorizontalIcon } from '@heroicons/react/20/solid';
import { Dialog, DialogActions, DialogDescription, DialogTitle } from '../../elements/Dialog';

const GroupManagementContainer: React.FC = () => {
    const [groups, setGroups] = useState<AlbumGroup[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const { clearAndAddHttpError, addFlash } = useFlash();
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [groupForEdit, setGroupForEdit] = useState<AlbumGroup | null>(null);
    const [groupForDelete, setGroupForDelete] = useState<AlbumGroup | null>(null);

    const fetchGroups = useCallback(async () => {
        try {
            const data = await listGroups();
            setGroups(data);
        } catch (error: any) {
            clearAndAddHttpError({ error, key: 'groups' });
        } finally {
            setIsLoading(false);
        }
    }, [clearAndAddHttpError]);

    useEffect(() => {
        fetchGroups();
    }, [fetchGroups]);

    const handleChanged = useCallback(() => {
        void queryClient.invalidateQueries({ queryKey: queryKeys.groups.all() });
        void fetchGroups();
    }, [fetchGroups]);

    const confirmDeleteGroup = useCallback(async () => {
        if (!groupForDelete) return;
        try {
            await deleteGroup(groupForDelete.id);
            setGroups((prev) => prev.filter((g) => g.id !== groupForDelete.id));
            void queryClient.invalidateQueries({ queryKey: queryKeys.groups.all() });
            addFlash({
                key: 'groups',
                type: 'success',
                title: 'Group Removed',
                message: `Group "${groupForDelete.name}" has been deleted.`,
            });
        } catch (error: any) {
            clearAndAddHttpError({ error, key: 'groups' });
        } finally {
            setGroupForDelete(null);
        }
    }, [groupForDelete, addFlash, clearAndAddHttpError]);

    if (isLoading) {
        return <LoadingSpinner />;
    }

    return (
        <PageContentBlock title={'Group Management'}>
            <div className='mb-6 flex w-full flex-wrap items-end justify-between gap-4'>
                <div>
                    <Heading>Groups</Heading>
                    <Text className='mt-1 text-sm text-gray-600 dark:text-gray-300'>
                        Groups organise albums into collections visible on the home page.
                    </Text>
                </div>
                <Button onClick={() => setShowCreateModal(true)}>Create Group</Button>
            </div>

            <FlashMessageRender byKey={'groups'} className={'mb-4'} />

            <CreateGroupForm
                isOpen={showCreateModal}
                onClose={() => setShowCreateModal(false)}
                onCreated={handleChanged}
            />
            <EditGroupForm
                key={groupForEdit?.id ?? 'edit-group'}
                isOpen={!!groupForEdit}
                onClose={() => setGroupForEdit(null)}
                onUpdated={handleChanged}
                group={groupForEdit ?? undefined}
            />

            {groups.length === 0 ? (
                <div className='flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 p-10 text-center'>
                    <Heading level={4} className='text-lg font-semibold'>
                        No groups yet
                    </Heading>
                    <Text className='mt-2 text-sm text-gray-600'>
                        Create a group to start organising albums into collections.
                    </Text>
                </div>
            ) : (
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableHeader>Name</TableHeader>
                            <TableHeader>Slug</TableHeader>
                            <TableHeader>Hidden</TableHeader>
                            <TableHeader>Albums</TableHeader>
                            <TableHeader>Actions</TableHeader>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {groups.map((group) => (
                            <TableRow key={group.id}>
                                <TableCell className='font-medium'>{group.name}</TableCell>
                                <TableCell>
                                    <span className='font-mono text-sm text-gray-500'>{group.slug}</span>
                                </TableCell>
                                <TableCell>
                                    {group.is_hidden ? (
                                        <span className='text-sm text-gray-400'>Yes</span>
                                    ) : (
                                        <span className='text-sm'>No</span>
                                    )}
                                </TableCell>
                                <TableCell>
                                    <span className='text-sm'>{group.albums?.length ?? 0}</span>
                                </TableCell>
                                <TableCell>
                                    <Dropdown>
                                        <DropdownButton plain aria-label='Actions'>
                                            <EllipsisHorizontalIcon className='size-5' />
                                        </DropdownButton>
                                        <DropdownMenu>
                                            <DropdownItem onClick={() => setGroupForEdit(group)}>Edit</DropdownItem>
                                            <DropdownSeparator />
                                            <DropdownItem
                                                onClick={() => setGroupForDelete(group)}
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

            <Dialog open={!!groupForDelete} onClose={() => setGroupForDelete(null)} size='sm'>
                <DialogTitle>Delete Group</DialogTitle>
                <DialogDescription>
                    Are you sure you want to delete group &ldquo;{groupForDelete?.name}&rdquo;? This action cannot be
                    undone. Albums in this group will not be deleted, but will be unassigned from it.
                </DialogDescription>
                <DialogActions>
                    <Button plain onClick={() => setGroupForDelete(null)}>
                        Cancel
                    </Button>
                    <Button color='red' onClick={confirmDeleteGroup}>
                        Delete
                    </Button>
                </DialogActions>
            </Dialog>
        </PageContentBlock>
    );
};

export default GroupManagementContainer;
