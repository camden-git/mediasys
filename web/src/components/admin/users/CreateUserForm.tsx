import React, { useState } from 'react';
import { Button } from '../../elements/Button';
import { Dialog, DialogActions, DialogBody, DialogTitle, DialogDescription } from '../../elements/Dialog';
import { FieldGroup } from '../../elements/Fieldset';
import FormikFieldComponent from '../../elements/FormikField';
import { Formik, Form } from 'formik';
import * as Yup from 'yup';
import { UserCreatePayload } from '../../../types';
import { createUser } from '../../../api/admin/users';
import { useFlash } from '../../../hooks/useFlash';
import { Can } from '../../elements/Can';
import RolePicker from './RolePicker';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../lib/queryKeys';

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

    const queryClient = useQueryClient();
    const { addFlash, clearFlashes } = useFlash();

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
                        await createUser(values);
                        queryClient.invalidateQueries({ queryKey: queryKeys.users.all() });

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
                {({ values, setFieldValue }) => (
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
                                {/* /admin/roles needs role.* permissions, so the picker is hidden without them */}
                                <Can permission='role.*'>
                                    <RolePicker
                                        selected={values.role_ids}
                                        disabled={isSubmitting}
                                        onChange={(ids) => setFieldValue('role_ids', ids)}
                                    />
                                </Can>
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
                            <Button type='submit' disabled={isSubmitting}>
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
