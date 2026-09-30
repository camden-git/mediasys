import React from 'react';
import { Routes, Route, useParams } from 'react-router-dom';
import AlbumView from '../components/album/AlbumView.tsx';
import { StackedLayout } from '../components/elements/StackedLayout.tsx';
import { Navbar, NavbarItem, NavbarSection } from '../components/elements/Navbar.tsx';
import { Sidebar, SidebarBody, SidebarItem, SidebarSection } from '../components/elements/Sidebar.tsx';
import { usePublicAlbumDetail } from '../api/query/usePublicAlbum';

const navItems = [{ label: 'Index', url: '/' }];

const AlbumRouter: React.FC = () => {
    const params = useParams<{ identifier: string }>();
    const identifier = params.identifier;

    const { isLoading } = usePublicAlbumDetail(identifier);

    if (identifier && isLoading) {
        return (
            <div className='mx-auto flex justify-center'>
                <svg
                    className='mr-3 h-6 w-6 animate-spin text-white'
                    xmlns='http://www.w3.org/2000/svg'
                    fill='none'
                    viewBox='0 0 24 24'
                >
                    <circle
                        className='opacity-25'
                        cx='12'
                        cy='12'
                        r='10'
                        stroke='currentColor'
                        strokeWidth='4'
                    ></circle>
                    <path
                        className='opacity-75'
                        fill='currentColor'
                        d='M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z'
                    ></path>
                </svg>
                <p className='my-auto font-light text-gray-400'>Loading {identifier}</p>
            </div>
        );
    }

    return (
        <StackedLayout
            navbar={
                <Navbar>
                    <NavbarSection className='max-lg:hidden'>
                        {navItems.map(({ label, url }) => (
                            <NavbarItem key={label} to={url}>
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
                <Route index element={<AlbumView />} />
                <Route path='image/*' element={<AlbumView />} />
            </Routes>
        </StackedLayout>
    );
};

export default AlbumRouter;
