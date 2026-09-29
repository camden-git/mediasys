import { useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { AdminAlbumResponse, getAlbumBySlug } from '../api/admin/albums';
import { queryKeys } from '../lib/queryKeys';

/**
 * The current admin album, read from the TanStack Query cache. Only valid beneath AdminAlbumRouter,
 * which renders its children only once the album has loaded.
 */
export const useAlbumData = (): AdminAlbumResponse => {
    const { slug } = useParams<{ slug: string }>();
    const { data } = useQuery({
        queryKey: queryKeys.albums.bySlug(slug!),
        queryFn: () => getAlbumBySlug(slug!),
        enabled: !!slug,
    });
    return data!;
};

export const useAlbumId = () => useAlbumData().id;
export const useAlbumName = () => useAlbumData().name;
export const useAlbumSlug = () => useAlbumData().slug;

export type { AdminAlbumResponse };
