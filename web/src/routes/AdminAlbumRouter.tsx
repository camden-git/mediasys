import React from 'react';
import { Routes, Route, useParams } from 'react-router-dom';
import ProtectedRoute from '../components/router/ProtectedRoute';
import { StackedLayout } from '../components/elements/StackedLayout.tsx';
import { Navbar, NavbarItem, NavbarLabel, NavbarSection } from '../components/elements/Navbar.tsx';
import { ArrowLeftIcon, PlusIcon } from '@heroicons/react/20/solid';
import { Sidebar, SidebarBody, SidebarItem, SidebarSection } from '../components/elements/Sidebar.tsx';
import AlbumView from '../components/admin/albums/AlbumView.tsx';
import AlbumSubusersPage from '../components/admin/albums/AlbumSubusersPage.tsx';
import { Can } from '../components/elements/Can.tsx';
import { NoPermission, AdminNotFound } from '../components/admin/shared/StatusMessages.tsx';
import LoadingSpinner from '../components/elements/LoadingSpinner';
import {
    Dropdown,
    DropdownButton,
    DropdownDivider,
    DropdownItem,
    DropdownLabel,
    DropdownMenu,
} from '../components/elements/Dropdown.tsx';
import { ChevronDownIcon, Cog8ToothIcon } from '@heroicons/react/16/solid';
import { useQuery } from '@tanstack/react-query';
import { getAlbumBySlug, listAlbums } from '../api/admin/albums';
import { queryKeys } from '../lib/queryKeys';
import OverviewContainer from '../components/admin/albums/overview/OverviewContainer.tsx';
import { SettingsContainer } from '../components/admin/albums/settings/SettingsContainer.tsx';
import AlbumFaceTaggingContainer from '../components/admin/albums/faces/AlbumFaceTaggingContainer.tsx';

export interface AdminAlbumRouteDefinition {
    path: string;
    name: string;
    component: React.ComponentType;
    permission: string | string[] | null;
}

const AdminAlbumRouter: React.FC = () => {
    const { slug } = useParams<{ slug: string }>();
    const { data: albumsData } = useQuery({
        queryKey: queryKeys.albums.list(),
        queryFn: ({ signal }) => listAlbums(signal),
    });
    const albums = albumsData ?? [];

    const {
        data: album,
        isLoading,
        error,
    } = useQuery({
        queryKey: queryKeys.albums.bySlug(slug!),
        queryFn: ({ signal }) => getAlbumBySlug(slug!, signal),
        enabled: !!slug,
    });
    const albumName = album?.name;

    if (error && !album) {
        return <div className='text-center text-red-600'>Error loading album: {(error as Error).message}</div>;
    }

    if (isLoading || !album) {
        return (
            <div className='flex h-64 items-center justify-center'>
                <LoadingSpinner />
            </div>
        );
    }

    const navItems: AdminAlbumRouteDefinition[] = [
        {
            path: '/',
            name: 'Details',
            component: AlbumView,
            permission: ['album.list', 'album.view.content'],
        },
        {
            path: '/overview',
            name: 'Media',
            component: OverviewContainer,
            permission: ['album.list', 'album.view.content'],
        },
        {
            path: '/settings',
            name: 'Settings',
            component: SettingsContainer,
            permission: ['album.edit.general', 'album.photo.editmeta'],
        },
        {
            path: '/subusers',
            name: 'Subusers',
            component: AlbumSubusersPage,
            permission: ['album.manage.members.global', 'album.manage.members'],
        },
        {
            path: '/faces',
            name: 'Face Tagging',
            component: AlbumFaceTaggingContainer,
            permission: 'face.manage',
        },
    ];

    return (
        <StackedLayout
            navbar={
                <Navbar>
                    <Dropdown>
                        <DropdownButton as={NavbarItem} className='max-lg:hidden'>
                            <Cog8ToothIcon />
                            <NavbarLabel>{albumName || 'Loading...'}</NavbarLabel>
                            <ChevronDownIcon />
                        </DropdownButton>
                        <DropdownMenu className='min-w-80 lg:min-w-64' anchor='bottom start'>
                            <DropdownItem to='/admin/albums'>
                                <ArrowLeftIcon />
                                <DropdownLabel>Return to Album listing</DropdownLabel>
                            </DropdownItem>
                            <DropdownDivider />
                            {albums.map((album) => (
                                <DropdownItem key={album.id} to={`/admin/albums/view/${album.slug}`}>
                                    <Cog8ToothIcon />
                                    <DropdownLabel>{album.name}</DropdownLabel>
                                </DropdownItem>
                            ))}
                            <Can permission={'album.create'}>
                                <DropdownDivider />
                                <DropdownItem to='/admin/albums/create'>
                                    <PlusIcon />
                                    <DropdownLabel>New album&hellip;</DropdownLabel>
                                </DropdownItem>
                            </Can>
                        </DropdownMenu>
                    </Dropdown>
                    <NavbarSection className='max-lg:hidden'>
                        {navItems.map(({ path, permission, name }) => (
                            <Can permission={permission} albumId={album.id} key={path}>
                                <NavbarItem to={`/admin/albums/view/${slug}${path}`}>{name}</NavbarItem>
                            </Can>
                        ))}
                    </NavbarSection>
                </Navbar>
            }
            sidebar={
                <Sidebar>
                    <SidebarBody>
                        <SidebarSection>
                            {navItems.map(({ path, permission, name }) => (
                                <Can permission={permission} albumId={album.id} key={path}>
                                    <SidebarItem to={`/admin/albums/view/${slug}${path}`}>{name}</SidebarItem>
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
                                <Can permission={permission} albumId={album.id} fallback={<NoPermission />}>
                                    <Component key={slug} />
                                </Can>
                            }
                        />
                    ))}
                    <Route
                        path='*'
                        element={<AdminNotFound backTo={`/admin/albums/view/${slug}`} backLabel='Back to album' />}
                    />
                </Route>
            </Routes>
        </StackedLayout>
    );
};

export default AdminAlbumRouter;
