import React, { useState, useEffect } from 'react';
import { useStoreActions } from '../../store/hooks';
import { useNavigate } from 'react-router-dom';
import { Heading } from '../elements/Heading.tsx';
import FormikFieldComponent from '../elements/FormikField.tsx';
import { Strong, Text, TextLink } from '../elements/Text.tsx';
import { Button } from '../elements/Button.tsx';
import { Logo } from '../elements/Logo.tsx';
import FlashMessageRender from '../elements/FlashMessageRender.tsx';
import { useFlash } from '../../hooks/useFlash';
import { Formik, Form } from 'formik';
import * as Yup from 'yup';

export const RegisterForm: React.FC = () => {
    const [isRegistering, setIsRegistering] = useState(false);
    const [registrationSuccess, setRegistrationSuccess] = useState(false);

    const register = useStoreActions((actions: any) => actions.auth.register);
    const navigate = useNavigate();
    const { clearFlashes, addFlash, clearAndAddHttpError } = useFlash();

    useEffect(() => {
        clearFlashes('auth:register');
        setRegistrationSuccess(false);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const validationSchema = Yup.object().shape({
        first_name: Yup.string().required('First name is required.'),
        last_name: Yup.string().required('Last name is required.'),
        username: Yup.string().required('Email is required.'),
        password: Yup.string().required('Password is required.'),
        invite_code: Yup.string()
            .matches(/^\d{6}$/, 'Invite code must be a 6-digit PIN.')
            .required('Invite code is required.'),
    });

    if (registrationSuccess) {
        return (
            <div className='grid w-full max-w-sm grid-cols-1 gap-8'>
                <Logo className='h-6 text-zinc-950 dark:text-white forced-colors:text-[CanvasText]' />
                <Heading>Registration successful</Heading>
                <FlashMessageRender byKey={'auth:register'} />
                <Button type='button' className='w-full' onClick={() => navigate('/auth/login')}>
                    Go to Login
                </Button>
            </div>
        );
    }

    return (
        <Formik
            initialValues={{ first_name: '', last_name: '', username: '', password: '', invite_code: '' }}
            validationSchema={validationSchema}
            onSubmit={async (values, { setSubmitting }) => {
                setIsRegistering(true);
                clearFlashes('auth:register');
                try {
                    await register(values as any);
                    addFlash({
                        key: 'auth:register',
                        type: 'success',
                        title: 'Registration successful',
                        message: 'Your account has been created. You can now sign in.',
                    });
                    setRegistrationSuccess(true);
                } catch (err: any) {
                    clearAndAddHttpError({ error: err, key: 'auth:register' });
                } finally {
                    setIsRegistering(false);
                    setSubmitting(false);
                }
            }}
        >
            {({ isSubmitting }) => (
                <Form className='grid w-full max-w-sm grid-cols-1 gap-8'>
                    <Logo className='h-6 text-zinc-950 dark:text-white forced-colors:text-[CanvasText]' />
                    <Heading>Create your account</Heading>
                    <FlashMessageRender byKey={'auth:register'} />

                    <FormikFieldComponent
                        name='first_name'
                        label='First name'
                        type='text'
                        disabled={isRegistering || isSubmitting}
                    />
                    <FormikFieldComponent
                        name='last_name'
                        label='Last name'
                        type='text'
                        disabled={isRegistering || isSubmitting}
                    />
                    <FormikFieldComponent
                        name='username'
                        label='Email'
                        type='text'
                        disabled={isRegistering || isSubmitting}
                    />
                    <FormikFieldComponent
                        name='password'
                        label='Password'
                        type='password'
                        disabled={isRegistering || isSubmitting}
                    />
                    <FormikFieldComponent
                        name='invite_code'
                        label='Invite code'
                        type='text'
                        inputMode='numeric'
                        pattern='\\d{6}'
                        maxLength={6}
                        placeholder='6-digit PIN'
                        disabled={isRegistering || isSubmitting}
                    />

                    <Button type='submit' disabled={isRegistering || isSubmitting} className='w-full'>
                        {isRegistering || isSubmitting ? 'Registering...' : 'Register'}
                    </Button>

                    <Text>
                        Already have an account?{' '}
                        <TextLink to='/auth/login'>
                            <Strong>Sign in</Strong>
                        </TextLink>
                    </Text>
                </Form>
            )}
        </Formik>
    );
};
