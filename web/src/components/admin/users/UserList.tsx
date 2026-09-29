import React, { useEffect, useState } from 'react';
import { Button } from '../../elements/Button';
import { Can } from '../../elements/Can';
import CreateUserForm from './CreateUserForm';
import { useUsers } from '../../../api/query/useUsers';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import { ErrorMessage } from '../../elements/Fieldset';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../elements/Table.tsx';
import { PaginationControls } from '../../elements/PaginationControls';
import { Heading } from '../../elements/Heading';
import { Text } from '../../elements/Text';

const DEFAULT_PER_PAGE = 25;

const UserList: React.FC = () => {
    const [page, setPage] = useState(1);
    const perPage = DEFAULT_PER_PAGE;

    const { data: userResult, error } = useUsers({ page, perPage });
    const { clearFlashes, clearAndAddHttpError } = useFlash();
    const [isCreateModalOpen, setCreateModalOpen] = useState(false);

    useEffect(() => {
        if (!error) {
            clearFlashes('users');
            return;
        }

        clearAndAddHttpError({ error, key: 'users' });
    }, [error, clearFlashes, clearAndAddHttpError]);

    const users = userResult?.items ?? [];
    const pagination = userResult?.pagination;
    useEffect(() => {
        if (pagination && pagination.totalPages > 0 && page > pagination.totalPages) {
            setPage(pagination.totalPages);
        }
    }, [pagination, page]);

    if (error && !userResult) {
        return <ErrorMessage>Error: {error.message}</ErrorMessage>;
    }

    if (!userResult) {
        return <LoadingSpinner />;
    }

    return (
        <div>
            <div className='mb-6 flex w-full flex-wrap items-end justify-between gap-4'>
                <div>
                    <Heading>Users</Heading>
                    <Text className='mt-1'>A list of all the users in the system.</Text>
                </div>
                <Can permission='user.create'>
                    <Button onClick={() => setCreateModalOpen(true)}>Create User</Button>
                </Can>
            </div>

            <FlashMessageRender byKey={'users'} className={'mb-4'} />

            {users.length === 0 ? (
                <p className='mt-4 text-gray-500'>No users found.</p>
            ) : (
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableHeader>Name</TableHeader>
                            <TableHeader>Username</TableHeader>
                            <TableHeader>Roles</TableHeader>
                            <TableHeader>Created At</TableHeader>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {users.map((user) => (
                            <TableRow key={user.id} href={`/admin/users/${user.id}`}>
                                <TableCell>{`${user.first_name} ${user.last_name}`.trim()}</TableCell>
                                <TableCell>{user.username}</TableCell>
                                <TableCell>{user.roles?.map((role) => role.name).join(', ') || 'No roles'}</TableCell>
                                <TableCell>{new Date(user.created_at).toLocaleDateString()}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            )}

            {pagination && pagination.totalPages > 1 && (
                <PaginationControls
                    className='mt-6'
                    pagination={pagination}
                    currentPage={page}
                    onPageChange={setPage}
                />
            )}

            <CreateUserForm isOpen={isCreateModalOpen} onClose={() => setCreateModalOpen(false)} />
        </div>
    );
};

export default UserList;
