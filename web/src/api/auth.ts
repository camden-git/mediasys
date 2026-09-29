import http from './http';
import { LoginPayload, RegisterPayload, AuthResponse, User } from '../types';

export const loginUser = async (payload: LoginPayload): Promise<AuthResponse> => {
    const response = await http.post('/auth/login', payload);
    return response.data;
};

export const registerUser = async (payload: RegisterPayload): Promise<{ message: string }> => {
    const response = await http.post('/auth/register', payload);
    return response.data;
};

export const getCurrentUser = async (): Promise<User> => {
    const response = await http.get('/auth/me');
    return response.data;
};

export interface UpdateProfilePayload {
    first_name?: string;
    last_name?: string;
    username?: string;
    current_password?: string;
    new_password?: string;
}

export const updateProfile = async (payload: UpdateProfilePayload): Promise<User> => {
    const response = await http.put('/auth/me', payload);
    return response.data.data;
};
