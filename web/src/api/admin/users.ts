import http from '../http';
import { AdminUserResponse, UserCreatePayload, UserUpdatePayload } from '../../types';
import { ApiResponse, PaginatedResult, PaginationRequest, toPaginatedResult, toPaginationQuery } from '../standard';

export const listUsers = async (
    params?: PaginationRequest,
    signal?: AbortSignal,
): Promise<PaginatedResult<AdminUserResponse>> => {
    const response = await http.get<ApiResponse<AdminUserResponse[]>>('/admin/users', {
        params: toPaginationQuery(params),
        signal,
    });
    return toPaginatedResult(response.data);
};

export const getUser = async (userId: number, signal?: AbortSignal): Promise<AdminUserResponse> => {
    const response = await http.get<ApiResponse<AdminUserResponse>>(`/admin/users/${userId}`, { signal });
    return response.data.data;
};

export const createUser = async (payload: UserCreatePayload): Promise<AdminUserResponse> => {
    const response = await http.post<ApiResponse<AdminUserResponse>>('/admin/users', payload);
    return response.data.data;
};

export const updateUser = async (userId: number, payload: UserUpdatePayload): Promise<AdminUserResponse> => {
    const response = await http.put<ApiResponse<AdminUserResponse>>(`/admin/users/${userId}`, payload);
    return response.data.data;
};
