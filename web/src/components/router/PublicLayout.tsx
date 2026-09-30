import React from 'react';
import { Outlet } from 'react-router-dom';
import { useAuthStore } from '../../store/useAuthStore';
import { StackedLayout } from '../elements/StackedLayout.tsx';
import { Navbar, NavbarItem, NavbarSection, NavbarSpacer } from '../elements/Navbar.tsx';
import { Sidebar, SidebarBody, SidebarItem, SidebarSection } from '../elements/Sidebar.tsx';

const navItems = [
    { label: 'Index', url: '/' },
    { label: 'Groups', url: '/groups' },
    { label: 'Collections', url: '/collections' },
    { label: 'People', url: '/people' },
];

// Shared top nav for the public browsing pages (groups, collections, people).
const PublicLayout: React.FC = () => {
    const isAuthenticated = useAuthStore((s) => s.isAuthenticated());
    const isInitializing = useAuthStore((s) => s.isInitializing);
    // Logged-out visitors get a discreet way to the login page; signed-in users get the admin area.
    const accountItem = isAuthenticated ? { label: 'Admin', url: '/admin' } : { label: 'Sign in', url: '/auth/login' };

    return (
        <StackedLayout
            navbar={
                <Navbar>
                    <NavbarSection className='max-lg:hidden'>
                        {navItems.map(({ label, url }) => (
                            <NavbarItem key={label} to={url} includeSubPaths>
                                {label}
                            </NavbarItem>
                        ))}
                    </NavbarSection>
                    <NavbarSpacer />
                    {!isInitializing && (
                        <NavbarSection className='max-lg:hidden'>
                            <NavbarItem to={accountItem.url}>{accountItem.label}</NavbarItem>
                        </NavbarSection>
                    )}
                </Navbar>
            }
            sidebar={
                <Sidebar>
                    <SidebarBody>
                        <SidebarSection>
                            {navItems.map(({ label, url }) => (
                                <SidebarItem key={label} to={url}>
                                    {label}
                                </SidebarItem>
                            ))}
                            {!isInitializing && <SidebarItem to={accountItem.url}>{accountItem.label}</SidebarItem>}
                        </SidebarSection>
                    </SidebarBody>
                </Sidebar>
            }
        >
            <Outlet />
        </StackedLayout>
    );
};

export default PublicLayout;
