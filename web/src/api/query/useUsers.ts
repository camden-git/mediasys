import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listUsers, getUser, updateUser } from '../admin/users';
import { fetchAllPages } from '../admin/paginate';
import { AdminUserResponse, UserUpdatePayload } from '../../types';
import { PaginatedResult, PaginationRequest } from '../standard';
import { queryKeys } from '../../lib/queryKeys';
import { refreshAuthUser } from '../../store/useAuthStore';

export const useUsers = (params?: PaginationRequest) => {
    return useQuery<PaginatedResult<AdminUserResponse>>({
        queryKey: queryKeys.users.list(params),
        queryFn: ({ signal }) => listUsers(params, signal),
    });
};

// Every user, across all pages
export const useAllUsers = (options?: { enabled?: boolean }) => {
    return useQuery<AdminUserResponse[]>({
        queryKey: queryKeys.users.allItems(),
        queryFn: () => fetchAllPages(listUsers),
        enabled: options?.enabled ?? true,
    });
};

export const useUser = (userId: number) => {
    return useQuery<AdminUserResponse>({
        queryKey: queryKeys.users.detail(userId),
        queryFn: ({ signal }) => getUser(userId, signal),
        enabled: !!userId,
    });
};

export const useUpdateUser = () => {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: ({ id, payload }: { id: number; payload: UserUpdatePayload }) => updateUser(id, payload),
        onSuccess: (updatedUser) => {
            queryClient.invalidateQueries({ queryKey: queryKeys.users.all() });
            queryClient.invalidateQueries({ queryKey: queryKeys.roles.all() });
            refreshAuthUser(updatedUser.id);
            queryClient.setQueryData(queryKeys.users.detail(updatedUser.id), updatedUser);
        },
    });
};
