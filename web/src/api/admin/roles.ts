import http from '../http';
import {
    AdminRoleResponse,
    PermissionGroupDefinition,
    RoleCreatePayload,
    RoleUpdatePayload,
    UserSummary,
} from '../../types';
import { ApiResponse, PaginatedResult, PaginationRequest, toPaginatedResult, toPaginationQuery } from '../standard';

export const listRoles = async (params?: PaginationRequest): Promise<PaginatedResult<AdminRoleResponse>> => {
    const response = await http.get<ApiResponse<AdminRoleResponse[]>>('/admin/roles', {
        params: toPaginationQuery(params),
    });
    return toPaginatedResult(response.data);
};

export const getRole = async (roleId: number): Promise<AdminRoleResponse> => {
    const response = await http.get<ApiResponse<AdminRoleResponse>>(`/admin/roles/${roleId}`);
    return response.data.data;
};

export const createRole = async (payload: RoleCreatePayload): Promise<AdminRoleResponse> => {
    const response = await http.post<ApiResponse<AdminRoleResponse>>('/admin/roles', payload);
    return response.data.data;
};

export const updateRole = async (roleId: number, payload: RoleUpdatePayload): Promise<AdminRoleResponse> => {
    const response = await http.put<ApiResponse<AdminRoleResponse>>(`/admin/roles/${roleId}`, payload);
    return response.data.data;
};

export const deleteRole = async (roleId: number): Promise<void> => {
    await http.delete(`/admin/roles/${roleId}`);
};

export const getRoleUsers = async (
    roleId: number,
    params?: PaginationRequest,
): Promise<PaginatedResult<UserSummary>> => {
    const response = await http.get<ApiResponse<UserSummary[]>>(`/admin/roles/${roleId}/users`, {
        params: toPaginationQuery(params),
    });
    return toPaginatedResult(response.data);
};

export const addUserToRole = async (roleId: number, userId: number): Promise<void> => {
    await http.post(`/admin/roles/${roleId}/users`, { user_id: userId });
};

export const removeUserFromRole = async (roleId: number, userId: number): Promise<void> => {
    await http.delete(`/admin/roles/${roleId}/users/${userId}`);
};

export const getPermissionDefinitions = async () => {
    const response = await http.get<ApiResponse<PermissionGroupDefinition[]>>('/permissions');
    return response.data.data;
};
