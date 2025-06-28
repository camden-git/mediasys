import React, { useState, useEffect } from 'react';
import { useStoreActions } from '../../store/hooks';
import { useNavigate } from 'react-router-dom';

export const RegisterForm: React.FC = () => {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [inviteCode, setInviteCode] = useState('');
    const [isRegistering, setIsRegistering] = useState(false);
    const [registrationError, setRegistrationError] = useState<string | null>(null);
    const [registrationSuccess, setRegistrationSuccess] = useState(false);

    const register = useStoreActions((actions: any) => actions.auth.register);

    const navigate = useNavigate();

    useEffect(() => {
        setRegistrationError(null);
        setRegistrationSuccess(false);
    }, []);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setIsRegistering(true);
        setRegistrationError(null);
        setRegistrationSuccess(false);
        try {
            await register({ username, password, invite_code: inviteCode });
            setRegistrationSuccess(true);
        } catch (err: any) {
            setRegistrationError(err.message || 'Registration failed. Please try again.');
        } finally {
            setIsRegistering(false);
        }
    };

    if (registrationSuccess) {
        return (
            <div>
                <p>Registration successful! Please proceed to login.</p>
                <button onClick={() => navigate('/login')}>Go to Login</button>
            </div>
        );
    }

    return (
        <form onSubmit={handleSubmit}>
            <h2>Register</h2>
            {registrationError && <p style={{ color: 'red' }}>{registrationError}</p>}
            <div>
                <label htmlFor='username'>Username:</label>
                <input
                    type='text'
                    id='username'
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    required
                    disabled={isRegistering}
                />
            </div>
            <div>
                <label htmlFor='password'>Password:</label>
                <input
                    type='password'
                    id='password'
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    disabled={isRegistering}
                />
            </div>
            <div>
                <label htmlFor='inviteCode'>Invite Code:</label>
                <input
                    type='text'
                    id='inviteCode'
                    value={inviteCode}
                    onChange={(e) => setInviteCode(e.target.value)}
                    required
                    disabled={isRegistering}
                />
            </div>
            <button type='submit' disabled={isRegistering}>
                {isRegistering ? 'Registering...' : 'Register'}
            </button>
        </form>
    );
};
