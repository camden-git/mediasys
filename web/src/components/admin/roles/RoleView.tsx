import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { DescriptionList, DescriptionTerm, DescriptionDetails } from '../../elements/DescriptionList';
import { ErrorMessage } from '../../elements/Fieldset';
import { Heading } from '../../elements/Heading';
import ContentBlock from '../../elements/PageContentBlock.tsx';
import { Button } from '../../elements/Button';
import { Can } from '../../elements/Can';
import EditRoleForm from './EditRoleForm';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../lib/queryKeys';
import { refreshAuthUser } from '../../../store/useAuthStore';
import { useRole, useRoleUsers } from '../../../api/query/useRoles';
import AddUserToRoleDialog from './AddUserToRoleDialog';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import { addUserToRole, removeUserFromRole } from '../../../api/admin/roles';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { Text } from '../../elements/Text.tsx';
import { PaginationControls } from '../../elements/PaginationControls';
import { Dialog, DialogActions, DialogDescription, DialogTitle } from '../../elements/Dialog';

const RoleView: React.FC = () => {
    const { id } = useParams<{ id: string }>();
    const roleId = id ? parseInt(id, 10) : 0;

    const { data: role, error } = useRole(roleId);
    const { addFlash, clearFlashes, clearAndAddHttpError } = useFlash();
    const queryClient = useQueryClient();

    const [userPage, setUserPage] = useState(1);
    const userPerPage = 25;

    // single fetch path for the members list; membership changes invalidate it via queryKeys.roles.all()
    const {
        data: userResult,
        isLoading: isLoadingUsers,
        error: userQueryError,
    } = useRoleUsers(roleId, { page: userPage, perPage: userPerPage });
    const userError = userQueryError ? userQueryError.message || 'Failed to load users' : null;

    const [isEditModalOpen, setEditModalOpen] = useState(false);
    const [isAddUserModalOpen, setAddUserModalOpen] = useState(false);
    const [userForRemove, setUserForRemove] = useState<{ id: number; username: string } | null>(null);
    const [isRemoving, setIsRemoving] = useState(false);

    const refreshRoleMembership = (userId: number) => {
        void queryClient.invalidateQueries({ queryKey: queryKeys.roles.all() });
        void queryClient.invalidateQueries({ queryKey: queryKeys.users.all() });
        refreshAuthUser(userId);
    };

    useEffect(() => {
        if (!error) {
            clearFlashes('role-view');
            return;
        }

        clearAndAddHttpError({ error, key: 'role-view' });
    }, [error, clearFlashes, clearAndAddHttpError]);

    if (error && !role) {
        return <ErrorMessage>Error: {error.message}</ErrorMessage>;
    }

    if (!role) {
        return <LoadingSpinner />;
    }

    const handleAddUserToRole = async (userId: number): Promise<void> => {
        try {
            await addUserToRole(roleId, userId);
            refreshRoleMembership(userId);
            addFlash({
                key: 'role-view',
                type: 'success',
                message: 'User added to role successfully!',
            });
            setAddUserModalOpen(false);
        } catch (error: any) {
            addFlash({
                key: 'role-view',
                type: 'error',
                message: error.message || 'Failed to add user to role.',
            });
        }
    };

    const handleRemoveUserFromRole = (userId: number, username: string) => {
        setUserForRemove({ id: userId, username });
    };

    const confirmRemoveUser = async () => {
        if (!userForRemove || isRemoving) return;
        setIsRemoving(true);
        try {
            await removeUserFromRole(roleId, userForRemove.id);
            refreshRoleMembership(userForRemove.id);
            addFlash({
                key: 'role-view',
                type: 'success',
                message: 'User removed from role successfully!',
            });
        } catch (error: any) {
            addFlash({
                key: 'role-view',
                type: 'error',
                message: error.message || 'Failed to remove user from role.',
            });
        } finally {
            setIsRemoving(false);
            setUserForRemove(null);
        }
    };

    const renderPermissionChips = (permissions: string[] | undefined) => {
        if (!permissions || permissions.length === 0) {
            return <span className='text-sm text-gray-500'>None</span>;
        }
        return (
            <div className='flex flex-wrap gap-1'>
                {permissions.map((perm) => (
                    <span
                        key={perm}
                        className='inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-800'
                    >
                        {perm}
                    </span>
                ))}
            </div>
        );
    };

    const renderAlbumRules = (currentRole: typeof role) => {
        if (!currentRole) return null;
        const globalAlbum = currentRole.global_album_permissions || [];
        const scopedAlbums = currentRole.album_permissions || [];
        if (globalAlbum.length === 0 && scopedAlbums.length === 0) {
            return <span className='text-sm text-gray-500'>None</span>;
        }
        return (
            <div className='space-y-3 text-sm'>
                {globalAlbum.length > 0 && (
                    <div>
                        <div className='text-xs font-semibold text-gray-500 uppercase'>All Albums</div>
                        <div className='mt-1 flex flex-wrap gap-1'>
                            {globalAlbum.map((perm) => (
                                <span
                                    key={`role-${currentRole.id}-global-${perm}`}
                                    className='inline-flex items-center rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-800'
                                >
                                    {perm}
                                </span>
                            ))}
                        </div>
                    </div>
                )}
                {scopedAlbums.length > 0 && (
                    <div className='space-y-2'>
                        {scopedAlbums.map((ap) => (
                            <div
                                key={`${currentRole.id}-${ap.album_id}`}
                                className='rounded border border-gray-200 p-3'
                            >
                                <div className='text-xs font-semibold text-gray-500 uppercase'>
                                    Album #{ap.album_id}
                                </div>
                                <div className='mt-1 flex flex-wrap gap-1'>
                                    {ap.permissions.map((perm) => (
                                        <span
                                            key={`${currentRole.id}-${ap.album_id}-${perm}`}
                                            className='inline-flex items-center rounded-full bg-gray-200 px-2 py-0.5 text-xs font-medium text-gray-800'
                                        >
                                            {perm}
                                        </span>
                                    ))}
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        );
    };

    const users = userResult?.items ?? [];
    const userPagination = userResult?.pagination;

    return (
        <>
            <FlashMessageRender byKey={'role-view'} className={'mb-4'} />
            <FlashMessageRender byKey={'role-edit'} className={'mb-4'} />

            <ContentBlock>
                <div className='flex items-center justify-between'>
                    <Heading level={2}>Role Details</Heading>
                    <Can permission='role.edit'>
                        <Button onClick={() => setEditModalOpen(true)}>Edit Role</Button>
                    </Can>
                </div>
                <Text className='mt-2 text-sm text-gray-600 dark:text-gray-300'>
                    Review the permissions bundled in this role and manage which users inherit its access.
                </Text>
                <DescriptionList className='mt-4'>
                    <DescriptionTerm>ID</DescriptionTerm>
                    <DescriptionDetails>{role.id}</DescriptionDetails>

                    <DescriptionTerm>Name</DescriptionTerm>
                    <DescriptionDetails>{role.name}</DescriptionDetails>

                    <DescriptionTerm>Global Permissions</DescriptionTerm>
                    <DescriptionDetails>{renderPermissionChips(role.global_permissions)}</DescriptionDetails>

                    <DescriptionTerm>Global Album Permissions</DescriptionTerm>
                    <DescriptionDetails>{renderPermissionChips(role.global_album_permissions)}</DescriptionDetails>

                    <DescriptionTerm>Album-Specific Permissions</DescriptionTerm>
                    <DescriptionDetails>{renderAlbumRules(role)}</DescriptionDetails>
                </DescriptionList>
            </ContentBlock>

            <EditRoleForm
                isOpen={isEditModalOpen}
                onClose={() => {
                    setEditModalOpen(false);
                }}
                role={role}
            />

            <ContentBlock className='mt-6'>
                <div className='flex items-center justify-between'>
                    <Heading level={3}>Users in this Role</Heading>
                    {/* the picker lists every user, which needs user.list on top of role.edit.users */}
                    <Can permission={['role.edit.users', 'user.list']} requireAll>
                        <Button onClick={() => setAddUserModalOpen(true)}>Add User</Button>
                    </Can>
                </div>
                <Can permission='role.view.users'>
                    <>
                        {isLoadingUsers && <p>Loading users...</p>}
                        {userError && <ErrorMessage>Error: {userError}</ErrorMessage>}
                        {!isLoadingUsers && !userError && (
                            <div className='mt-4 flow-root'>
                                <div className='-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8'>
                                    <div className='inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8'>
                                        <table className='min-w-full divide-y divide-gray-300 dark:divide-zinc-600'>
                                            <thead>
                                                <tr>
                                                    <th
                                                        scope='col'
                                                        className='py-3.5 pr-3 pl-4 text-left text-sm font-semibold sm:pl-0'
                                                    >
                                                        Username
                                                    </th>
                                                    <th
                                                        scope='col'
                                                        className='px-3 py-3.5 text-left text-sm font-semibold'
                                                    >
                                                        User ID
                                                    </th>
                                                    <th scope='col' className='relative py-3.5 pr-4 pl-3 sm:pr-0'>
                                                        <span className='sr-only'>Remove</span>
                                                    </th>
                                                </tr>
                                            </thead>
                                            <tbody className='divide-y divide-gray-200 dark:divide-zinc-700'>
                                                {users.length > 0 ? (
                                                    users.map((user) => (
                                                        <tr key={user.id}>
                                                            <td className='py-4 pr-3 pl-4 text-sm font-medium whitespace-nowrap sm:pl-0'>
                                                                {user.username}
                                                            </td>
                                                            <td className='px-3 py-4 text-sm whitespace-nowrap text-gray-500'>
                                                                {user.id}
                                                            </td>
                                                            <td className='relative py-4 pr-4 pl-3 text-right text-sm font-medium whitespace-nowrap sm:pr-0'>
                                                                <Can permission='role.edit.users'>
                                                                    <Button
                                                                        plain
                                                                        onClick={() =>
                                                                            handleRemoveUserFromRole(
                                                                                user.id,
                                                                                user.username,
                                                                            )
                                                                        }
                                                                    >
                                                                        Remove
                                                                    </Button>
                                                                </Can>
                                                            </td>
                                                        </tr>
                                                    ))
                                                ) : (
                                                    <tr>
                                                        <td
                                                            colSpan={3}
                                                            className='py-4 pr-3 pl-4 text-center text-sm text-gray-500 sm:pl-0'
                                                        >
                                                            No users are assigned to this role.
                                                        </td>
                                                    </tr>
                                                )}
                                            </tbody>
                                        </table>
                                    </div>
                                </div>
                            </div>
                        )}
                    </>
                </Can>
                {userPagination && userPagination.totalPages > 1 && (
                    <PaginationControls
                        className='mt-4'
                        pagination={userPagination}
                        currentPage={userPage}
                        onPageChange={setUserPage}
                    />
                )}
            </ContentBlock>

            <AddUserToRoleDialog
                isOpen={isAddUserModalOpen}
                roleId={roleId}
                onClose={() => setAddUserModalOpen(false)}
                onAdd={handleAddUserToRole}
            />

            <Dialog open={!!userForRemove} onClose={() => setUserForRemove(null)} size='sm'>
                <DialogTitle>Remove User</DialogTitle>
                <DialogDescription>
                    Are you sure you want to remove {userForRemove?.username} from this role?
                </DialogDescription>
                <DialogActions>
                    <Button plain onClick={() => setUserForRemove(null)} disabled={isRemoving}>
                        Cancel
                    </Button>
                    <Button color='red' onClick={confirmRemoveUser} disabled={isRemoving}>
                        {isRemoving ? 'Removing...' : 'Remove'}
                    </Button>
                </DialogActions>
            </Dialog>
        </>
    );
};

export default RoleView;
