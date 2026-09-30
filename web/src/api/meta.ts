import http, { unwrap } from './http';
import { ApiResponse } from './standard';

export interface ServerMeta {
    face_recognition_enabled: boolean;
}

export const getServerMeta = async (signal?: AbortSignal): Promise<ServerMeta> =>
    unwrap(await http.get<ApiResponse<ServerMeta>>('/meta', { signal }));
