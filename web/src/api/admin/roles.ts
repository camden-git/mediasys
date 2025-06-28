import http from '../http';
import { AdminRoleResponse, RoleCreatePayload, RoleUpdatePayload, UserSummary } from '../../types';

export const listRoles = async (): Promise<AdminRoleResponse[]> => {
    const response = await http.get('/admin/roles');
    return response.data;
};

export const getRole = async (roleId: number): Promise<AdminRoleResponse> => {
    const response = await http.get(`/admin/roles/${roleId}`);
    return response.data;
};

export const createRole = async (payload: RoleCreatePayload): Promise<AdminRoleResponse> => {
    const response = await http.post('/admin/roles', payload);
    return response.data;
};

export const updateRole = async (roleId: number, payload: RoleUpdatePayload): Promise<AdminRoleResponse> => {
    const response = await http.put(`/admin/roles/${roleId}`, payload);
    return response.data;
};

export const deleteRole = async (roleId: number): Promise<void> => {
    await http.delete(`/admin/roles/${roleId}`);
};

export const getRoleUsers = async (roleId: number): Promise<UserSummary[]> => {
    const response = await http.get(`/admin/roles/${roleId}/users`);
    return response.data;
};

export const addUserToRole = async (roleId: number, userId: number): Promise<void> => {
    await http.post(`/admin/roles/${roleId}/users`, { user_id: userId });
};

export const removeUserFromRole = async (roleId: number, userId: number): Promise<void> => {
    await http.delete(`/admin/roles/${roleId}/users/${userId}`);
};

export const getPermissionDefinitions = async () => {
    const response = await http.get('/permissions');
    return response.data;
};
