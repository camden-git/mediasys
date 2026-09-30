import React from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuthStore } from '../../../store/useAuthStore';
import PageContentBlock from '../../elements/PageContentBlock.tsx';
import { Can } from '../../elements/Can.tsx';
import { Button } from '../../elements/Button';

const linkClass = 'text-blue-600 hover:underline dark:text-blue-400';

const HomeContainer: React.FC = () => {
    const user = useAuthStore((s) => s.user);
    const effectivePermissions = React.useMemo(() => {
        if (!user) return [];
        const perms = new Set<string>();
        user.global_permissions?.forEach((p) => perms.add(p));
        user.roles?.forEach((role) => role.global_permissions?.forEach((p) => perms.add(p)));
        return Array.from(perms);
    }, [user]);
    const logout = useAuthStore((s) => s.logout);
    const navigate = useNavigate();

    const handleLogout = () => {
        logout();
        navigate('/auth/login');
    };

    // the admin router only renders this page for authenticated users
    if (!user) return null;

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
                            <Link to='/admin/users' className={linkClass}>
                                Manage Users
                            </Link>
                        </li>
                    </Can>
                    <Can permission={['role.list', 'role.view', 'role.create', 'role.edit', 'role.delete']}>
                        <li>
                            <Link to='/admin/roles' className={linkClass}>
                                Manage Roles
                            </Link>
                        </li>
                    </Can>
                    <Can permission={['invite.list', 'invite.view', 'invite.create', 'invite.edit', 'invite.delete']}>
                        <li>
                            <Link to='/admin/invite-codes' className={linkClass}>
                                Manage Invite Codes
                            </Link>
                        </li>
                    </Can>
                    <Can permission='album.*' allowAnyAlbumAccess>
                        <li>
                            <Link to='/admin/albums' className={linkClass}>
                                Manage Albums
                            </Link>
                        </li>
                    </Can>
                    <Can permission='album.group.manage'>
                        <li>
                            <Link to='/admin/groups' className={linkClass}>
                                Manage Groups
                            </Link>
                        </li>
                    </Can>
                    <Can permission='collection.manage'>
                        <li>
                            <Link to='/admin/collections' className={linkClass}>
                                Manage Collections
                            </Link>
                        </li>
                    </Can>
                    <Can permission='people.manage'>
                        <li>
                            <Link to='/admin/people' className={linkClass}>
                                Manage People
                            </Link>
                        </li>
                    </Can>
                    <Can permission='face.manage'>
                        <li>
                            <Link to='/admin/faces' className={linkClass}>
                                Face Tagging
                            </Link>
                        </li>
                    </Can>
                </ul>
            </div>

            <Button outline onClick={handleLogout} className='mt-6'>
                Logout
            </Button>
        </PageContentBlock>
    );
};

export default HomeContainer;
