import React from 'react';
import { Formik, Form } from 'formik';
import * as Yup from 'yup';
import { useAuthStore } from '../../../store/useAuthStore';
import { useUIStore } from '../../../store/useUIStore';
import { updateProfile } from '../../../api/auth';
import { FieldGroup } from '../../elements/Fieldset';
import HeaderedContent from '../../elements/HeaderedContent';
import FormikFieldComponent from '../../elements/FormikField';
import FlashMessageRender from '../../elements/FlashMessageRender';
import { UnsavedChangesBar } from '../../elements/UnsavedChangesBar';
import { AuthenticatedUser } from '../../../store/useAuthStore';

const profileInfoSchema = Yup.object({
    first_name: Yup.string().required('First name is required'),
    last_name: Yup.string().required('Last name is required'),
    username: Yup.string().required('Username is required'),
});

const changePasswordSchema = Yup.object({
    current_password: Yup.string().required('Current password is required'),
    new_password: Yup.string().min(8, 'Password must be at least 8 characters').required('New password is required'),
    confirm_new_password: Yup.string()
        .oneOf([Yup.ref('new_password')], 'Passwords must match')
        .required('Please confirm your new password'),
});

const ProfileContainer: React.FC = () => {
    const user = useAuthStore((s) => s.user);
    const setUser = useAuthStore((s) => s.setUser);
    const addFlash = useUIStore((s) => s.addFlash);

    if (!user) return null;

    return (
        <div className='space-y-16'>
            <FlashMessageRender byKey='profile-info' />
            <HeaderedContent title='Profile Information' description='Update your name and username.' className='pb-8'>
                <Formik
                    initialValues={{
                        first_name: user.first_name,
                        last_name: user.last_name,
                        username: user.username,
                    }}
                    validationSchema={profileInfoSchema}
                    enableReinitialize
                    onSubmit={async (values, { setSubmitting }) => {
                        try {
                            const updated = await updateProfile(values);
                            setUser(updated as AuthenticatedUser);
                            addFlash({
                                key: 'profile-info',
                                type: 'success',
                                title: 'Saved',
                                message: 'Profile updated successfully',
                            });
                        } catch (err: any) {
                            addFlash({
                                key: 'profile-info',
                                type: 'error',
                                title: 'Error',
                                message: err.message || 'Failed to update profile',
                            });
                        } finally {
                            setSubmitting(false);
                        }
                    }}
                >
                    {({ isSubmitting, dirty, resetForm }) => (
                        <Form className='space-y-6'>
                            <FieldGroup>
                                <FormikFieldComponent name='first_name' label='First Name' required />
                                <FormikFieldComponent name='last_name' label='Last Name' required />
                                <FormikFieldComponent name='username' label='Username' required />
                            </FieldGroup>
                            <UnsavedChangesBar isDirty={dirty} isSubmitting={isSubmitting} onDiscard={resetForm} />
                        </Form>
                    )}
                </Formik>
            </HeaderedContent>

            <FlashMessageRender byKey='profile-password' />
            <HeaderedContent title='Change Password' description='Update your account password.' className='pb-8'>
                <Formik
                    initialValues={{
                        current_password: '',
                        new_password: '',
                        confirm_new_password: '',
                    }}
                    validationSchema={changePasswordSchema}
                    onSubmit={async (values, { setSubmitting, resetForm }) => {
                        try {
                            await updateProfile({
                                current_password: values.current_password,
                                new_password: values.new_password,
                            });
                            addFlash({
                                key: 'profile-password',
                                type: 'success',
                                title: 'Saved',
                                message: 'Password changed successfully',
                            });
                            resetForm();
                        } catch (err: any) {
                            addFlash({
                                key: 'profile-password',
                                type: 'error',
                                title: 'Error',
                                message: err.message || 'Failed to change password',
                            });
                        } finally {
                            setSubmitting(false);
                        }
                    }}
                >
                    {({ isSubmitting, dirty, resetForm }) => (
                        <Form className='space-y-6'>
                            <FieldGroup>
                                <FormikFieldComponent
                                    name='current_password'
                                    label='Current Password'
                                    type='password'
                                    required
                                />
                                <FormikFieldComponent
                                    name='new_password'
                                    label='New Password'
                                    type='password'
                                    required
                                />
                                <FormikFieldComponent
                                    name='confirm_new_password'
                                    label='Confirm New Password'
                                    type='password'
                                    required
                                />
                            </FieldGroup>
                            <UnsavedChangesBar isDirty={dirty} isSubmitting={isSubmitting} onDiscard={resetForm} />
                        </Form>
                    )}
                </Formik>
            </HeaderedContent>
        </div>
    );
};

export default ProfileContainer;
