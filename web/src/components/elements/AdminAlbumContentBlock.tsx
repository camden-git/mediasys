import React, { useEffect } from 'react';
import { clsx } from 'clsx';
import { useStoreState } from '../../store/hooks.ts';

export interface AdminAlbumContentBlockProps {
    title?: string;
    className?: string;
    width?: string;
    children?: React.ReactNode;
}
const AdminAlbumContentBlock: React.FC<AdminAlbumContentBlockProps> = ({
    title,
    className,
    width = 'max-w-6xl p-6 lg:p-10',
    children,
}) => {
    const album = useStoreState((state) => state.albumContext.data!);

    useEffect(() => {
        if (title) {
            document.title = album.name + ' - ' + title;
        }
    }, [title, album.name]);

    return (
        <>
            <div className={clsx('mx-auto', width, className)}>{children}</div>
        </>
    );
};

export default AdminAlbumContentBlock;
