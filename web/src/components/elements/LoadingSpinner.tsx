import React from 'react';

const LoadingSpinner: React.FC = () => {
    return (
        <div className='flex items-center justify-center p-4'>
            <div className='h-8 w-8 animate-spin rounded-full border-b-2 border-blue-500'></div>
            <span className='ml-3 text-gray-600'>Loading...</span>
        </div>
    );
};

export default LoadingSpinner;
