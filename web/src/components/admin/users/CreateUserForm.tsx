import React, { useState } from 'react';
import { Button } from '../../elements/Button';
import { Dialog, DialogActions, DialogBody, DialogTitle, DialogDescription } from '../../elements/Dialog';
import { Field, FieldGroup, Label } from '../../elements/Fieldset';
import FormikFieldComponent from '../../elements/FormikField';
import { Formik, Form } from 'formik';
import * as Yup from 'yup';
import { Role, UserCreatePayload } from '../../../types';
import { createUser } from '../../../api/admin/users';
import { useFlash } from '../../../hooks/useFlash';
import { useUsers } from '../../../api/swr/useUsers';
import { useRoles } from '../../../api/swr/useRoles';

const UserCreationSchema = Yup.object().shape({
    username: Yup.string().required('Username is required.'),
    password: Yup.string().required('Password is required.').min(8, 'Password must be at least 8 characters.'),
    first_name: Yup.string().required('First name is required.'),
    last_name: Yup.string().required('Last name is required.'),
    role_ids: Yup.array().of(Yup.number()),
});

interface CreateUserFormProps {
    isOpen: boolean;
    onClose: () => void;
}

const CreateUserForm: React.FC<CreateUserFormProps> = ({ isOpen, onClose }) => {
    const [isSubmitting, setIsSubmitting] = useState(false);

    const { addFlash, clearFlashes } = useFlash();
    const { mutate: mutateUsers } = useUsers();
    const { data: roles } = useRoles();

    const initialValues: UserCreatePayload = {
        username: '',
        password: '',
        role_ids: [],
        first_name: '',
        last_name: '',
    };

    if (!isOpen) return null;

    return (
        <Dialog open={isOpen} onClose={onClose}>
            <Formik
                initialValues={initialValues}
                validationSchema={UserCreationSchema}
                onSubmit={async (values, { resetForm }) => {
                    setIsSubmitting(true);
                    clearFlashes('users');

                    try {
                        const newUser = await createUser(values);

                        mutateUsers((currentData) => {
                            if (!currentData) return [newUser];
                            return [newUser, ...currentData];
                        }, false);

                        addFlash({
                            key: 'users',
                            type: 'success',
                            message: 'User created successfully!',
                        });
                        resetForm();
                        onClose();
                    } catch (error: any) {
                        addFlash({
                            key: 'users',
                            type: 'error',
                            message: error.message || 'Failed to create user.',
                        });
                    } finally {
                        setIsSubmitting(false);
                    }
                }}
            >
                {({ values, handleChange }) => (
                    <Form>
                        <DialogTitle>Create New User</DialogTitle>
                        <DialogDescription>Create a new user account and assign roles.</DialogDescription>
                        <DialogBody>
                            <FieldGroup>
                                <FormikFieldComponent
                                    name='first_name'
                                    label='First Name'
                                    type='text'
                                    disabled={isSubmitting}
                                    required
                                />
                                <FormikFieldComponent
                                    name='last_name'
                                    label='Last Name'
                                    type='text'
                                    disabled={isSubmitting}
                                    required
                                />
                                <FormikFieldComponent
                                    name='username'
                                    label='Username'
                                    type='text'
                                    disabled={isSubmitting}
                                    required
                                />
                                <FormikFieldComponent
                                    name='password'
                                    label='Password'
                                    type='password'
                                    disabled={isSubmitting}
                                    required
                                    minLength={8}
                                />
                                <Field>
                                    <Label>Roles</Label>
                                    {!roles && <p>Loading roles...</p>}
                                    <div className='mt-1 grid max-h-60 grid-cols-2 gap-2 overflow-y-auto rounded border p-2'>
                                        {roles?.map((role: Role) => (
                                            <label key={role.id} className='flex items-center space-x-2 text-sm'>
                                                <input
                                                    type='checkbox'
                                                    name='role_ids'
                                                    value={role.id}
                                                    checked={values.role_ids.includes(role.id)}
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
                            <Button
                                plain
                                onClick={() => {
                                    onClose();
                                    clearFlashes('users');
                                }}
                                disabled={isSubmitting}
                            >
                                Cancel
                            </Button>
                            <Button type='submit' disabled={isSubmitting || !roles}>
                                {isSubmitting ? 'Creating...' : 'Create User'}
                            </Button>
                        </DialogActions>
                    </Form>
                )}
            </Formik>
        </Dialog>
    );
};

export default CreateUserForm;
