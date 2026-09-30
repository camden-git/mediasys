import http, { unwrap } from './http';
import { AuthResponse, LoginPayload, RegisterPayload, User } from '../types';
import { ApiResponse } from './standard';

export interface UpdateProfilePayload {
    first_name?: string;
    last_name?: string;
    username?: string;
    current_password?: string;
    new_password?: string;
}

export const loginUser = async (payload: LoginPayload): Promise<AuthResponse> =>
    unwrap(await http.post<ApiResponse<AuthResponse>>('/auth/login', payload));

export const registerUser = async (payload: RegisterPayload): Promise<{ message: string }> =>
    unwrap(await http.post<ApiResponse<{ message: string }>>('/auth/register', payload));

export const getCurrentUser = async (signal?: AbortSignal): Promise<User> =>
    unwrap(await http.get<ApiResponse<User>>('/auth/me', { signal }));

export const updateProfile = async (payload: UpdateProfilePayload): Promise<User> =>
    unwrap(await http.put<ApiResponse<User>>('/auth/me', payload));
