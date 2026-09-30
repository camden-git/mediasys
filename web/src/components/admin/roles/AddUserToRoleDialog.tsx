import React, { useMemo, useState } from 'react';
import { Button } from '../../elements/Button';
import { Dialog, DialogActions, DialogBody, DialogTitle } from '../../elements/Dialog';
import { Field, Label } from '../../elements/Fieldset';
import { Input } from '../../elements/Input';
import { Select } from '../../elements/Select';
import { useAllUsers } from '../../../api/query/useUsers';
import { useAllRoleUsers } from '../../../api/query/useRoles';

interface AddUserToRoleDialogProps {
    isOpen: boolean;
    roleId: number;
    onClose: () => void;
    onAdd: (userId: number) => Promise<void>;
}

const AddUserToRoleDialog: React.FC<AddUserToRoleDialogProps> = ({ isOpen, roleId, onClose, onAdd }) => {
    // The lists are only fetched while the dialog is open; both walk every page so the picker
    // neither truncates at the page size nor mistakes members on other pages for non-members.
    const { data: allUsers, isError: usersFailed } = useAllUsers({ enabled: isOpen });
    const { data: members, isError: membersFailed } = useAllRoleUsers(roleId, { enabled: isOpen });
    const [search, setSearch] = useState('');
    const [isAdding, setIsAdding] = useState(false);

    const candidates = useMemo(() => {
        if (!allUsers || !members) return null;
        const memberIds = new Set(members.map((m) => m.id));
        const needle = search.trim().toLowerCase();
        return allUsers.filter((u) => !memberIds.has(u.id) && (!needle || u.username.toLowerCase().includes(needle)));
    }, [allUsers, members, search]);

    const handleClose = () => {
        if (isAdding) return;
        setSearch('');
        onClose();
    };

    const handleSelect = async (e: React.ChangeEvent<HTMLSelectElement>) => {
        const userId = parseInt(e.target.value, 10);
        if (!userId || isAdding) return;
        setIsAdding(true);
        try {
            await onAdd(userId);
            setSearch('');
        } finally {
            setIsAdding(false);
        }
    };

    return (
        <Dialog open={isOpen} onClose={handleClose} size='sm'>
            <DialogTitle>Add User to Role</DialogTitle>
            <DialogBody>
                {usersFailed || membersFailed ? (
                    <p className='text-sm text-red-600'>Failed to load users.</p>
                ) : !candidates ? (
                    <p className='text-sm text-gray-500'>Loading users...</p>
                ) : (
                    <div className='space-y-4'>
                        <Field>
                            <Label>Search users</Label>
                            <Input
                                type='search'
                                value={search}
                                onChange={(e) => setSearch(e.target.value)}
                                placeholder='Filter by username'
                                disabled={isAdding}
                            />
                        </Field>
                        <Field>
                            <Label>User</Label>
                            <Select
                                value=''
                                onChange={handleSelect}
                                disabled={isAdding}
                                aria-label='Select a user to add'
                            >
                                <option value=''>{isAdding ? 'Adding...' : 'Select a user...'}</option>
                                {candidates.map((user) => (
                                    <option key={user.id} value={user.id}>
                                        {user.username}
                                    </option>
                                ))}
                            </Select>
                        </Field>
                        {candidates.length === 0 && (
                            <p className='text-sm text-gray-500'>No matching users available to add.</p>
                        )}
                    </div>
                )}
            </DialogBody>
            <DialogActions>
                <Button plain onClick={handleClose} disabled={isAdding}>
                    Cancel
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default AddUserToRoleDialog;
