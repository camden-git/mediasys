import React from 'react';
import ContentBlock from '../../elements/PageContentBlock.tsx';
import UserList from './UserList';

const UsersContainer: React.FC = () => {
    return (
        <ContentBlock>
            <UserList />
        </ContentBlock>
    );
};

export default UsersContainer;
