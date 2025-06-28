import React, { useState } from 'react';
import { Button } from '../../elements/Button';
import { Dialog, DialogActions, DialogBody, DialogTitle, DialogDescription } from '../../elements/Dialog';
import { Field, FieldGroup, Label, ErrorMessage as FieldErrorMessage } from '../../elements/Fieldset';
import { Input } from '../../elements/Input';
import { Formik, Form, FieldArray, ErrorMessage } from 'formik';
import * as Yup from 'yup';
import { AdminRoleResponse, Album, RoleUpdatePayload } from '../../../types';
import { updateRole } from '../../../api/admin/roles';
import { useFlash } from '../../../hooks/useFlash';
import { useRoles, usePermissionDefinitions } from '../../../api/swr/useRoles';
import { useAlbums } from '../../../api/swr/useAlbums';
import { Select } from '../../elements/Select.tsx';

const RoleUpdateSchema = Yup.object().shape({
    name: Yup.string().required('Role name is required.'),
    global_permissions: Yup.array().of(Yup.string()),
    global_album_permissions: Yup.array().of(Yup.string()),
    album_permissions: Yup.array().of(
        Yup.object().shape({
            album_id: Yup.number().required('Album selection is required.'),
            permissions: Yup.array().of(Yup.string()).min(1, 'At least one permission is required for an album rule.'),
        }),
    ),
});

interface EditRoleFormProps {
    isOpen: boolean;
    onClose: () => void;
    role: AdminRoleResponse; // The role to edit
}

const EditRoleForm: React.FC<EditRoleFormProps> = ({ isOpen, onClose, role }) => {
    const [isSubmitting, setIsSubmitting] = useState(false);

    const { addFlash, clearFlashes } = useFlash();
    const { mutate } = useRoles();
    const { data: permissionDefinitions } = usePermissionDefinitions();
    const { data: albums, isLoading: isLoadingAlbums, error: albumError } = useAlbums();

    // Set initial values from the role prop
    const initialValues: RoleUpdatePayload = {
        name: role.name,
        global_permissions: role.global_permissions || [],
        global_album_permissions: role.global_album_permissions || [],
        album_permissions:
            role.album_permissions.map((ap) => ({
                id: ap.id,
                album_id: ap.album_id,
                permissions: ap.permissions || [],
            })) || [],
    };

    const getPermissionsByScope = (scope: 'global' | 'album'): { key: string; name: string }[] => {
        const perms: { key: string; name: string }[] = [];
        permissionDefinitions?.forEach((group) => {
            group.permissions.forEach((p) => {
                if (p.scope === scope) {
                    perms.push({ key: p.key, name: p.name });
                }
            });
        });
        return perms.sort((a, b) => a.name.localeCompare(b.name));
    };

    const globalPermissionsOptions = getPermissionsByScope('global');
    const albumPermissionsOptions = getPermissionsByScope('album');

    if (!isOpen) return null;

    return (
        <Dialog open={isOpen} onClose={onClose} size='2xl'>
            <Formik
                initialValues={initialValues}
                validationSchema={RoleUpdateSchema}
                enableReinitialize
                onSubmit={async (values) => {
                    setIsSubmitting(true);
                    clearFlashes('role-edit');

                    try {
                        const updatedRole = await updateRole(role.id, values);

                        mutate((currentData) => {
                            if (!currentData) return [updatedRole];
                            return currentData.map((r) => (r.id === updatedRole.id ? updatedRole : r));
                        }, false);

                        addFlash({
                            key: 'role-edit',
                            type: 'success',
                            message: 'Role updated successfully!',
                        });
                        onClose();
                    } catch (error: any) {
                        addFlash({
                            key: 'role-edit',
                            type: 'error',
                            message: error.message || 'Failed to update role.',
                        });
                    } finally {
                        setIsSubmitting(false);
                    }
                }}
            >
                {({ values, handleChange, handleBlur, setFieldValue }) => (
                    <Form>
                        <DialogTitle>Edit Role: {role.name}</DialogTitle>
                        <DialogDescription>Modify the role and its permissions.</DialogDescription>
                        <DialogBody className='max-h-[70vh] overflow-y-auto'>
                            <FieldGroup>
                                <Field>
                                    <Label htmlFor='name'>Role Name</Label>
                                    <Input
                                        id='name'
                                        name='name'
                                        type='text'
                                        value={values.name}
                                        onChange={handleChange}
                                        onBlur={handleBlur}
                                        disabled={isSubmitting}
                                    />
                                    <ErrorMessage name='name' component={FieldErrorMessage} />
                                </Field>

                                <Field>
                                    <Label>Global Permissions</Label>
                                    {!permissionDefinitions && <p>Loading permissions...</p>}
                                    <div className='mt-1 grid max-h-60 grid-cols-2 gap-2 overflow-y-auto rounded border p-2'>
                                        {globalPermissionsOptions.map((perm) => (
                                            <label key={perm.key} className='flex items-center space-x-2 text-sm'>
                                                <input
                                                    type='checkbox'
                                                    name='global_permissions'
                                                    value={perm.key}
                                                    checked={values.global_permissions?.includes(perm.key)}
                                                    onChange={handleChange}
                                                    disabled={isSubmitting}
                                                    className='rounded'
                                                />
                                                <span>
                                                    {perm.name}{' '}
                                                    <span className='text-xs text-gray-500'>({perm.key})</span>
                                                </span>
                                            </label>
                                        ))}
                                    </div>
                                </Field>

                                <Field>
                                    <Label>Global Album Permissions</Label>
                                    {!permissionDefinitions && <p>Loading permissions...</p>}
                                    <div className='mt-1 grid max-h-60 grid-cols-2 gap-2 overflow-y-auto rounded border p-2'>
                                        {albumPermissionsOptions.map((perm) => (
                                            <label key={perm.key} className='flex items-center space-x-2 text-sm'>
                                                <input
                                                    type='checkbox'
                                                    name='global_album_permissions'
                                                    value={perm.key}
                                                    checked={values.global_album_permissions?.includes(perm.key)}
                                                    onChange={handleChange}
                                                    disabled={isSubmitting}
                                                    className='rounded'
                                                />
                                                <span>
                                                    {perm.name}{' '}
                                                    <span className='text-xs text-gray-500'>({perm.key})</span>
                                                </span>
                                            </label>
                                        ))}
                                    </div>
                                </Field>

                                <Field>
                                    <Label>Album Specific Permissions</Label>
                                    <FieldArray name='album_permissions'>
                                        {({ push, remove }) => (
                                            <div className='mt-2 space-y-4'>
                                                {values.album_permissions?.map((ap, index) => (
                                                    <div
                                                        key={index}
                                                        className='space-y-3 rounded-md border bg-gray-50 p-3 dark:bg-zinc-800/50'
                                                    >
                                                        <div className='flex items-start justify-between'>
                                                            <h4 className='text-sm font-medium'>
                                                                Album Rule #{index + 1}
                                                            </h4>
                                                            <Button
                                                                type='button'
                                                                plain
                                                                onClick={() => remove(index)}
                                                                disabled={isSubmitting}
                                                                className='text-xs text-red-600 hover:text-red-800 dark:text-red-500 dark:hover:text-red-400'
                                                            >
                                                                Remove Rule
                                                            </Button>
                                                        </div>
                                                        <Field>
                                                            <Label htmlFor={`album_permissions.${index}.album_id`}>
                                                                Album
                                                            </Label>
                                                            <Select
                                                                id={`album_permissions.${index}.album_id`}
                                                                name={`album_permissions.${index}.album_id`}
                                                                value={ap.album_id?.toString() || ''}
                                                                onChange={(e) =>
                                                                    setFieldValue(
                                                                        `album_permissions.${index}.album_id`,
                                                                        e.target.value
                                                                            ? parseInt(e.target.value, 10)
                                                                            : 0,
                                                                    )
                                                                }
                                                                disabled={isSubmitting || isLoadingAlbums}
                                                            >
                                                                <option value=''>Select an Album</option>
                                                                {isLoadingAlbums && (
                                                                    <option value='' disabled>
                                                                        Loading albums...
                                                                    </option>
                                                                )}
                                                                {!isLoadingAlbums && albumError && (
                                                                    <option value='' disabled>
                                                                        Error loading albums
                                                                    </option>
                                                                )}
                                                                {!isLoadingAlbums &&
                                                                    !albumError &&
                                                                    albums?.map((album: Album) => (
                                                                        <option
                                                                            key={album.id}
                                                                            value={album.id.toString()}
                                                                        >
                                                                            {album.name} (ID: {album.id})
                                                                        </option>
                                                                    ))}
                                                            </Select>
                                                            <ErrorMessage
                                                                name={`album_permissions.${index}.album_id`}
                                                                component={FieldErrorMessage}
                                                            />
                                                        </Field>
                                                        <Field>
                                                            <Label>Permissions for this Album</Label>
                                                            <div className='mt-1 grid max-h-40 grid-cols-2 gap-2 overflow-y-auto rounded border p-2'>
                                                                {albumPermissionsOptions.map((perm) => (
                                                                    <label
                                                                        key={perm.key}
                                                                        className='flex items-center space-x-2 text-sm'
                                                                    >
                                                                        <input
                                                                            type='checkbox'
                                                                            name={`album_permissions.${index}.permissions`}
                                                                            value={perm.key}
                                                                            checked={ap.permissions.includes(perm.key)}
                                                                            onChange={handleChange}
                                                                            disabled={isSubmitting}
                                                                            className='rounded'
                                                                        />
                                                                        <span>
                                                                            {perm.name}{' '}
                                                                            <span className='text-xs text-gray-500'>
                                                                                ({perm.key})
                                                                            </span>
                                                                        </span>
                                                                    </label>
                                                                ))}
                                                            </div>
                                                            <ErrorMessage
                                                                name={`album_permissions.${index}.permissions`}
                                                                component={FieldErrorMessage}
                                                            />
                                                        </Field>
                                                    </div>
                                                ))}
                                                <Button
                                                    type='button'
                                                    onClick={() => push({ album_id: 0, permissions: [] })}
                                                    disabled={isSubmitting}
                                                    className='mt-2'
                                                >
                                                    Add Album Permission Rule
                                                </Button>
                                            </div>
                                        )}
                                    </FieldArray>
                                </Field>
                            </FieldGroup>
                        </DialogBody>
                        <DialogActions>
                            <Button
                                plain
                                onClick={() => {
                                    onClose();
                                    clearFlashes('role-edit');
                                }}
                                disabled={isSubmitting}
                            >
                                Cancel
                            </Button>
                            <Button type='submit' disabled={isSubmitting || !permissionDefinitions || isLoadingAlbums}>
                                {isSubmitting ? 'Saving...' : 'Save Changes'}
                            </Button>
                        </DialogActions>
                    </Form>
                )}
            </Formik>
        </Dialog>
    );
};

export default EditRoleForm;
