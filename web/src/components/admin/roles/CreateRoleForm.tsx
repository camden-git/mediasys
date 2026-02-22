import React, { useEffect, useMemo, useState } from 'react';
import { Button } from '../../elements/Button';
import { Dialog, DialogActions, DialogBody, DialogTitle, DialogDescription } from '../../elements/Dialog';
import { Field, FieldGroup, Label, ErrorMessage as FieldErrorMessage, Description } from '../../elements/Fieldset';
import { CheckboxField, Checkbox } from '../../elements/Checkbox';
import { Input } from '../../elements/Input';
import { Formik, Form, FieldArray, ErrorMessage } from 'formik';
import * as Yup from 'yup';
import { RoleCreatePayload, Album } from '../../../types';
import { createRole } from '../../../api/admin/roles';
import { useFlash } from '../../../hooks/useFlash';
import { usePermissionDefinitions } from '../../../api/swr/useRoles';
import { useAlbums } from '../../../api/swr/useAlbums';
import { Select } from '../../elements/Select.tsx';
import { useSWRConfig } from 'swr';

const RoleCreationSchema = Yup.object().shape({
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

interface CreateRoleFormProps {
    isOpen: boolean;
    onClose: () => void;
}

const CreateRoleForm: React.FC<CreateRoleFormProps> = ({ isOpen, onClose }) => {
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [permissionFilter, setPermissionFilter] = useState('');

    const { addFlash, clearFlashes } = useFlash();
    const { mutate } = useSWRConfig();
    const { data: permissionDefinitions } = usePermissionDefinitions();
    const { albums, isLoading: isLoadingAlbums, error: albumError } = useAlbums();

    const initialValues: RoleCreatePayload = {
        name: '',
        global_permissions: [],
        global_album_permissions: [],
        album_permissions: [],
    };

    const permissionOptions = useMemo(() => {
        const bucket: Record<'global' | 'album', { key: string; name: string }[]> = {
            global: [],
            album: [],
        };
        permissionDefinitions?.forEach((group) => {
            group.permissions.forEach((perm) => {
                bucket[perm.scope].push({ key: perm.key, name: perm.name });
            });
        });
        bucket.global.sort((a, b) => a.name.localeCompare(b.name));
        bucket.album.sort((a, b) => a.name.localeCompare(b.name));
        return bucket;
    }, [permissionDefinitions]);

    const filterPermissions = (list: { key: string; name: string }[]) => {
        if (!permissionFilter.trim()) {
            return list;
        }
        const query = permissionFilter.trim().toLowerCase();
        return list.filter((perm) => perm.name.toLowerCase().includes(query) || perm.key.toLowerCase().includes(query));
    };

    const globalPermissionsOptions = filterPermissions(permissionOptions.global);
    const albumPermissionsOptions = filterPermissions(permissionOptions.album);

    useEffect(() => {
        if (!isOpen) {
            setPermissionFilter('');
        }
    }, [isOpen]);

    if (!isOpen) return null;

    return (
        <Dialog open={isOpen} onClose={onClose} size='2xl'>
            <Formik
                initialValues={initialValues}
                validationSchema={RoleCreationSchema}
                onSubmit={async (values, { resetForm }) => {
                    setIsSubmitting(true);
                    clearFlashes('roles');

                    try {
                        await createRole(values);
                        mutate((key) => Array.isArray(key) && key[0] === 'roles');

                        addFlash({
                            key: 'roles',
                            type: 'success',
                            message: 'Role created successfully!',
                        });
                        resetForm();
                        onClose();
                    } catch (error: any) {
                        addFlash({
                            key: 'roles',
                            type: 'error',
                            message: error.message || 'Failed to create role.',
                        });
                    } finally {
                        setIsSubmitting(false);
                    }
                }}
            >
                {({ values, handleChange, handleBlur, setFieldValue }) => (
                    <Form>
                        <DialogTitle>Create New Role</DialogTitle>
                        <DialogDescription>Define a new role and its permissions.</DialogDescription>
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
                                    <Label htmlFor='permission-filter'>Filter Permissions</Label>
                                    <Input
                                        id='permission-filter'
                                        type='search'
                                        value={permissionFilter}
                                        placeholder='Search permissions by name or key…'
                                        onChange={(event) => setPermissionFilter(event.target.value)}
                                        disabled={!permissionDefinitions}
                                    />
                                </Field>

                                <Field>
                                    <Label>Global Permissions</Label>
                                    <div className='mt-3 grid max-h-60 grid-cols-2 gap-2 overflow-y-auto rounded border border-zinc-950/10 p-2 dark:border-white/10'>
                                        {globalPermissionsOptions.map((perm) => (
                                            <CheckboxField key={perm.key}>
                                                <Checkbox
                                                    checked={values.global_permissions.includes(perm.key)}
                                                    onChange={(checked) => {
                                                        const current = values.global_permissions;
                                                        setFieldValue(
                                                            'global_permissions',
                                                            checked
                                                                ? [...current, perm.key]
                                                                : current.filter((k) => k !== perm.key),
                                                        );
                                                    }}
                                                    disabled={isSubmitting}
                                                />
                                                <Label>
                                                    {perm.name}{' '}
                                                    <span className='text-xs text-zinc-500'>({perm.key})</span>
                                                </Label>
                                            </CheckboxField>
                                        ))}
                                    </div>
                                    <ErrorMessage name='global_permissions' component={FieldErrorMessage} />
                                </Field>

                                <Field>
                                    <Label>Global Album Permissions</Label>
                                    <Description>These permissions apply to ALL albums.</Description>
                                    <div className='mt-3 grid max-h-60 grid-cols-2 gap-2 overflow-y-auto rounded border border-zinc-950/10 p-2 dark:border-white/10'>
                                        {albumPermissionsOptions.map((perm) => (
                                            <CheckboxField key={perm.key}>
                                                <Checkbox
                                                    checked={values.global_album_permissions.includes(perm.key)}
                                                    onChange={(checked) => {
                                                        const current = values.global_album_permissions;
                                                        setFieldValue(
                                                            'global_album_permissions',
                                                            checked
                                                                ? [...current, perm.key]
                                                                : current.filter((k) => k !== perm.key),
                                                        );
                                                    }}
                                                    disabled={isSubmitting}
                                                />
                                                <Label>
                                                    {perm.name}{' '}
                                                    <span className='text-xs text-zinc-500'>({perm.key})</span>
                                                </Label>
                                            </CheckboxField>
                                        ))}
                                    </div>
                                    <ErrorMessage name='global_album_permissions' component={FieldErrorMessage} />
                                </Field>

                                <Field>
                                    <Label>Album Specific Permissions</Label>
                                    <Description>Define permissions for specific albums.</Description>
                                    <FieldArray name='album_permissions'>
                                        {({ push, remove }) => (
                                            <div className='mt-2 space-y-4'>
                                                {values.album_permissions.map((ap, index) => (
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
                                                                onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
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
                                                            <div className='mt-3 grid max-h-40 grid-cols-2 gap-2 overflow-y-auto rounded border border-zinc-950/10 p-2 dark:border-white/10'>
                                                                {albumPermissionsOptions.map((perm) => (
                                                                    <CheckboxField key={perm.key}>
                                                                        <Checkbox
                                                                            checked={ap.permissions.includes(perm.key)}
                                                                            onChange={(checked) => {
                                                                                const current = ap.permissions;
                                                                                setFieldValue(
                                                                                    `album_permissions.${index}.permissions`,
                                                                                    checked
                                                                                        ? [...current, perm.key]
                                                                                        : current.filter(
                                                                                              (k) => k !== perm.key,
                                                                                          ),
                                                                                );
                                                                            }}
                                                                            disabled={isSubmitting}
                                                                        />
                                                                        <Label>
                                                                            {perm.name}{' '}
                                                                            <span className='text-xs text-zinc-500'>
                                                                                ({perm.key})
                                                                            </span>
                                                                        </Label>
                                                                    </CheckboxField>
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
                                    clearFlashes('roles');
                                }}
                                disabled={isSubmitting}
                            >
                                Cancel
                            </Button>
                            <Button type='submit' disabled={isSubmitting || !permissionDefinitions || isLoadingAlbums}>
                                {isSubmitting || !permissionDefinitions || isLoadingAlbums
                                    ? 'Creating...'
                                    : 'Create Role'}
                            </Button>
                        </DialogActions>
                    </Form>
                )}
            </Formik>
        </Dialog>
    );
};

export default CreateRoleForm;
