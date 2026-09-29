import http from './http';
import { User } from '../types';

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
