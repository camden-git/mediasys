import React from 'react';
import { createGroup, GroupCreatePayload } from '../../../api/admin/groups';
import GroupForm from './GroupForm';

interface CreateGroupFormProps {
    isOpen: boolean;
    onClose: () => void;
    onCreated: () => void;
}

const initialValues: GroupCreatePayload = {
    name: '',
    slug: '',
    description: '',
    is_hidden: false,
};

const CreateGroupForm: React.FC<CreateGroupFormProps> = ({ isOpen, onClose, onCreated }) => (
    <GroupForm
        isOpen={isOpen}
        onClose={onClose}
        title='Create Group'
        description='Create a new album group to organise related albums.'
        initialValues={initialValues}
        successMessage='Group created successfully!'
        failureMessage='Failed to create group.'
        submitLabel='Create Group'
        submittingLabel='Creating...'
        onSave={createGroup}
        onSaved={onCreated}
    />
);

export default CreateGroupForm;
