import React from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useStoreActions, useStoreState } from 'easy-peasy';
import PageContentBlock from '../../elements/PageContentBlock.tsx';
import { Can } from '../../elements/Can.tsx';

const HomeContainer: React.FC = () => {
    const user = useStoreState((state: any) => state.auth.user);
    const effectivePermissions = useStoreState((state: any) => state.auth.currentUserPermissions);
    const logout = useStoreActions((actions: any) => actions.auth.logout);
    const navigate = useNavigate();

    const handleLogout = () => {
        logout();
        navigate('/login');
    };

    if (!user) {
        return <p>Loading user information or not authenticated...</p>;
    }

    return (
        <PageContentBlock title={'Dashboard'}>
            <h1>Admin Portal</h1>
            <p>Welcome, {user.username}!</p>
            <p>Your ID: {user.id}</p>

            <h2>Your Effective Global Permissions:</h2>
            {effectivePermissions && effectivePermissions.length > 0 ? (
                <ul>
                    {effectivePermissions.map((permission: string) => (
                        <li key={permission}>{permission}</li>
                    ))}
                </ul>
            ) : (
                <p>No global permissions effectively assigned.</p>
            )}

            <h2>Your Roles:</h2>
            {user.roles && user.roles.length > 0 ? (
                <ul>
                    {user.roles.map((role: { id: number; name: string }) => (
                        <li key={role.id}>{role.name}</li>
                    ))}
                </ul>
            ) : (
                <p>No roles assigned.</p>
            )}

            <div>
                <h2>Management Links:</h2>
                <ul>
                    <Can permission={['user.list', 'user.view', 'user.create', 'user.edit', 'user.delete']}>
                        <li>
                            <Link to='/admin/users'>Manage Users</Link>
                        </li>
                    </Can>
                    <Can permission={['role.list', 'role.view', 'role.create', 'role.edit', 'role.delete']}>
                        <li>
                            <Link to='/admin/roles'>Manage Roles</Link>
                        </li>
                    </Can>
                    <Can permission={['invite.list', 'invite.view', 'invite.create', 'invite.edit', 'invite.delete']}>
                        <li>
                            <Link to='/admin/invite-codes'>Manage Invite Codes</Link>
                        </li>
                    </Can>
                </ul>
            </div>

            <button onClick={handleLogout}>Logout</button>
        </PageContentBlock>
    );
};

export default HomeContainer;
