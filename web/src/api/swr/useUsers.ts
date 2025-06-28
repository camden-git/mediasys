import useSWR, { mutate } from 'swr';
import { listUsers, getUser, updateUser } from '../admin/users';
import { AdminUserResponse, UserUpdatePayload } from '../../types';

export const useUsers = () => {
    return useSWR<AdminUserResponse[]>('users', listUsers);
};

export const useUser = (userId: number) => {
    return useSWR<AdminUserResponse>(userId ? `user-${userId}` : null, () => getUser(userId));
};

export const updateUserMutation = async (id: number, payload: UserUpdatePayload) => {
    const updatedUser = await updateUser(id, payload);

    // Update the cache
    mutate('users');
    mutate(`user-${id}`, updatedUser);

    return updatedUser;
};
