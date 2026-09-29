import { useQuery } from '@tanstack/react-query';
import {
    listAlbums,
    getAlbum,
    getAlbumUsers,
    getAvailableUsers,
    AdminAlbumResponse,
    AlbumUserPermissionResponse,
} from '../admin/albums';
import { User } from '../../types';
import { queryKeys } from '../../lib/queryKeys';

export const useAdminAlbums = () => {
    const { data, error, isLoading } = useQuery<AdminAlbumResponse[]>({
        queryKey: queryKeys.albums.list(),
        queryFn: listAlbums,
    });
    return { albums: data ?? [], isLoading, error };
};

export const useAdminAlbum = (id: number | null) => {
    const { data, error, isLoading } = useQuery<AdminAlbumResponse>({
        queryKey: queryKeys.albums.detail(id!),
        queryFn: () => getAlbum(id!),
        enabled: !!id,
    });
    return { album: data, isLoading, error };
};

export const useAlbumUsers = (albumId: number | null) => {
    const { data, error, isLoading, refetch } = useQuery<AlbumUserPermissionResponse[]>({
        queryKey: queryKeys.albums.users(albumId!),
        queryFn: () => getAlbumUsers(albumId!),
        enabled: !!albumId,
    });
    return { users: data ?? [], isLoading, error, mutate: refetch };
};

export const useAvailableUsers = (albumId: number | null) => {
    const { data, error, isLoading, refetch } = useQuery<User[]>({
        queryKey: queryKeys.albums.availableUsers(albumId!),
        queryFn: () => getAvailableUsers(albumId!),
        enabled: !!albumId,
    });
    return { users: data ?? [], isLoading, error, mutate: refetch };
};
