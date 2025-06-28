import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { DescriptionList, DescriptionTerm, DescriptionDetails } from '../../elements/DescriptionList';
import { Heading } from '../../elements/Heading';
import ContentBlock from '../../elements/ContentBlock';
import { Button } from '../../elements/Button';
import { Can } from '../../elements/Can';
import EditRoleForm from './EditRoleForm';
import { useRole } from '../../../api/swr/useRoles';
import { useUsers } from '../../../api/swr/useUsers';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import { getRoleUsers, addUserToRole, removeUserFromRole } from '../../../api/admin/roles';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { Select } from '../../elements/Select.tsx';

const RoleView: React.FC = () => {
    const { id } = useParams<{ id: string }>();
    const roleId = id ? parseInt(id, 10) : 0;

    const { data: role, error, isValidating } = useRole(roleId);
    const { data: allUsers } = useUsers();
    const { addFlash, clearFlashes, clearAndAddHttpError } = useFlash();

    const [users, setUsers] = useState<any[]>([]);
    const [isLoadingUsers, setIsLoadingUsers] = useState(false);
    const [userError, setUserError] = useState<string | null>(null);

    const [isEditModalOpen, setEditModalOpen] = useState(false);
    const [isAddUserModalOpen, setAddUserModalOpen] = useState(false);

    const loadRoleUsers = async () => {
        if (!roleId) return;

        setIsLoadingUsers(true);
        setUserError(null);
        try {
            const roleUsers = await getRoleUsers(roleId);
            setUsers(roleUsers);
        } catch (error: any) {
            setUserError(error.message || 'Failed to load users');
        } finally {
            setIsLoadingUsers(false);
        }
    };

    useEffect(() => {
        if (roleId) {
            loadRoleUsers();
        }
    }, [roleId]);

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
        return <p style={{ color: 'red' }}>Error: {error.message}</p>;
    }

    const handleAddUserToRole = async (userId: number) => {
        try {
            await addUserToRole(roleId, userId);
            await loadRoleUsers();
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

    const handleRemoveUserFromRole = async (userId: number, username: string) => {
        if (window.confirm(`Are you sure you want to remove ${username} from this role?`)) {
            try {
                await removeUserFromRole(roleId, userId);
                await loadRoleUsers();
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
            }
        }
    };

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
                <DescriptionList className='mt-4'>
                    <DescriptionTerm>ID</DescriptionTerm>
                    <DescriptionDetails>{role.id}</DescriptionDetails>

                    <DescriptionTerm>Name</DescriptionTerm>
                    <DescriptionDetails>{role.name}</DescriptionDetails>

                    <DescriptionTerm>Global Permissions</DescriptionTerm>
                    <DescriptionDetails>
                        {role.global_permissions && role.global_permissions.length > 0 ? (
                            <ul className='list-inside list-disc'>
                                {role.global_permissions.map((p) => (
                                    <li key={p}>{p}</li>
                                ))}
                            </ul>
                        ) : (
                            'None'
                        )}
                    </DescriptionDetails>

                    <DescriptionTerm>Global Album Permissions</DescriptionTerm>
                    <DescriptionDetails>
                        {role.global_album_permissions && role.global_album_permissions.length > 0 ? (
                            <ul className='list-inside list-disc'>
                                {role.global_album_permissions.map((p) => (
                                    <li key={p}>{p}</li>
                                ))}
                            </ul>
                        ) : (
                            'None'
                        )}
                    </DescriptionDetails>

                    <DescriptionTerm>Album-Specific Permissions</DescriptionTerm>
                    <DescriptionDetails>
                        {role.album_permissions && role.album_permissions.length > 0 ? (
                            <div className='space-y-2'>
                                {role.album_permissions.map((ap) => (
                                    <div key={ap.album_id}>
                                        <h4 className='font-semibold'>Album ID: {ap.album_id}</h4>
                                        <ul className='list-inside list-disc pl-4'>
                                            {ap.permissions.map((p) => (
                                                <li key={p}>{p}</li>
                                            ))}
                                        </ul>
                                    </div>
                                ))}
                            </div>
                        ) : (
                            'None'
                        )}
                    </DescriptionDetails>
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
                        {userError && <p style={{ color: 'red' }}>Error: {userError}</p>}
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
            </ContentBlock>

            {isAddUserModalOpen && (
                <div
                    className='fixed inset-0 z-10 bg-zinc-400/25 backdrop-blur-sm dark:bg-black/40'
                    aria-hidden='true'
                />
            )}
            {isAddUserModalOpen && (
                <div
                    className='fixed inset-0 z-10 w-screen overflow-y-auto p-4 sm:p-6 md:p-20'
                    role='dialog'
                    aria-modal='true'
                >
                    <div className='mx-auto max-w-lg transform rounded-xl bg-white p-6 shadow-2xl ring-1 ring-black/5 transition-all dark:bg-zinc-900'>
                        <h3 className='text-lg leading-6 font-medium'>Add User to Role</h3>
                        <div className='mt-4'>
                            {!allUsers ? (
                                <p className='mb-4 text-sm text-gray-500'>Loading users...</p>
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
                        </div>
                        <div className='mt-6 flex justify-end'>
                            <Button plain onClick={() => setAddUserModalOpen(false)}>
                                Cancel
                            </Button>
                        </div>
                    </div>
                </div>
            )}
        </>
    );
};

export default RoleView;
