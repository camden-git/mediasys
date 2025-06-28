import http from '../http';
import { Album } from '../../types';

export const listAlbums = async (): Promise<Album[]> => {
    const response = await http.get('/albums');
    return response.data;
};
