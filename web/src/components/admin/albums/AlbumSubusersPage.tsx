import React, { useState } from 'react';
import { Can } from '../../elements/Can';
import { useAlbumUsers, useAvailableUsers } from '../../../api/query/useAlbums';
import {
    addUserToAlbum,
    updateUserAlbumPermissions,
    removeUserFromAlbum,
    AddUserToAlbumPayload,
    UpdateUserAlbumPermissionsPayload,
} from '../../../api/admin/albums';
import { useFlash } from '../../../hooks/useFlash';
import { Button } from '../../elements/Button';
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../../elements/Dialog';
import { Heading } from '../../elements/Heading';
import { Field, FieldGroup, Label, Description } from '../../elements/Fieldset';
import { Select } from '../../elements/Select';
import { Checkbox } from '../../elements/Checkbox';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../elements/Table';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { usePermissionDefinitions } from '../../../api/query/useRoles';
import { useAlbumContextStore } from '../../../store/useAlbumContextStore';

const AlbumSubusersPage: React.FC = () => {
    const albumId = useAlbumContextStore((s) => s.data!.id);
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
    const [showRemoveModal, setShowRemoveModal] = useState(false);
    const [userToRemove, setUserToRemove] = useState<{ id: number; username: string } | null>(null);
    const [selectedUser, setSelectedUser] = useState<any>(null);
    const [selectedPermissions, setSelectedPermissions] = useState<string[]>([]);
    const [selectedUserId, setSelectedUserId] = useState<number | null>(null);
    const [isSubmitting, setIsSubmitting] = useState(false);

    const albumPermissions = permissionDefinitions?.find((group) => group.key === 'album')?.permissions || [];
    const albumScopedPermissions = albumPermissions.filter((p) => p.scope === 'album');

    const renderPermissionTags = (perms: string[], styleClass: string) =>
        perms.map((perm) => {
            const permDef = albumScopedPermissions.find((p) => p.key === perm);
            return (
                <span
                    key={`${perm}-${styleClass}`}
                    className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${styleClass}`}
                >
                    {permDef?.name || perm}
                </span>
            );
        });

    const handleAddUser = async () => {
        if (!selectedUserId) return;

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
                title: 'Success',
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
                title: 'Error',
                message: error.message || 'Failed to add user to album',
            });
        } finally {
            setIsSubmitting(false);
        }
    };

    const handleUpdatePermissions = async () => {
        if (!selectedUser) return;

        setIsSubmitting(true);
        try {
            const payload: UpdateUserAlbumPermissionsPayload = {
                permissions: selectedPermissions,
            };

            if (selectedUser.user_album_permission) {
                await updateUserAlbumPermissions(albumId!, selectedUser.user.id, payload);
            } else {
                await addUserToAlbum(albumId!, {
                    user_id: selectedUser.user.id,
                    permissions: selectedPermissions,
                });
            }
            addFlash({
                key: 'album-permissions-updated',
                type: 'success',
                title: 'Success',
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
                title: 'Error',
                message: error.message || 'Failed to update user permissions',
            });
        } finally {
            setIsSubmitting(false);
        }
    };

    const handleRemoveClick = (userId: number, username: string) => {
        setUserToRemove({ id: userId, username });
        setShowRemoveModal(true);
    };

    const handleRemoveConfirm = async () => {
        if (!userToRemove) return;

        try {
            await removeUserFromAlbum(albumId!, userToRemove.id);
            addFlash({
                key: 'album-user-removed',
                type: 'success',
                title: 'Success',
                message: 'User removed from album successfully',
            });
            setShowRemoveModal(false);
            setUserToRemove(null);
            mutateUsers();
        } catch (error: any) {
            addFlash({
                key: 'album-user-removed-error',
                type: 'error',
                title: 'Error',
                message: error.message || 'Failed to remove user from album',
            });
        }
    };

    const openEditModal = (user: any) => {
        setSelectedUser(user);
        setSelectedPermissions(user.direct_permissions || []);
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
                <Heading>Album Subusers</Heading>
                <Can permission={['album.manage.members.global', 'album.manage.members']} albumId={albumId}>
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
                                        <div className='space-y-3'>
                                            <div>
                                                <div className='text-xs font-semibold tracking-wide text-gray-500 uppercase'>
                                                    Direct
                                                </div>
                                                <div className='mt-1 flex flex-wrap gap-1'>
                                                    {userData.direct_permissions &&
                                                    userData.direct_permissions.length > 0 ? (
                                                        renderPermissionTags(
                                                            userData.direct_permissions,
                                                            'bg-blue-100 text-blue-800',
                                                        )
                                                    ) : (
                                                        <span className='text-xs text-gray-400'>None</span>
                                                    )}
                                                </div>
                                            </div>
                                            {userData.inherited_permissions &&
                                                userData.inherited_permissions.length > 0 && (
                                                    <div>
                                                        <div className='text-xs font-semibold tracking-wide text-gray-500 uppercase'>
                                                            Inherited
                                                        </div>
                                                        <div className='mt-1 flex flex-wrap gap-1'>
                                                            {renderPermissionTags(
                                                                userData.inherited_permissions,
                                                                'bg-purple-100 text-purple-800',
                                                            )}
                                                        </div>
                                                    </div>
                                                )}
                                            {userData.role_contributions && userData.role_contributions.length > 0 && (
                                                <div className='rounded-md border border-gray-100 p-3 text-xs text-gray-600'>
                                                    <div className='font-semibold tracking-wide text-gray-500 uppercase'>
                                                        Role-derived
                                                    </div>
                                                    <ul className='mt-2 space-y-2'>
                                                        {userData.role_contributions.map((contrib) => (
                                                            <li key={contrib.role_id}>
                                                                <div className='font-medium text-gray-700'>
                                                                    {contrib.role_name}
                                                                </div>
                                                                {contrib.for_all_albums &&
                                                                    contrib.for_all_albums.length > 0 && (
                                                                        <div className='mt-1 flex flex-wrap gap-1'>
                                                                            {renderPermissionTags(
                                                                                contrib.for_all_albums,
                                                                                'bg-amber-100 text-amber-800',
                                                                            )}
                                                                        </div>
                                                                    )}
                                                                {contrib.album_specific &&
                                                                    contrib.album_specific.length > 0 && (
                                                                        <div className='mt-1 flex flex-wrap gap-1'>
                                                                            {renderPermissionTags(
                                                                                contrib.album_specific,
                                                                                'bg-gray-200 text-gray-800',
                                                                            )}
                                                                        </div>
                                                                    )}
                                                            </li>
                                                        ))}
                                                    </ul>
                                                </div>
                                            )}
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        <div className='flex space-x-2'>
                                            <Can
                                                permission={['album.manage.members.global', 'album.manage.members']}
                                                albumId={albumId}
                                            >
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
                                                        handleRemoveClick(userData.user.id, userData.user.username)
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
                    <Button onClick={handleAddUser} disabled={isSubmitting || !selectedUserId}>
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
                    <Button onClick={handleUpdatePermissions} disabled={isSubmitting}>
                        {isSubmitting ? 'Updating...' : 'Update Permissions'}
                    </Button>
                </DialogActions>
            </Dialog>

            <Dialog open={showRemoveModal} onClose={() => setShowRemoveModal(false)}>
                <DialogTitle>Remove User</DialogTitle>
                <DialogDescription>
                    Are you sure you want to remove {userToRemove?.username} from this album? This action cannot be
                    undone.
                </DialogDescription>
                <DialogActions>
                    <Button plain onClick={() => setShowRemoveModal(false)}>
                        Cancel
                    </Button>
                    <Button color='red' onClick={handleRemoveConfirm}>
                        Remove
                    </Button>
                </DialogActions>
            </Dialog>
        </div>
    );
};

export default AlbumSubusersPage;
