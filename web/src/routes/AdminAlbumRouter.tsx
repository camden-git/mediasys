import React, { useEffect } from 'react';
import { Routes, Route, useParams, useNavigate } from 'react-router-dom';
import ProtectedRoute from '../components/router/ProtectedRoute';
import { StackedLayout } from '../components/elements/StackedLayout.tsx';
import { Navbar, NavbarItem, NavbarLabel, NavbarSection, NavbarSpacer } from '../components/elements/Navbar.tsx';
import { ArrowLeftIcon, InboxIcon, MagnifyingGlassIcon, PlusIcon } from '@heroicons/react/20/solid';
import { Sidebar, SidebarBody, SidebarItem, SidebarSection } from '../components/elements/Sidebar.tsx';
import AlbumView from '../components/admin/albums/AlbumView.tsx';
import EditAlbumForm from '../components/admin/albums/EditAlbumForm.tsx';
import AlbumSubusersPage from '../components/admin/albums/AlbumSubusersPage.tsx';
import { Can } from '../components/elements/Can.tsx';
import { useAlbums } from '../api/swr/useAlbums';
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
import { useStoreState, useStoreActions } from '../store/hooks';
import { getAlbum } from '../api/admin/albums';
import OverviewContainer from '../components/admin/albums/overview/OverviewContainer.tsx';
import { SettingsContainer } from '../components/admin/albums/settings/SettingsContainer.tsx';

export interface AdminAlbumRouteDefinition {
    path: string;
    name: string;
    component: React.ComponentType;
    permission: string | string[] | null;
}

const AdminAlbumRouter: React.FC = () => {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { albums } = useAlbums();
    const album = useStoreState((state) => state.albumContext.data);
    const isLoading = useStoreState((state) => state.albumContext.isLoading);
    const error = useStoreState((state) => state.albumContext.error);
    const setAlbum = useStoreActions((actions) => actions.albumContext.setAlbum);
    const setIsLoading = useStoreActions((actions) => actions.albumContext.setIsLoading);
    const setError = useStoreActions((actions) => actions.albumContext.setError);
    const clearAlbum = useStoreActions((actions) => actions.albumContext.clearAlbum);
    const albumName = useStoreState((state) => state.albumContext.data?.name);

    useEffect(() => {
        if (id && id !== 'create') {
            const albumId = parseInt(id, 10);
            if (!isNaN(albumId)) {
                setIsLoading(true);
                setError(null);

                getAlbum(albumId)
                    .then((albumData) => {
                        setAlbum(albumData);
                    })
                    .catch((err) => {
                        setError(err.response?.data?.error || 'Failed to load album');
                        console.error('Failed to load album:', err);
                    })
                    .finally(() => {
                        setIsLoading(false);
                    });
            } else {
                setError('Invalid album ID');
            }
        } else {
            clearAlbum();
        }
    }, [id, setAlbum, setIsLoading, setError, clearAlbum]);

    useEffect(() => {
        if (id && (isNaN(parseInt(id, 10)) || id === 'create')) {
            navigate('/admin/albums');
        }
    }, [id, navigate]);

    if (isLoading) {
        return (
            <div className='flex h-64 items-center justify-center'>
                <LoadingSpinner />
            </div>
        );
    }

    if (error || !album) {
        return <div className='text-center text-red-600'>Error loading album: {error || 'Album not found'}</div>;
    }

    const navItems: AdminAlbumRouteDefinition[] = [
        {
            path: '/',
            name: 'Details',
            component: AlbumView,
            permission: 'album.list',
        },
        {
            path: '/overview',
            name: 'Overview',
            component: OverviewContainer,
            permission: 'album.list',
        },
        {
            path: '/settings',
            name: 'Settings',
            component: SettingsContainer,
            permission: 'album.edit.general',
        },
        {
            path: '/edit',
            name: 'Edit',
            component: EditAlbumForm,
            permission: 'album.edit.general',
        },
        {
            path: '/subusers',
            name: 'Subusers',
            component: AlbumSubusersPage,
            permission: 'album.manage.members.global',
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
                                <DropdownItem key={album.id} to={`/admin/albums/view/${album.id}`}>
                                    <Cog8ToothIcon />
                                    <DropdownLabel>{album.name}</DropdownLabel>
                                </DropdownItem>
                            ))}
                            <DropdownDivider />
                            <DropdownItem to='/admin/albums/create'>
                                <PlusIcon />
                                <DropdownLabel>New team&hellip;</DropdownLabel>
                            </DropdownItem>
                        </DropdownMenu>
                    </Dropdown>
                    <NavbarSection className='max-lg:hidden'>
                        {navItems.map(({ path, permission, name }) => (
                            <Can permission={permission} key={path}>
                                <NavbarItem to={`/admin/albums/view/${id}${path}`}>{name}</NavbarItem>
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
                            {navItems.map(({ path, permission, name }) => (
                                <Can permission={permission} key={path}>
                                    <SidebarItem to={`/admin/albums/view/${id}${path}`}>{name}</SidebarItem>
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
                                <Can permission={permission}>
                                    <Component />
                                </Can>
                            }
                        />
                    ))}
                </Route>
            </Routes>
        </StackedLayout>
    );
};

export default AdminAlbumRouter;
