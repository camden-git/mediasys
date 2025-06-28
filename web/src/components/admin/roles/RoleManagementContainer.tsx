import React, { useEffect, useState } from 'react';
import { Heading } from '../../elements/Heading';
import PageContentBlock from '../../elements/ContentBlock';
import { Can } from '../../elements/Can';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../elements/Table';
import { AdminRoleResponse } from '../../../types';
import { Button } from '../../elements/Button';
import CreateRoleForm from './CreateRoleForm';
import { Link } from 'react-router-dom';
import { useRoles } from '../../../api/swr/useRoles';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import { deleteRole } from '../../../api/admin/roles';
import { mutate } from 'swr';

const RoleManagementContainer: React.FC = () => {
    const { data: roles, error, isValidating } = useRoles();
    const { clearFlashes, clearAndAddHttpError, addFlash } = useFlash();
    const [showCreateModal, setShowCreateModal] = useState(false);

    useEffect(() => {
        if (!error) {
            clearFlashes('roles');
            return;
        }

        clearAndAddHttpError({ error, key: 'roles' });
    }, [error, clearFlashes, clearAndAddHttpError]);

    const handleDeleteRole = async (role: AdminRoleResponse) => {
        if (window.confirm(`Are you sure you want to delete role "${role.name}"? This action cannot be undone.`)) {
            try {
                await deleteRole(role.id);

                mutate('roles');

                addFlash({
                    key: 'roles',
                    type: 'success',
                    message: `Role "${role.name}" has been deleted.`,
                });
            } catch (error: any) {
                clearAndAddHttpError({ error, key: 'roles' });
            }
        }
    };

    if (!roles || (error && isValidating)) {
        return <LoadingSpinner />;
    }

    return (
        <PageContentBlock title={'Role Management'}>
            <div className='mb-16 flex w-full flex-wrap items-end justify-between gap-4'>
                <Heading>Roles</Heading>
                <div className='flex gap-4'>
                    <Can permission={'role.create'}>
                        <Button onClick={() => setShowCreateModal(true)}>Create Role</Button>
                    </Can>
                </div>
            </div>

            <FlashMessageRender byKey={'roles'} className={'mb-4'} />

            <CreateRoleForm isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} />

            <Can permission={'role.list'}>
                {roles.length === 0 ? (
                    <p>No roles found.</p>
                ) : (
                    <Table>
                        <TableHead>
                            <TableRow>
                                <TableHeader>ID</TableHeader>
                                <TableHeader>Name</TableHeader>
                                <TableHeader>Global Permissions</TableHeader>
                                <TableHeader>Album Permissions</TableHeader>
                                <TableHeader>Actions</TableHeader>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {roles.map((role: AdminRoleResponse) => (
                                <TableRow key={role.id}>
                                    <TableCell>{role.id}</TableCell>
                                    <TableCell className='font-medium'>
                                        <Can permission='role.view'>
                                            <Link to={`/admin/roles/${role.id}`} className='hover:underline'>
                                                {role.name}
                                            </Link>
                                        </Can>
                                    </TableCell>
                                    <TableCell>
                                        {role.global_permissions && role.global_permissions.length > 0
                                            ? role.global_permissions.join(', ')
                                            : 'None'}
                                    </TableCell>
                                    <TableCell>
                                        {role.album_permissions && role.album_permissions.length > 0
                                            ? `${role.album_permissions.length} album-specific rule(s)`
                                            : 'None'}
                                    </TableCell>
                                    <TableCell className='flex flex-wrap gap-x-2 gap-y-1'>
                                        <Can permission='role.view'>
                                            <Button plain to={`/admin/roles/${role.id}`} className='px-2 py-1 text-xs'>
                                                View
                                            </Button>
                                        </Can>
                                        <Can permission='role.edit'>
                                            <Button
                                                plain
                                                className='px-2 py-1 text-xs'
                                                onClick={() => alert(`Edit for role ${role.id} not implemented.`)}
                                            >
                                                Edit
                                            </Button>
                                        </Can>
                                        <Can permission='role.delete'>
                                            <Button
                                                color='red'
                                                className='px-2 py-1 text-xs'
                                                onClick={() => handleDeleteRole(role)}
                                            >
                                                Delete
                                            </Button>
                                        </Can>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                )}
            </Can>
        </PageContentBlock>
    );
};

export default RoleManagementContainer;
