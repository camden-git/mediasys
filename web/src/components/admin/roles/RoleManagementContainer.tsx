import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Heading } from '../../elements/Heading';
import PageContentBlock from '../../elements/PageContentBlock.tsx';
import { Can } from '../../elements/Can';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../elements/Table';
import { AdminRoleResponse } from '../../../types';
import { Button } from '../../elements/Button';
import CreateRoleForm from './CreateRoleForm';
import EditRoleForm from './EditRoleForm';
import { useRoles } from '../../../api/swr/useRoles';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import { deleteRole } from '../../../api/admin/roles';
import { Input } from '../../elements/Input';
import { Text } from '../../elements/Text.tsx';
import { mutate } from 'swr';
import { PaginationControls } from '../../elements/PaginationControls';
import { Dropdown, DropdownButton, DropdownItem, DropdownMenu, DropdownSeparator } from '../../elements/Dropdown';
import { EllipsisHorizontalIcon } from '@heroicons/react/20/solid';
import { Dialog, DialogActions, DialogDescription, DialogTitle } from '../../elements/Dialog';

const DEFAULT_PER_PAGE = 25;

const RoleManagementContainer: React.FC = () => {
    const [page, setPage] = useState(1);
    const perPage = DEFAULT_PER_PAGE;
    const { data: rolesResult, error, isValidating } = useRoles({ page, perPage });
    const { clearFlashes, clearAndAddHttpError, addFlash } = useFlash();
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [searchQuery, setSearchQuery] = useState('');
    const [roleForEdit, setRoleForEdit] = useState<AdminRoleResponse | null>(null);
    const [roleForDelete, setRoleForDelete] = useState<AdminRoleResponse | null>(null);

    const roles = rolesResult?.items ?? [];
    const pagination = rolesResult?.pagination;
    useEffect(() => {
        if (pagination && pagination.totalPages > 0 && page > pagination.totalPages) {
            setPage(pagination.totalPages);
        }
    }, [pagination, page]);

    useEffect(() => {
        if (!error) {
            clearFlashes('roles');
            return;
        }

        clearAndAddHttpError({ error, key: 'roles' });
    }, [error, clearFlashes, clearAndAddHttpError]);

    const handleDeleteRole = useCallback((role: AdminRoleResponse) => {
        setRoleForDelete(role);
    }, []);

    const confirmDeleteRole = useCallback(async () => {
        if (!roleForDelete) return;
        try {
            await deleteRole(roleForDelete.id);
            mutate((key) => Array.isArray(key) && key[0] === 'roles');
            addFlash({
                key: 'roles',
                type: 'success',
                title: 'Role Removed',
                message: `Role "${roleForDelete.name}" has been deleted.`,
            });
        } catch (error: any) {
            clearAndAddHttpError({ error, key: 'roles' });
        } finally {
            setRoleForDelete(null);
        }
    }, [roleForDelete, addFlash, clearAndAddHttpError]);

    const filteredRoles = useMemo(() => {
        if (!roles) return [];
        const query = searchQuery.trim().toLowerCase();
        if (!query) return roles;
        return roles.filter((role) => {
            const haystack = [
                role.name,
                ...(role.global_permissions || []),
                ...(role.global_album_permissions || []),
                ...(role.album_permissions?.flatMap((ap) => ap.permissions) || []),
            ]
                .join(' ')
                .toLowerCase();
            return haystack.includes(query);
        });
    }, [roles, searchQuery]);

    if (!rolesResult || (error && isValidating)) {
        return <LoadingSpinner />;
    }

    return (
        <PageContentBlock title={'Role Management'}>
            <div className='mb-6 flex w-full flex-wrap items-end justify-between gap-4'>
                <div>
                    <Heading>Roles</Heading>
                    <Text className='mt-1 text-sm text-gray-600 dark:text-gray-300'>
                        Roles bundle permissions that can be assigned to users. Use them to grant consistent access
                        across your team.
                    </Text>
                </div>
                <div className='flex gap-4'>
                    <Can permission={'role.create'}>
                        <Button onClick={() => setShowCreateModal(true)}>Create Role</Button>
                    </Can>
                </div>
            </div>

            <FlashMessageRender byKey={'roles'} className={'mb-4'} />

            <CreateRoleForm isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} />
            <EditRoleForm
                key={roleForEdit?.id ?? 'edit-role'}
                isOpen={!!roleForEdit}
                onClose={() => setRoleForEdit(null)}
                role={roleForEdit ?? undefined}
            />

            <div className='mb-6 flex flex-wrap items-center justify-between gap-4'>
                <Input
                    type='search'
                    placeholder='Search roles by name or permission…'
                    value={searchQuery}
                    onChange={(event) => setSearchQuery(event.target.value)}
                    className='w-full max-w-md'
                />
                <Text className='text-sm text-gray-500'>
                    Showing {filteredRoles.length} of {pagination?.count ?? roles.length} role
                    {(pagination?.count ?? roles.length) === 1 ? '' : 's'} on this page
                    {pagination?.total ? ` (Total ${pagination.total})` : ''}
                </Text>
            </div>

            <Can permission={'role.list'}>
                {filteredRoles.length === 0 ? (
                    <div className='flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 p-10 text-center'>
                        <Heading level={4} className='text-lg font-semibold'>
                            No roles match your filters
                        </Heading>
                        <Text className='mt-2 text-sm text-gray-600'>
                            Try adjusting your search, or create a new role to get started.
                        </Text>
                    </div>
                ) : (
                    <Table>
                        <TableHead>
                            <TableRow>
                                <TableHeader>Name</TableHeader>
                                <TableHeader>Global</TableHeader>
                                <TableHeader>Album</TableHeader>
                                <TableHeader>Actions</TableHeader>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {filteredRoles.map((role: AdminRoleResponse) => (
                                <TableRow key={role.id} href={`/admin/roles/${role.id}`}>
                                    <TableCell className='font-medium'>{role.name}</TableCell>
                                    <TableCell>
                                        {(role.global_permissions?.length ?? 0) === 0 ? (
                                            <span className='text-sm text-gray-400'>None</span>
                                        ) : (
                                            <span className='text-sm'>{role.global_permissions!.length}</span>
                                        )}
                                    </TableCell>
                                    <TableCell>
                                        {(() => {
                                            const g = role.global_album_permissions?.length ?? 0;
                                            const s = role.album_permissions?.length ?? 0;
                                            if (g === 0 && s === 0) {
                                                return <span className='text-sm text-gray-400'>None</span>;
                                            }
                                            const parts: string[] = [];
                                            if (g > 0) parts.push(`${g} global`);
                                            if (s > 0) parts.push(`${s} scoped`);
                                            return <span className='text-sm'>{parts.join(' · ')}</span>;
                                        })()}
                                    </TableCell>
                                    <TableCell>
                                        <Dropdown>
                                            <DropdownButton plain aria-label='Actions'>
                                                <EllipsisHorizontalIcon className='size-5' />
                                            </DropdownButton>
                                            <DropdownMenu>
                                                <Can permission='role.view'>
                                                    <DropdownItem to={`/admin/roles/${role.id}`}>View</DropdownItem>
                                                </Can>
                                                <Can permission='role.edit'>
                                                    <DropdownItem onClick={() => setRoleForEdit(role)}>Edit</DropdownItem>
                                                </Can>
                                                <DropdownSeparator />
                                                <Can permission='role.delete'>
                                                    <DropdownItem
                                                        onClick={() => handleDeleteRole(role)}
                                                        className='text-red-600 data-[focus]:bg-red-500'
                                                    >
                                                        Delete
                                                    </DropdownItem>
                                                </Can>
                                            </DropdownMenu>
                                        </Dropdown>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                )}
            </Can>

            {pagination && pagination.totalPages > 1 && (
                <PaginationControls
                    className='mt-6'
                    pagination={pagination}
                    currentPage={page}
                    onPageChange={setPage}
                />
            )}

            <Dialog open={!!roleForDelete} onClose={() => setRoleForDelete(null)} size='sm'>
                <DialogTitle>Delete Role</DialogTitle>
                <DialogDescription>
                    Are you sure you want to delete role &ldquo;{roleForDelete?.name}&rdquo;? This action cannot be
                    undone.
                </DialogDescription>
                <DialogActions>
                    <Button plain onClick={() => setRoleForDelete(null)}>
                        Cancel
                    </Button>
                    <Button color='red' onClick={confirmDeleteRole}>
                        Delete
                    </Button>
                </DialogActions>
            </Dialog>
        </PageContentBlock>
    );
};

export default RoleManagementContainer;
