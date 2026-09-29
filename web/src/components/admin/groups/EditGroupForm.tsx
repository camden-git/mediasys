import React from 'react';
import { updateGroup } from '../../../api/admin/groups';
import { AlbumGroup } from '../../../types';
import GroupForm from './GroupForm';

interface EditGroupFormProps {
    isOpen: boolean;
    onClose: () => void;
    onUpdated: () => void;
    group?: AlbumGroup;
}

const EditGroupForm: React.FC<EditGroupFormProps> = ({ isOpen, onClose, onUpdated, group }) => {
    if (!isOpen || !group) return null;

    return (
        <GroupForm
            isOpen={isOpen}
            onClose={onClose}
            title={`Edit Group: ${group.name}`}
            description='Update the group details.'
            initialValues={{
                name: group.name,
                slug: group.slug,
                description: group.description ?? '',
                is_hidden: group.is_hidden,
            }}
            enableReinitialize
            successMessage='Group updated successfully!'
            failureMessage='Failed to update group.'
            submitLabel='Save Changes'
            submittingLabel='Saving...'
            onSave={(values) => updateGroup(group.id, values)}
            onSaved={onUpdated}
        />
    );
};

export default EditGroupForm;
