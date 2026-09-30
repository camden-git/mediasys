import React from 'react';
import { Outlet } from 'react-router-dom';
import { StackedLayout } from '../elements/StackedLayout.tsx';
import { Navbar, NavbarItem, NavbarSection } from '../elements/Navbar.tsx';
import { Sidebar, SidebarBody, SidebarItem, SidebarSection } from '../elements/Sidebar.tsx';

const navItems = [
    { label: 'Index', url: '/' },
    { label: 'Groups', url: '/groups' },
    { label: 'Collections', url: '/collections' },
    { label: 'People', url: '/people' },
];

// Shared top nav for the public browsing pages (groups, collections, people).
const PublicLayout: React.FC = () => (
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
                    </SidebarSection>
                </SidebarBody>
            </Sidebar>
        }
    >
        <Outlet />
    </StackedLayout>
);

export default PublicLayout;
