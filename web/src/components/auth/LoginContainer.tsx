import React, { useEffect, useRef, useState } from 'react';
import { Turnstile, type TurnstileInstance } from '@marsidev/react-turnstile';
import { useAuthStore } from '../../store/useAuthStore';
import { useLocation, useNavigate } from 'react-router-dom';
import { Heading } from '../elements/Heading.tsx';
import FormikFieldComponent from '../elements/FormikField.tsx';
import { Strong, Text, TextLink } from '../elements/Text.tsx';
import { Button } from '../elements/Button.tsx';
import { Logo } from '../elements/Logo.tsx';
import FlashMessageRender from '../elements/FlashMessageRender.tsx';
import { useFlash } from '../../hooks/useFlash';
import { Formik, Form } from 'formik';
import * as Yup from 'yup';
import { safeRedirectPath, withRateLimitMessage } from '../../lib/helpers';
import { useDocumentTitle } from '../../hooks/useDocumentTitle';

export const LoginContainer: React.FC = () => {
    const [isLoading, setIsLoading] = useState(false);
    const [turnstileToken, setTurnstileToken] = useState<string | null>(null);
    const turnstileRef = useRef<TurnstileInstance>(null);
    const siteKey = (import.meta as any).env.VITE_TURNSTILE_SITE_KEY as string | undefined;

    const login = useAuthStore((s) => s.login);
    const isAuthenticated = useAuthStore((s) => s.isAuthenticated());
    const navigate = useNavigate();
    const location = useLocation();
    const redirectTo = safeRedirectPath((location.state as { from?: unknown } | null)?.from, '/admin');
    const { clearFlashes, clearAndAddHttpError } = useFlash();

    const validationSchema = Yup.object().shape({
        username: Yup.string().required('A username or email must be provided.'),
        password: Yup.string().required('Please enter your account password.'),
    });

    useEffect(() => {
        if (isAuthenticated) {
            navigate(redirectTo, { replace: true });
        }
    }, [isAuthenticated, navigate, redirectTo]);

    useDocumentTitle('Sign in');

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
                    clearAndAddHttpError({ error: withRateLimitMessage(err), key: 'auth:login' });
                    // Turnstile tokens are single-use; get a fresh one for the retry
                    setTurnstileToken(null);
                    turnstileRef.current?.reset();
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
                        label='Username or email'
                        type='text'
                        autoComplete='username'
                        disabled={isLoading || isSubmitting}
                    />
                    <FormikFieldComponent
                        name='password'
                        label='Password'
                        type='password'
                        autoComplete='current-password'
                        disabled={isLoading || isSubmitting}
                    />

                    {siteKey ? (
                        <Turnstile
                            ref={turnstileRef}
                            siteKey={siteKey}
                            className='w-full'
                            options={{ theme: 'light', size: 'flexible' }}
                            onSuccess={(token: string) => setTurnstileToken(token)}
                            onExpire={() => setTurnstileToken(null)}
                            onError={() => setTurnstileToken(null)}
                        />
                    ) : null}

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
