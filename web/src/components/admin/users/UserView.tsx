import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { DescriptionList, DescriptionTerm, DescriptionDetails } from '../../elements/DescriptionList';
import { Heading } from '../../elements/Heading';
import ContentBlock from '../../elements/PageContentBlock.tsx';
import { Button } from '../../elements/Button';
import { Can } from '../../elements/Can';
import EditUserForm from './EditUserForm';
import { useUser } from '../../../api/swr/useUsers';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import LoadingSpinner from '../../elements/LoadingSpinner';

const UserView: React.FC = () => {
    const { id } = useParams<{ id: string }>();
    const userId = id ? parseInt(id, 10) : 0;

    const { data: user, error, isValidating } = useUser(userId);
    const { clearFlashes, clearAndAddHttpError } = useFlash();
    const [isEditModalOpen, setEditModalOpen] = useState(false);

    useEffect(() => {
        if (!error) {
            clearFlashes('user-view');
            return;
        }

        clearAndAddHttpError({ error, key: 'user-view' });
    }, [error, clearFlashes, clearAndAddHttpError]);

    if (!user || (error && isValidating)) {
        return <LoadingSpinner />;
    }

    if (error) {
        return <p style={{ color: 'red' }}>Error: {error.message}</p>;
    }

    return (
        <>
            <FlashMessageRender byKey={'user-view'} className={'mb-4'} />

            <ContentBlock>
                <div className='flex items-center justify-between'>
                    <Heading level={2}>User Details</Heading>
                    <Can permission='user.edit'>
                        <Button onClick={() => setEditModalOpen(true)}>Edit User</Button>
                    </Can>
                </div>
                <DescriptionList className='mt-4'>
                    <DescriptionTerm>ID</DescriptionTerm>
                    <DescriptionDetails>{user.id}</DescriptionDetails>

                    <DescriptionTerm>Name</DescriptionTerm>
                    <DescriptionDetails>
                        {user.first_name} {user.last_name}
                    </DescriptionDetails>

                    <DescriptionTerm>Username</DescriptionTerm>
                    <DescriptionDetails>{user.username}</DescriptionDetails>

                    <DescriptionTerm>Roles</DescriptionTerm>
                    <DescriptionDetails>
                        {user.roles && user.roles.length > 0 ? (
                            <ul className='list-inside list-disc'>
                                {user.roles.map((r) => (
                                    <li key={r.id}>{r.name}</li>
                                ))}
                            </ul>
                        ) : (
                            'None'
                        )}
                    </DescriptionDetails>

                    <DescriptionTerm>Global Permissions</DescriptionTerm>
                    <DescriptionDetails>
                        {user.global_permissions && user.global_permissions.length > 0 ? (
                            <ul className='list-inside list-disc'>
                                {user.global_permissions.map((p) => (
                                    <li key={p}>{p}</li>
                                ))}
                            </ul>
                        ) : (
                            'None'
                        )}
                    </DescriptionDetails>
                </DescriptionList>
            </ContentBlock>

            <EditUserForm isOpen={isEditModalOpen} onClose={() => setEditModalOpen(false)} user={user} />
        </>
    );
};

export default UserView;
