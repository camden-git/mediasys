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
