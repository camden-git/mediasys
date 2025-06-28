import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Button } from '../../elements/Button';
import { Can } from '../../elements/Can';
import CreateUserForm from './CreateUserForm';
import { useUsers } from '../../../api/swr/useUsers';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../elements/Table.tsx';

const UserList: React.FC = () => {
    const { data: users, error, isValidating } = useUsers();
    const { clearFlashes, clearAndAddHttpError } = useFlash();
    const [isCreateModalOpen, setCreateModalOpen] = useState(false);

    useEffect(() => {
        if (!error) {
            clearFlashes('users');
            return;
        }

        clearAndAddHttpError({ error, key: 'users' });
    }, [error, clearFlashes, clearAndAddHttpError]);

    if (!users || (error && isValidating)) {
        return <LoadingSpinner />;
    }

    return (
        <div>
            <div className='sm:flex sm:items-center sm:justify-between'>
                <div>
                    <h2 className='text-lg font-medium text-gray-900 dark:text-white'>Users</h2>
                    <p className='mt-1 text-sm text-gray-500 dark:text-zinc-400'>
                        A list of all the users in the system.
                    </p>
                </div>
                <Can permission='user.create'>
                    <Button onClick={() => setCreateModalOpen(true)} className='mt-4 sm:mt-0'>
                        Create User
                    </Button>
                </Can>
            </div>

            <FlashMessageRender byKey={'users'} className={'mb-4'} />

            {users.length === 0 ? (
                <p className='mt-4 text-gray-500'>No users found.</p>
            ) : (
                <div className='mt-8 flow-root'>
                    <div className='-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8'>
                        <div className='inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8'>
                            <Table>
                                <TableHead>
                                    <TableRow>
                                        <TableHeader>Username</TableHeader>
                                        <TableHeader>Roles</TableHeader>
                                        <TableHeader>Created At</TableHeader>
                                        <TableHeader>
                                            <span className='sr-only'>View</span>
                                        </TableHeader>
                                    </TableRow>
                                </TableHead>
                                <TableBody className='divide-y divide-gray-200 dark:divide-zinc-800'>
                                    {users.map((user) => (
                                        <TableRow key={user.id}>
                                            <TableCell>{user.username}</TableCell>
                                            <TableCell>
                                                {user.roles?.map((role) => role.name).join(', ') || 'No roles'}
                                            </TableCell>
                                            <TableCell>{new Date(user.created_at).toLocaleDateString()}</TableCell>
                                            <TableCell>
                                                <Can permission='user.view'>
                                                    <Link to={`/admin/users/${user.id}`}>
                                                        <Button plain>
                                                            View<span className='sr-only'>, {user.username}</span>
                                                        </Button>
                                                    </Link>
                                                </Can>
                                            </TableCell>
                                        </TableRow>
                                    ))}
                                </TableBody>
                            </Table>
                        </div>
                    </div>
                </div>
            )}

            <CreateUserForm isOpen={isCreateModalOpen} onClose={() => setCreateModalOpen(false)} />
        </div>
    );
};

export default UserList;
