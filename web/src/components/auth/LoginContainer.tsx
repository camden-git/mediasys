import React, { useState } from 'react';
import { useStoreActions, useStoreState } from '../../store/hooks';
import { useNavigate } from 'react-router-dom';
import { Heading } from '../elements/Heading.tsx';
import { Field, Label } from '../elements/Fieldset.tsx';
import { Input } from '../elements/Input.tsx';
import { Checkbox, CheckboxField } from '../elements/Checkbox.tsx';
import { Strong, Text, TextLink } from '../elements/Text.tsx';
import { Button } from '../elements/Button.tsx';
import { Logo } from '../elements/Logo.tsx';

export const LoginContainer: React.FC = () => {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const login = useStoreActions((actions: any) => actions.auth.login);
    const isAuthenticated = useStoreState((state: any) => state.auth.isAuthenticated);
    const navigate = useNavigate();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setIsLoading(true);
        setError(null);
        try {
            await login({ username, password });
        } catch (err: any) {
            setError(err.message || 'Login failed. Please try again.');
        } finally {
            setIsLoading(false);
        }
    };

    React.useEffect(() => {
        if (isAuthenticated) {
            navigate('/admin');
        }
    }, [isAuthenticated, navigate]);

    return (
        <form onSubmit={handleSubmit} className='grid w-full max-w-sm grid-cols-1 gap-8'>
            <Logo className='h-6 text-zinc-950 dark:text-white forced-colors:text-[CanvasText]' />
            <Heading>Sign in to your account</Heading>
            {error && <p style={{ color: 'red' }}>{error}</p>}

            <Field>
                <Label>Email</Label>
                <Input
                    type='text'
                    id='username'
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    required
                />
            </Field>
            <Field>
                <Label>Password</Label>
                <Input
                    type='password'
                    id='password'
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                />
            </Field>
            <div className='flex items-center justify-between'>
                <CheckboxField>
                    <Checkbox name='remember' />
                    <Label>Remember me</Label>
                </CheckboxField>
                <Text>
                    <TextLink to='#'>
                        <Strong>Forgot password?</Strong>
                    </TextLink>
                </Text>
            </div>
            <Button type='submit' disabled={isLoading} className='w-full'>
                {isLoading ? 'Logging in...' : 'Login'}
            </Button>
            <Text>
                Don’t have an account?{' '}
                <TextLink to='/auth/register'>
                    <Strong>Sign up</Strong>
                </TextLink>
            </Text>
        </form>
    );
};
