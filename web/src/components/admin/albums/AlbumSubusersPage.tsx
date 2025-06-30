import React, { useState } from 'react';
import { Can } from '../../elements/Can';
import { useAlbumUsers, useAvailableUsers } from '../../../api/swr/useAlbums';
import {
    addUserToAlbum,
    updateUserAlbumPermissions,
    removeUserFromAlbum,
    AddUserToAlbumPayload,
    UpdateUserAlbumPermissionsPayload,
} from '../../../api/admin/albums';
import { useFlash } from '../../../hooks/useFlash';
import { Button } from '../../elements/Button';
import { Dialog, DialogActions, DialogBody, DialogTitle } from '../../elements/Dialog';
import { Field, FieldGroup, Label, Description } from '../../elements/Fieldset';
import { Select } from '../../elements/Select';
import { Checkbox } from '../../elements/Checkbox';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../elements/Table';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { usePermissionDefinitions } from '../../../api/swr/useRoles';
import { useStoreState } from '../../../store/hooks';

const AlbumSubusersPage: React.FC = () => {
    const albumId = useStoreState((state) => state.albumContext.data!.id);
    const { addFlash } = useFlash();

    const {
        users: albumUsers,
        isLoading: isLoadingUsers,
        error: usersError,
        mutate: mutateUsers,
    } = useAlbumUsers(albumId);
    const { users: availableUsers } = useAvailableUsers(albumId);
    const { data: permissionDefinitions } = usePermissionDefinitions();

    const [showAddModal, setShowAddModal] = useState(false);
    const [showEditModal, setShowEditModal] = useState(false);
    const [selectedUser, setSelectedUser] = useState<any>(null);
    const [selectedPermissions, setSelectedPermissions] = useState<string[]>([]);
    const [selectedUserId, setSelectedUserId] = useState<number | null>(null);
    const [isSubmitting, setIsSubmitting] = useState(false);

    const albumPermissions = permissionDefinitions?.find((group) => group.key === 'album')?.permissions || [];
    const albumScopedPermissions = albumPermissions.filter((p) => p.scope === 'album');

    const handleAddUser = async () => {
        if (!selectedUserId || selectedPermissions.length === 0) return;

        setIsSubmitting(true);
        try {
            const payload: AddUserToAlbumPayload = {
                user_id: selectedUserId,
                permissions: selectedPermissions,
            };

            await addUserToAlbum(albumId!, payload);
            addFlash({
                key: 'album-user-added',
                type: 'success',
                message: 'User added to album successfully',
            });
            setShowAddModal(false);
            setSelectedUserId(null);
            setSelectedPermissions([]);
            mutateUsers();
        } catch (error: any) {
            addFlash({
                key: 'album-user-added-error',
                type: 'error',
                message: error.response?.data?.error || 'Failed to add user to album',
            });
        } finally {
            setIsSubmitting(false);
        }
    };

    const handleUpdatePermissions = async () => {
        if (!selectedUser || selectedPermissions.length === 0) return;

        setIsSubmitting(true);
        try {
            const payload: UpdateUserAlbumPermissionsPayload = {
                permissions: selectedPermissions,
            };

            await updateUserAlbumPermissions(albumId!, selectedUser.user.id, payload);
            addFlash({
                key: 'album-permissions-updated',
                type: 'success',
                message: 'User permissions updated successfully',
            });
            setShowEditModal(false);
            setSelectedUser(null);
            setSelectedPermissions([]);
            mutateUsers();
        } catch (error: any) {
            addFlash({
                key: 'album-permissions-updated-error',
                type: 'error',
                message: error.response?.data?.error || 'Failed to update user permissions',
            });
        } finally {
            setIsSubmitting(false);
        }
    };

    const handleRemoveUser = async (userId: number, username: string) => {
        if (!confirm(`Are you sure you want to remove ${username} from this album?`)) return;

        try {
            await removeUserFromAlbum(albumId!, userId);
            addFlash({
                key: 'album-user-removed',
                type: 'success',
                message: 'User removed from album successfully',
            });
            mutateUsers();
        } catch (error: any) {
            addFlash({
                key: 'album-user-removed-error',
                type: 'error',
                message: error.response?.data?.error || 'Failed to remove user from album',
            });
        }
    };

    const openEditModal = (user: any) => {
        setSelectedUser(user);
        setSelectedPermissions(user.permissions || []);
        setShowEditModal(true);
    };

    if (isLoadingUsers) {
        return (
            <div className='flex h-64 items-center justify-center'>
                <LoadingSpinner />
            </div>
        );
    }

    if (usersError) {
        return <div className='text-center text-red-600'>Error loading album users: {usersError.message}</div>;
    }

    return (
        <div className='space-y-6'>
            <div className='flex items-center justify-between'>
                <h1 className='text-2xl font-bold text-gray-900'>Album Subusers</h1>
                <Can permission='album.manage.members.global'>
                    <Button onClick={() => setShowAddModal(true)}>Add User</Button>
                </Can>
            </div>

            <div className='rounded-lg bg-white p-6 shadow'>
                {albumUsers.length === 0 ? (
                    <p className='text-gray-600'>No users have been added to this album yet.</p>
                ) : (
                    <Table>
                        <TableHead>
                            <TableRow>
                                <TableHeader>User</TableHeader>
                                <TableHeader>Permissions</TableHeader>
                                <TableHeader>Actions</TableHeader>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {albumUsers.map((userData) => (
                                <TableRow key={userData.user.id}>
                                    <TableCell>
                                        <div>
                                            <div className='font-medium'>{userData.user.username}</div>
                                            <div className='text-sm text-gray-500'>ID: {userData.user.id}</div>
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        <div className='flex flex-wrap gap-1'>
                                            {userData.permissions.map((perm) => {
                                                const permDef = albumScopedPermissions.find((p) => p.key === perm);
                                                return (
                                                    <span
                                                        key={perm}
                                                        className='inline-flex items-center rounded-full bg-blue-100 px-2 py-1 text-xs font-medium text-blue-800'
                                                    >
                                                        {permDef?.name || perm}
                                                    </span>
                                                );
                                            })}
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        <div className='flex space-x-2'>
                                            <Can permission='album.manage.members.global'>
                                                <Button
                                                    plain
                                                    onClick={() => openEditModal(userData)}
                                                    className='px-2 py-1 text-xs'
                                                >
                                                    Edit
                                                </Button>
                                                <Button
                                                    color='red'
                                                    onClick={() =>
                                                        handleRemoveUser(userData.user.id, userData.user.username)
                                                    }
                                                    className='px-2 py-1 text-xs'
                                                >
                                                    Remove
                                                </Button>
                                            </Can>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                )}
            </div>

            <Dialog open={showAddModal} onClose={() => setShowAddModal(false)}>
                <DialogTitle>Add User to Album</DialogTitle>
                <DialogBody>
                    <FieldGroup>
                        <Field>
                            <Label>Select User</Label>
                            <Select
                                value={selectedUserId || ''}
                                onChange={(e) => setSelectedUserId(parseInt(e.target.value) || null)}
                                disabled={isSubmitting}
                            >
                                <option value=''>Choose a user...</option>
                                {availableUsers.map((user) => (
                                    <option key={user.id} value={user.id}>
                                        {user.username} (ID: {user.id})
                                    </option>
                                ))}
                            </Select>
                            <Description>Select a user to add to this album</Description>
                        </Field>

                        <Field>
                            <Label>Permissions</Label>
                            <Description>Select the permissions to grant to this user for this album</Description>
                            <div className='mt-2 space-y-2'>
                                {albumScopedPermissions.map((perm) => (
                                    <label key={perm.key} className='flex items-center space-x-2'>
                                        <Checkbox
                                            checked={selectedPermissions.includes(perm.key)}
                                            onChange={(checked) => {
                                                if (checked) {
                                                    setSelectedPermissions([...selectedPermissions, perm.key]);
                                                } else {
                                                    setSelectedPermissions(
                                                        selectedPermissions.filter((p) => p !== perm.key),
                                                    );
                                                }
                                            }}
                                            disabled={isSubmitting}
                                        />
                                        <span className='text-sm'>
                                            {perm.name}
                                            <span className='ml-1 text-gray-500'>({perm.key})</span>
                                        </span>
                                    </label>
                                ))}
                            </div>
                        </Field>
                    </FieldGroup>
                </DialogBody>
                <DialogActions>
                    <Button outline onClick={() => setShowAddModal(false)} disabled={isSubmitting}>
                        Cancel
                    </Button>
                    <Button
                        onClick={handleAddUser}
                        disabled={isSubmitting || !selectedUserId || selectedPermissions.length === 0}
                    >
                        {isSubmitting ? 'Adding...' : 'Add User'}
                    </Button>
                </DialogActions>
            </Dialog>

            <Dialog open={showEditModal} onClose={() => setShowEditModal(false)}>
                <DialogTitle>Edit User Permissions</DialogTitle>
                <DialogBody>
                    {selectedUser && (
                        <div className='mb-4'>
                            <p className='text-sm text-gray-600'>
                                Editing permissions for <strong>{selectedUser.user.username}</strong>
                            </p>
                        </div>
                    )}
                    <FieldGroup>
                        <Field>
                            <Label>Permissions</Label>
                            <Description>Select the permissions to grant to this user for this album</Description>
                            <div className='mt-2 space-y-2'>
                                {albumScopedPermissions.map((perm) => (
                                    <label key={perm.key} className='flex items-center space-x-2'>
                                        <Checkbox
                                            checked={selectedPermissions.includes(perm.key)}
                                            onChange={(checked) => {
                                                if (checked) {
                                                    setSelectedPermissions([...selectedPermissions, perm.key]);
                                                } else {
                                                    setSelectedPermissions(
                                                        selectedPermissions.filter((p) => p !== perm.key),
                                                    );
                                                }
                                            }}
                                            disabled={isSubmitting}
                                        />
                                        <span className='text-sm'>
                                            {perm.name}
                                            <span className='ml-1 text-gray-500'>({perm.key})</span>
                                        </span>
                                    </label>
                                ))}
                            </div>
                        </Field>
                    </FieldGroup>
                </DialogBody>
                <DialogActions>
                    <Button outline onClick={() => setShowEditModal(false)} disabled={isSubmitting}>
                        Cancel
                    </Button>
                    <Button
                        onClick={handleUpdatePermissions}
                        disabled={isSubmitting || selectedPermissions.length === 0}
                    >
                        {isSubmitting ? 'Updating...' : 'Update Permissions'}
                    </Button>
                </DialogActions>
            </Dialog>
        </div>
    );
};

export default AlbumSubusersPage;
