import React from 'react';
import ContentBlock from '../../elements/ContentBlock';
import { Heading } from '../../elements/Heading';
import UserList from './UserList';

const UsersContainer: React.FC = () => {
    return (
        <ContentBlock>
            <Heading level={1}>User Management</Heading>
            <p className='mt-2 text-sm text-gray-600 dark:text-zinc-400'>
                Manage user accounts, roles, and permissions.
            </p>
            <div className='mt-6'>
                <UserList />
            </div>
        </ContentBlock>
    );
};

export default UsersContainer;
