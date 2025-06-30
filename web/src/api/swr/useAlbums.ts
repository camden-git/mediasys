import { listAlbums, getAlbum, AdminAlbumResponse } from '../admin/albums';
import { getAlbumUsers, getAvailableUsers, AlbumUserPermissionResponse } from '../admin/albums';
import { User } from '../../types';
import useSWR from 'swr';

export const useAlbums = () => {
    const { data, error, isLoading, mutate } = useSWR<AdminAlbumResponse[]>('admin/albums', () => listAlbums());

    return {
        albums: data || [],
        isLoading,
        error,
        mutate,
    };
};

export const useAlbum = (id: number | null) => {
    const { data, error, isLoading, mutate } = useSWR<AdminAlbumResponse>(id ? `admin/albums/${id}` : null, () =>
        getAlbum(id!),
    );

    return {
        album: data,
        isLoading,
        error,
        mutate,
    };
};

export const useAlbumUsers = (albumId: number | null) => {
    const { data, error, isLoading, mutate } = useSWR<AlbumUserPermissionResponse[]>(
        albumId ? `admin/albums/${albumId}/users` : null,
        () => getAlbumUsers(albumId!),
    );

    return {
        users: data || [],
        isLoading,
        error,
        mutate,
    };
};

export const useAvailableUsers = (albumId: number | null) => {
    const { data, error, isLoading, mutate } = useSWR<User[]>(
        albumId ? `admin/albums/${albumId}/users/available` : null,
        () => getAvailableUsers(albumId!),
    );

    return {
        users: data || [],
        isLoading,
        error,
        mutate,
    };
};
