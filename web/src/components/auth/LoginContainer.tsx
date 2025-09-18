import React, { useEffect, useState } from 'react';
import { Turnstile } from '@marsidev/react-turnstile';
import { useStoreActions, useStoreState } from '../../store/hooks';
import { useNavigate } from 'react-router-dom';
import { Heading } from '../elements/Heading.tsx';
import FormikFieldComponent from '../elements/FormikField.tsx';
import { Checkbox, CheckboxField } from '../elements/Checkbox.tsx';
import { Strong, Text, TextLink } from '../elements/Text.tsx';
import { Button } from '../elements/Button.tsx';
import { Logo } from '../elements/Logo.tsx';
import FlashMessageRender from '../elements/FlashMessageRender.tsx';
import { useFlash } from '../../hooks/useFlash';
import { Formik, Form } from 'formik';
import * as Yup from 'yup';

export const LoginContainer: React.FC = () => {
    const [isLoading, setIsLoading] = useState(false);
    const [turnstileToken, setTurnstileToken] = useState<string | null>(null);
    const siteKey = (import.meta as any).env.VITE_TURNSTILE_SITE_KEY as string | undefined;

    const login = useStoreActions((actions: any) => actions.auth.login);
    const isAuthenticated = useStoreState((state: any) => state.auth.isAuthenticated);
    const navigate = useNavigate();
    const { clearFlashes, clearAndAddHttpError } = useFlash();

    const validationSchema = Yup.object().shape({
        username: Yup.string().required('A username or email must be provided.'),
        password: Yup.string().required('Please enter your account password.'),
    });

    useEffect(() => {
        if (isAuthenticated) {
            navigate('/admin');
        }
    }, [isAuthenticated, navigate]);

    // Token is captured via component callbacks

    return (
        <Formik
            initialValues={{ username: '', password: '' }}
            validationSchema={validationSchema}
            onSubmit={async (values, { setSubmitting }) => {
                setIsLoading(true);
                clearFlashes('auth:login');
                try {
                    await login({ ...values, turnstile_token: siteKey ? (turnstileToken ?? undefined) : undefined });
                } catch (err: any) {
                    clearAndAddHttpError({ error: err, key: 'auth:login' });
                } finally {
                    setIsLoading(false);
                    setSubmitting(false);
                }
            }}
        >
            {({ isSubmitting }) => (
                <Form className='grid w-full max-w-sm grid-cols-1 gap-8'>
                    <Logo className='h-6 text-zinc-950 dark:text-white forced-colors:text-[CanvasText]' />
                    <Heading>Sign in to your account</Heading>
                    <FlashMessageRender byKey={'auth:login'} />

                    <FormikFieldComponent
                        name='username'
                        label='Email'
                        type='text'
                        disabled={isLoading || isSubmitting}
                    />
                    <FormikFieldComponent
                        name='password'
                        label='Password'
                        type='password'
                        disabled={isLoading || isSubmitting}
                    />

                    {siteKey ? (
                        <div className='flex justify-center'>
                            <Turnstile
                                siteKey={siteKey}
                                className='w-full'
                                options={{ theme: 'light' }}
                                onSuccess={(token: string) => setTurnstileToken(token)}
                                onExpire={() => setTurnstileToken(null)}
                                onError={() => setTurnstileToken(null)}
                            />
                        </div>
                    ) : null}

                    <div className='flex items-center justify-between'>
                        <CheckboxField>
                            <Checkbox name='remember' />
                            <span className='text-base/6 text-zinc-950 select-none sm:text-sm/6 dark:text-white'>
                                Remember me
                            </span>
                        </CheckboxField>
                        <Text>
                            <TextLink to='#'>
                                <Strong>Forgot password?</Strong>
                            </TextLink>
                        </Text>
                    </div>
                    <Button
                        type='submit'
                        disabled={isLoading || isSubmitting || (!!siteKey && !turnstileToken)}
                        className='w-full'
                    >
                        {isLoading || isSubmitting ? 'Logging in...' : 'Login'}
                    </Button>
                    <Text>
                        Don’t have an account?{' '}
                        <TextLink to='/auth/register'>
                            <Strong>Sign up</Strong>
                        </TextLink>
                    </Text>
                </Form>
            )}
        </Formik>
    );
};
