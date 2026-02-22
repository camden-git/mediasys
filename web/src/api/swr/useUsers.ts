import useSWR, { mutate } from 'swr';
import { listUsers, getUser, updateUser } from '../admin/users';
import { AdminUserResponse, UserUpdatePayload } from '../../types';
import { PaginatedResult, PaginationRequest } from '../standard';

const usersKey = (params?: PaginationRequest) => [
    'users',
    params?.page ?? 1,
    params?.perPage ?? undefined,
];

export const useUsers = (params?: PaginationRequest) => {
    return useSWR<PaginatedResult<AdminUserResponse>>(usersKey(params), () => listUsers(params));
};

export const useUser = (userId: number) => {
    return useSWR<AdminUserResponse>(userId ? `user-${userId}` : null, () => getUser(userId));
};

export const updateUserMutation = async (id: number, payload: UserUpdatePayload) => {
    const updatedUser = await updateUser(id, payload);

    mutate((key) => Array.isArray(key) && key[0] === 'users');
    mutate(`user-${id}`, updatedUser);

    return updatedUser;
};
