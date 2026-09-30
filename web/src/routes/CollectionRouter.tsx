import React from 'react';
import { Routes, Route } from 'react-router-dom';
import CollectionsPage from '../components/collections/CollectionsPage.tsx';
import CollectionView from '../components/collections/CollectionView.tsx';
import NotFound from '../components/NotFound.tsx';
import { StackedLayout } from '../components/elements/StackedLayout.tsx';
import { Navbar, NavbarItem, NavbarSection } from '../components/elements/Navbar.tsx';
import { Sidebar, SidebarBody, SidebarItem, SidebarSection } from '../components/elements/Sidebar.tsx';

const navItems = [
    { label: 'Index', url: '/' },
    { label: 'Groups', url: '/groups' },
    { label: 'Collections', url: '/collections' },
    { label: 'People', url: '/people' },
];

const CollectionRouter: React.FC = () => {
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
            <Routes>
                <Route index element={<CollectionsPage />} />
                <Route path=':slug' element={<CollectionView />} />
                <Route path='*' element={<NotFound />} />
            </Routes>
        </StackedLayout>
    );
};

export default CollectionRouter;
