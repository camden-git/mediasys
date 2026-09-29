import React from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { AdminRoleResponse } from '../../../types';
import { updateRole } from '../../../api/admin/roles';
import { queryKeys } from '../../../lib/queryKeys';
import { refreshAuthUser } from '../../../store/useAuthStore';
import RoleForm from './RoleForm';

interface EditRoleFormProps {
    isOpen: boolean;
    onClose: () => void;
    role?: AdminRoleResponse; // The role to edit
}

const EditRoleForm: React.FC<EditRoleFormProps> = ({ isOpen, onClose, role }) => {
    const queryClient = useQueryClient();

    if (!isOpen || !role) {
        return null;
    }

    return (
        <RoleForm
            isOpen={isOpen}
            onClose={onClose}
            title={`Edit Role: ${role.name}`}
            description='Modify the role and its permissions.'
            initialValues={{
                name: role.name,
                global_permissions: role.global_permissions || [],
                global_album_permissions: role.global_album_permissions || [],
                album_permissions: role.album_permissions.map((ap) => ({
                    id: ap.id,
                    album_id: ap.album_id,
                    permissions: ap.permissions || [],
                })),
            }}
            enableReinitialize
            flashKey='role-edit'
            successMessage='Role updated successfully!'
            failureMessage='Failed to update role.'
            submitLabel='Save Changes'
            submittingLabel='Saving...'
            onSave={async (values) => {
                await updateRole(role.id, values);
                queryClient.invalidateQueries({ queryKey: queryKeys.roles.all() });
                queryClient.invalidateQueries({ queryKey: queryKeys.users.all() });
                refreshAuthUser();
            }}
        />
    );
};

export default EditRoleForm;
