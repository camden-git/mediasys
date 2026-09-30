import React from 'react';
import { Link } from 'react-router-dom';
import { Heading } from '../../elements/Heading';
import { Text } from '../../elements/Text';

export const NoPermission: React.FC = () => (
    <div className='flex flex-col items-center justify-center rounded-lg border border-dashed border-zinc-300 p-16 text-center'>
        <Heading level={4}>No permission</Heading>
        <Text className='mt-2 text-sm text-zinc-500'>You do not have permission to view this page.</Text>
    </div>
);

export const AdminNotFound: React.FC<{ backTo: string; backLabel: string }> = ({ backTo, backLabel }) => (
    <div className='flex flex-col items-center justify-center rounded-lg border border-dashed border-zinc-300 p-16 text-center'>
        <Heading level={4}>Page not found</Heading>
        <Text className='mt-2 text-sm text-zinc-500'>
            <Link to={backTo} className='underline'>
                {backLabel}
            </Link>
        </Text>
    </div>
);
