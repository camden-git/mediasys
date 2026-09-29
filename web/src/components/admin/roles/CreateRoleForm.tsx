import React from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { createRole } from '../../../api/admin/roles';
import { queryKeys } from '../../../lib/queryKeys';
import RoleForm, { RoleFormValues } from './RoleForm';

interface CreateRoleFormProps {
    isOpen: boolean;
    onClose: () => void;
}

const initialValues: RoleFormValues = {
    name: '',
    global_permissions: [],
    global_album_permissions: [],
    album_permissions: [],
};

const CreateRoleForm: React.FC<CreateRoleFormProps> = ({ isOpen, onClose }) => {
    const queryClient = useQueryClient();

    return (
        <RoleForm
            isOpen={isOpen}
            onClose={onClose}
            title='Create New Role'
            description='Define a new role and its permissions.'
            initialValues={initialValues}
            flashKey='roles'
            successMessage='Role created successfully!'
            failureMessage='Failed to create role.'
            submitLabel='Create Role'
            submittingLabel='Creating...'
            onSave={async (values) => {
                await createRole(values);
                queryClient.invalidateQueries({ queryKey: queryKeys.roles.all() });
            }}
        />
    );
};

export default CreateRoleForm;
