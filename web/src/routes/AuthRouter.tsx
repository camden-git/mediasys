import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { LoginContainer } from '../components/auth/LoginContainer.tsx';
import { RegisterForm } from '../components/auth/RegisterForm.tsx';

const AuthRouter: React.FC = () => {
    return (
        <main className='flex min-h-dvh flex-col p-2'>
            <div className='flex grow items-center justify-center p-6 lg:rounded-lg lg:bg-white lg:p-10 lg:shadow-xs lg:ring-1 lg:ring-zinc-950/5 dark:lg:bg-zinc-900 dark:lg:ring-white/10'>
                <Routes>
                    <Route path='login' element={<LoginContainer />} />
                    <Route path='register' element={<RegisterForm />} />
                </Routes>
            </div>
        </main>
    );
};

export default AuthRouter;
