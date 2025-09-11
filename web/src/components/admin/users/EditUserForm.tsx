import React, { useEffect, useState } from 'react';
import { Button } from '../../elements/Button';
import { Dialog, DialogActions, DialogBody, DialogTitle, DialogDescription } from '../../elements/Dialog';
import { Field, FieldGroup, Label, ErrorMessage as FieldErrorMessage } from '../../elements/Fieldset';
import { Input } from '../../elements/Input';
import { Formik, Form, ErrorMessage } from 'formik';
import * as Yup from 'yup';
import { AdminUserResponse, Role, UserUpdatePayload } from '../../../types';
import { updateUserMutation } from '../../../api/swr/useUsers';
import { useRoles } from '../../../api/swr/useRoles';
import { useFlash } from '../../../hooks/useFlash';

const UserUpdateSchema = Yup.object().shape({
    username: Yup.string().required('Username is required.'),
    password: Yup.string().min(8, 'Password must be at least 8 characters if provided.'),
    first_name: Yup.string().optional(),
    last_name: Yup.string().optional(),
    role_ids: Yup.array().of(Yup.number()),
});

interface EditUserFormProps {
    isOpen: boolean;
    onClose: () => void;
    user: AdminUserResponse;
}

const EditUserForm: React.FC<EditUserFormProps> = ({ isOpen, onClose, user }) => {
    const [isSubmitting, setIsSubmitting] = useState(false);
    const { data: roles, isLoading: isLoadingRoles } = useRoles();
    const { clearFlashes, clearAndAddHttpError, addFlash } = useFlash();

    const [formMessage, setFormMessage] = useState<string | null>(null);

    useEffect(() => {
        if (!isOpen) {
            clearFlashes('edit-user-form');
        }
    }, [isOpen, clearFlashes]);

    const initialValues: UserUpdatePayload = {
        username: user.username,
        role_ids: user.roles?.map((r) => r.id) || [],
        first_name: user.first_name,
        last_name: user.last_name,
    };

    if (!isOpen) return null;

    return (
        <Dialog open={isOpen} onClose={onClose}>
            <Formik
                initialValues={initialValues}
                validationSchema={UserUpdateSchema}
                enableReinitialize
                onSubmit={async (values, { setSubmitting }) => {
                    setFormMessage(null);
                    clearFlashes('edit-user-form');
                    setIsSubmitting(true);

                    try {
                        await updateUserMutation(user.id, values);

                        addFlash({
                            key: 'edit-user-form',
                            type: 'success',
                            message: 'User updated successfully!',
                        });

                        onClose();
                    } catch (err: any) {
                        clearAndAddHttpError({
                            error: err,
                            key: 'edit-user-form',
                        });
                        setFormMessage(err.message || 'Failed to update user.');
                    } finally {
                        setIsSubmitting(false);
                        setSubmitting(false);
                    }
                }}
            >
                {({ values, handleChange, handleBlur }) => (
                    <Form>
                        <DialogTitle>Edit User: {user.first_name} {user.last_name} ({user.username})</DialogTitle>
                        <DialogDescription>Update the user's details and assigned roles.</DialogDescription>
                        <DialogBody>
                            {formMessage && <p style={{ color: 'red', marginBottom: '1rem' }}>{formMessage}</p>}
                            <FieldGroup>
                                <Field>
                                    <Label htmlFor='first_name'>First Name</Label>
                                    <Input
                                        id='first_name'
                                        name='first_name'
                                        type='text'
                                        value={values.first_name}
                                        onChange={handleChange}
                                        onBlur={handleBlur}
                                        disabled={isSubmitting}
                                    />
                                    <ErrorMessage name='first_name' component={FieldErrorMessage} />
                                </Field>
                                <Field>
                                    <Label htmlFor='last_name'>Last Name</Label>
                                    <Input
                                        id='last_name'
                                        name='last_name'
                                        type='text'
                                        value={values.last_name}
                                        onChange={handleChange}
                                        onBlur={handleBlur}
                                        disabled={isSubmitting}
                                    />
                                    <ErrorMessage name='last_name' component={FieldErrorMessage} />
                                </Field>
                                <Field>
                                    <Label htmlFor='username'>Username</Label>
                                    <Input
                                        id='username'
                                        name='username'
                                        type='text'
                                        value={values.username}
                                        onChange={handleChange}
                                        onBlur={handleBlur}
                                        disabled={isSubmitting}
                                    />
                                    <ErrorMessage name='username' component={FieldErrorMessage} />
                                </Field>
                                <Field>
                                    <Label htmlFor='password'>Change Password</Label>
                                    <Input
                                        id='password'
                                        name='password'
                                        type='password'
                                        placeholder='Leave blank to keep current password'
                                        onChange={handleChange}
                                        onBlur={handleBlur}
                                        disabled={isSubmitting}
                                    />
                                    <ErrorMessage name='password' component={FieldErrorMessage} />
                                </Field>
                                <Field>
                                    <Label>Roles</Label>
                                    {isLoadingRoles && <p>Loading roles...</p>}
                                    <div className='mt-1 grid max-h-60 grid-cols-2 gap-2 overflow-y-auto rounded border p-2'>
                                        {roles?.map((role: Role) => (
                                            <label key={role.id} className='flex items-center space-x-2 text-sm'>
                                                <input
                                                    type='checkbox'
                                                    name='role_ids'
                                                    value={role.id}
                                                    checked={values.role_ids?.includes(role.id)}
                                                    onChange={handleChange}
                                                    disabled={isSubmitting}
                                                    className='rounded'
                                                />
                                                <span>{role.name}</span>
                                            </label>
                                        ))}
                                    </div>
                                </Field>
                            </FieldGroup>
                        </DialogBody>
                        <DialogActions>
                            <Button plain onClick={onClose} disabled={isSubmitting}>
                                Cancel
                            </Button>
                            <Button type='submit' disabled={isSubmitting || isLoadingRoles}>
                                {isSubmitting ? 'Saving...' : 'Save Changes'}
                            </Button>
                        </DialogActions>
                    </Form>
                )}
            </Formik>
        </Dialog>
    );
};

export default EditUserForm;
