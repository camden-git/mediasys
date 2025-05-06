import React from 'react';

interface ErrorMessageProps {
    message: string | null;
}

const ErrorMessage: React.FC<ErrorMessageProps> = ({ message }) => {
    if (!message) return null;
    return (
        <div className='relative my-4 rounded border border-red-400 bg-red-100 px-4 py-3 text-red-700' role='alert'>
            <strong className='font-bold'>Error: </strong>
            <span className='block sm:inline'>{message}</span>
        </div>
    );
};

export default ErrorMessage;
