import React, { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { DescriptionList, DescriptionTerm, DescriptionDetails } from '../../elements/DescriptionList';
import { ErrorMessage } from '../../elements/Fieldset';
import { Heading } from '../../elements/Heading';
import ContentBlock from '../../elements/PageContentBlock.tsx';
import { Button } from '../../elements/Button';
import { Can } from '../../elements/Can';
import EditRoleForm from './EditRoleForm';
import { useRole } from '../../../api/query/useRoles';
import { useUsers } from '../../../api/query/useUsers';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import { getRoleUsers, addUserToRole, removeUserFromRole } from '../../../api/admin/roles';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { Select } from '../../elements/Select.tsx';
import { Text } from '../../elements/Text.tsx';
import { PaginatedResult } from '../../../api/standard';
import { UserSummary } from '../../../types';
import { PaginationControls } from '../../elements/PaginationControls';
import { Dialog, DialogActions, DialogDescription, DialogTitle } from '../../elements/Dialog';

const RoleView: React.FC = () => {
    const { id } = useParams<{ id: string }>();
    const roleId = id ? parseInt(id, 10) : 0;

    const { data: role, error, isFetching: isValidating } = useRole(roleId);
    const { data: allUsersResult } = useUsers({ perPage: 500 });
    const allUsers = allUsersResult?.items ?? [];
    const { addFlash, clearFlashes, clearAndAddHttpError } = useFlash();

    const [userResult, setUserResult] = useState<PaginatedResult<UserSummary> | null>(null);
    const [isLoadingUsers, setIsLoadingUsers] = useState(false);
    const [userError, setUserError] = useState<string | null>(null);
    const [userPage, setUserPage] = useState(1);
    const userPerPage = 25;

    const [isEditModalOpen, setEditModalOpen] = useState(false);
    const [isAddUserModalOpen, setAddUserModalOpen] = useState(false);
    const [userForRemove, setUserForRemove] = useState<{ id: number; username: string } | null>(null);

    const loadRoleUsers = useCallback(
        async (pageToLoad: number) => {
            if (!roleId) return;

            setIsLoadingUsers(true);
            setUserError(null);
            try {
                const result = await getRoleUsers(roleId, { page: pageToLoad, perPage: userPerPage });
                setUserResult(result);
            } catch (error: any) {
                setUserError(error.message || 'Failed to load users');
            } finally {
                setIsLoadingUsers(false);
            }
        },
        [roleId],
    );

    useEffect(() => {
        if (roleId) {
            loadRoleUsers(userPage);
        }
    }, [roleId, userPage, loadRoleUsers]);

    useEffect(() => {
        if (!error) {
            clearFlashes('role-view');
            return;
        }

        clearAndAddHttpError({ error, key: 'role-view' });
    }, [error, clearFlashes, clearAndAddHttpError]);

    if (!role || (error && isValidating)) {
        return <LoadingSpinner />;
    }

    if (error) {
        return <ErrorMessage>Error: {error.message}</ErrorMessage>;
    }

    const handleAddUserToRole = async (userId: number) => {
        try {
            await addUserToRole(roleId, userId);
            await loadRoleUsers(userPage);
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
        if (!userForRemove) return;
        try {
            await removeUserFromRole(roleId, userForRemove.id);
            await loadRoleUsers(userPage);
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
                    <Can permission='role.edit.users'>
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
                        onPageChange={(page) => {
                            setUserPage(page);
                            loadRoleUsers(page);
                        }}
                    />
                )}
            </ContentBlock>

            <Dialog open={isAddUserModalOpen} onClose={setAddUserModalOpen} size='sm'>
                <DialogTitle>Add User to Role</DialogTitle>
                <DialogActions>
                    {!allUsersResult ? (
                        <p className='text-sm text-gray-500'>Loading users...</p>
                    ) : (
                        <Select
                            onChange={async (e) => {
                                const userId = parseInt(e.target.value, 10);
                                if (userId) {
                                    await handleAddUserToRole(userId);
                                }
                            }}
                        >
                            <option value=''>Select a user...</option>
                            {allUsers
                                .filter((user) => !users.some((roleUser) => roleUser.id === user.id))
                                .map((user) => (
                                    <option key={user.id} value={user.id}>
                                        {user.username}
                                    </option>
                                ))}
                        </Select>
                    )}
                    <Button plain onClick={() => setAddUserModalOpen(false)}>
                        Cancel
                    </Button>
                </DialogActions>
            </Dialog>

            <Dialog open={!!userForRemove} onClose={() => setUserForRemove(null)} size='sm'>
                <DialogTitle>Remove User</DialogTitle>
                <DialogDescription>
                    Are you sure you want to remove {userForRemove?.username} from this role?
                </DialogDescription>
                <DialogActions>
                    <Button plain onClick={() => setUserForRemove(null)}>
                        Cancel
                    </Button>
                    <Button color='red' onClick={confirmRemoveUser}>
                        Remove
                    </Button>
                </DialogActions>
            </Dialog>
        </>
    );
};

export default RoleView;
