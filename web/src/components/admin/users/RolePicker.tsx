import React from 'react';
import { Field, Label } from '../../elements/Fieldset';
import { CheckboxField, Checkbox } from '../../elements/Checkbox';
import { useAllRoles } from '../../../api/query/useRoles';

interface RolePickerProps {
    selected: number[];
    disabled: boolean;
    onChange: (ids: number[]) => void;
}

// Lists every role (all pages). Only render this for users allowed to list roles.
const RolePicker: React.FC<RolePickerProps> = ({ selected, disabled, onChange }) => {
    const { data: roles = [], isLoading, isError } = useAllRoles();

    return (
        <Field>
            <Label>Roles</Label>
            {isLoading && <p className='mt-2 text-sm text-zinc-500'>Loading roles...</p>}
            {isError && <p className='mt-2 text-sm text-red-600'>Failed to load roles.</p>}
            <div className='mt-3 grid max-h-60 grid-cols-2 gap-2 overflow-y-auto rounded border border-zinc-950/10 p-2 dark:border-white/10'>
                {roles.map((role) => (
                    <CheckboxField key={role.id}>
                        <Checkbox
                            checked={selected.includes(role.id)}
                            onChange={(checked) =>
                                onChange(checked ? [...selected, role.id] : selected.filter((id) => id !== role.id))
                            }
                            disabled={disabled}
                        />
                        <Label>{role.name}</Label>
                    </CheckboxField>
                ))}
            </div>
        </Field>
    );
};

export default RolePicker;
