import http from '../http';
import { AdminUserResponse, UserCreatePayload, UserUpdatePayload } from '../../types';

export const listUsers = async (): Promise<AdminUserResponse[]> => {
    const response = await http.get('/admin/users');
    return response.data;
};

export const getUser = async (userId: number): Promise<AdminUserResponse> => {
    const response = await http.get(`/admin/users/${userId}`);
    return response.data;
};

export const createUser = async (payload: UserCreatePayload): Promise<AdminUserResponse> => {
    const response = await http.post('/admin/users', payload);
    return response.data;
};

export const updateUser = async (userId: number, payload: UserUpdatePayload): Promise<AdminUserResponse> => {
    const response = await http.put(`/admin/users/${userId}`, payload);
    return response.data;
};

export const deleteUser = async (userId: number): Promise<void> => {
    await http.delete(`/admin/users/${userId}`);
};
