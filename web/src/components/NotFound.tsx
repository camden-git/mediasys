import React from 'react';
import { Link } from 'react-router-dom';
import { useDocumentTitle } from '../hooks/useDocumentTitle.ts';

const NotFound: React.FC = () => {
    useDocumentTitle('Page not found');

    return (
        <div className='mx-auto max-w-xl px-4 py-24 text-center'>
            <p className='text-sm font-semibold text-zinc-500 dark:text-zinc-400'>404</p>
            <h1 className='mt-2 text-3xl font-bold text-zinc-950 dark:text-white'>Page not found</h1>
            <p className='mt-2 text-zinc-500 dark:text-zinc-400'>The page you are looking for does not exist.</p>
            <Link to='/' className='mt-6 inline-block text-sm font-semibold text-zinc-950 underline dark:text-white'>
                Back to home
            </Link>
        </div>
    );
};

export default NotFound;
