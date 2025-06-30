import React from 'react';
import { Routes, Route } from 'react-router-dom';
import ProtectedRoute from '../components/router/ProtectedRoute';
import { StackedLayout } from '../components/elements/StackedLayout.tsx';
import { Navbar, NavbarItem, NavbarSection, NavbarSpacer } from '../components/elements/Navbar.tsx';
import { InboxIcon, MagnifyingGlassIcon } from '@heroicons/react/20/solid';
import { Sidebar, SidebarBody, SidebarItem, SidebarSection } from '../components/elements/Sidebar.tsx';
import InviteCodeManagementContainer from '../components/admin/invites/InviteCodeManagementContainer.tsx';
import RoleManagementContainer from '../components/admin/roles/RoleManagementContainer.tsx';
import RoleView from '../components/admin/roles/RoleView.tsx';
import HomeContainer from '../components/admin/home/HomeContainer.tsx';
import UsersContainer from '../components/admin/users/UsersContainer.tsx';
import { Can } from '../components/elements/Can.tsx';
import UserView from '../components/admin/users/UserView.tsx';
import AlbumManagementContainer from '../components/admin/albums/AlbumManagementContainer.tsx';
import CreateAlbumForm from '../components/admin/albums/CreateAlbumForm.tsx';

export interface RouteDefinition {
    path: string;
    // If undefined is passed, this route is still rendered into the router itself,
    // but no navigation link is displayed in the sub-navigation menu.
    name: string | undefined;
    component: React.ComponentType;
    exact?: boolean;
}

export interface AdminRouteDefinition extends RouteDefinition {
    permission: string | string[] | null;
}

const navItems: AdminRouteDefinition[] = [
    {
        path: '/',
        permission: null,
        name: 'Home',
        component: HomeContainer,
        exact: true,
    },
    {
        path: 'invite-codes',
        permission: 'invite.*',
        name: 'Invite Codes',
        component: InviteCodeManagementContainer,
    },
    {
        path: 'roles',
        permission: 'role.*',
        name: 'Roles',
        component: RoleManagementContainer,
    },
    {
        path: 'roles/:id',
        permission: 'role.view',
        name: undefined,
        component: RoleView,
    },
    {
        path: 'users',
        permission: 'user.*',
        name: 'Users',
        component: UsersContainer,
    },
    {
        path: 'users/:id',
        permission: 'user.view',
        name: undefined,
        component: UserView,
    },
    {
        path: 'albums',
        permission: 'album.*',
        name: 'Albums',
        component: AlbumManagementContainer,
    },
    {
        path: 'albums/create',
        permission: 'album.create',
        name: undefined,
        component: CreateAlbumForm,
    },
];

const AdminRouter: React.FC = () => {
    return (
        <StackedLayout
            navbar={
                <Navbar>
                    <NavbarSection className='max-lg:hidden'>
                        {navItems
                            .filter((route) => !!route.name)
                            .map(({ path, permission, name, exact }) => (
                                <Can permission={permission} key={path}>
                                    <NavbarItem to={`/admin/${path}`.replace(/\/$/, '')} end={exact}>
                                        {name}
                                    </NavbarItem>
                                </Can>
                            ))}
                    </NavbarSection>
                    <NavbarSpacer />
                    <NavbarSection>
                        <NavbarItem to='/search' aria-label='Search'>
                            <MagnifyingGlassIcon />
                        </NavbarItem>
                        <NavbarItem to='/inbox' aria-label='Inbox'>
                            <InboxIcon />
                        </NavbarItem>
                    </NavbarSection>
                </Navbar>
            }
            sidebar={
                <Sidebar>
                    <SidebarBody>
                        <SidebarSection>
                            {navItems
                                .filter((route) => !!route.name)
                                .map(({ path, permission, name, exact }) => (
                                    <Can permission={permission}>
                                        <SidebarItem key={path} to={`/admin/${path}`.replace(/\/$/, '')} end={exact}>
                                            {name}
                                        </SidebarItem>
                                    </Can>
                                ))}
                        </SidebarSection>
                    </SidebarBody>
                </Sidebar>
            }
        >
            <Routes>
                <Route element={<ProtectedRoute />}>
                    {navItems.map(({ path, permission, component: Component }) => (
                        <Route
                            path={path.replace(/\/$/, '')}
                            key={path}
                            element={
                                <>
                                    <Can permission={permission}>
                                        <Component />
                                    </Can>
                                </>
                            }
                        />
                    ))}

                    {/* <Route path="users" element={<UserManagementPage />} /> */}
                    {/* <Route path="roles" element={<RoleManagementPage />} /> */}
                </Route>
            </Routes>
        </StackedLayout>
    );
};

export default AdminRouter;
