import useSWR from 'swr';
import { listAlbums } from '../admin/albums';
import { Album } from '../../types';

export const useAlbums = () => {
    return useSWR<Album[]>('albums', listAlbums);
};
