import React from 'react';
import { Routes, Route } from 'react-router-dom';
import ProtectedRoute from '../components/router/ProtectedRoute';
import { StackedLayout } from '../components/elements/StackedLayout.tsx';
import { Navbar, NavbarItem, NavbarSection } from '../components/elements/Navbar.tsx';
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
import PeopleManagementContainer from '../components/admin/people/PeopleManagementContainer.tsx';
import PersonAdminView from '../components/admin/people/PersonAdminView.tsx';
import FaceTaggingContainer from '../components/admin/faces/FaceTaggingContainer.tsx';
import GroupManagementContainer from '../components/admin/groups/GroupManagementContainer.tsx';
import CollectionManagementContainer from '../components/admin/collections/CollectionManagementContainer.tsx';
import ProfileContainer from '../components/admin/profile/ProfileContainer.tsx';

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
    // when true, users who only hold per-album permissions (no matching global permission)
    // can still see/access this route, since it leads to an album list filtered to what
    // they're allowed to see rather than requiring blanket global access.
    allowAnyAlbumAccess?: boolean;
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
        allowAnyAlbumAccess: true,
    },
    {
        path: 'albums/create',
        permission: 'album.create',
        name: undefined,
        component: CreateAlbumForm,
    },
    {
        path: 'people',
        permission: null,
        name: 'People',
        component: PeopleManagementContainer,
    },
    {
        path: 'people/:id',
        permission: null,
        name: undefined,
        component: PersonAdminView,
    },
    {
        path: 'faces',
        permission: null,
        name: 'Face Tagging',
        component: FaceTaggingContainer,
    },
    {
        path: 'groups',
        permission: null,
        name: 'Groups',
        component: GroupManagementContainer,
    },
    {
        path: 'collections',
        permission: 'collection.manage',
        name: 'Collections',
        component: CollectionManagementContainer,
    },
    {
        path: 'profile',
        permission: null,
        name: 'Account',
        component: ProfileContainer,
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
                            .map(({ path, permission, name, exact, allowAnyAlbumAccess }) => (
                                <Can permission={permission} allowAnyAlbumAccess={allowAnyAlbumAccess} key={path}>
                                    <NavbarItem to={`/admin/${path}`.replace(/\/$/, '')} end={exact}>
                                        {name}
                                    </NavbarItem>
                                </Can>
                            ))}
                    </NavbarSection>
                </Navbar>
            }
            sidebar={
                <Sidebar>
                    <SidebarBody>
                        <SidebarSection>
                            {navItems
                                .filter((route) => !!route.name)
                                .map(({ path, permission, name, exact, allowAnyAlbumAccess }) => (
                                    <Can permission={permission} allowAnyAlbumAccess={allowAnyAlbumAccess} key={path}>
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
                    {navItems.map(({ path, permission, component: Component, allowAnyAlbumAccess }) => (
                        <Route
                            path={path.replace(/\/$/, '')}
                            key={path}
                            element={
                                <>
                                    <Can permission={permission} allowAnyAlbumAccess={allowAnyAlbumAccess}>
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
