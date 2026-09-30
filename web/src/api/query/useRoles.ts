import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { listRoles, getRole, getRoleUsers, getPermissionDefinitions } from '../admin/roles';
import { fetchAllPages } from '../admin/paginate';
import { AdminRoleResponse, PermissionGroupDefinition, UserSummary } from '../../types';
import { PaginatedResult, PaginationRequest } from '../standard';
import { queryKeys } from '../../lib/queryKeys';

export const useRoles = (params?: PaginationRequest, options?: { enabled?: boolean }) => {
    return useQuery<PaginatedResult<AdminRoleResponse>>({
        queryKey: queryKeys.roles.list(params),
        queryFn: () => listRoles(params),
        enabled: options?.enabled ?? true,
    });
};

// Every role, across all pages (for pickers that must not silently truncate)
export const useAllRoles = (options?: { enabled?: boolean }) => {
    return useQuery<AdminRoleResponse[]>({
        queryKey: queryKeys.roles.allItems(),
        queryFn: () => fetchAllPages(listRoles),
        enabled: options?.enabled ?? true,
    });
};

export const useRoleUsers = (roleId: number, params: PaginationRequest) => {
    return useQuery<PaginatedResult<UserSummary>>({
        queryKey: queryKeys.roles.users(roleId, params),
        queryFn: () => getRoleUsers(roleId, params),
        enabled: !!roleId,
        placeholderData: keepPreviousData,
    });
};

// Every member of a role, across all pages
export const useAllRoleUsers = (roleId: number, options?: { enabled?: boolean }) => {
    return useQuery<UserSummary[]>({
        queryKey: queryKeys.roles.allUsers(roleId),
        queryFn: () => fetchAllPages((p) => getRoleUsers(roleId, p)),
        enabled: !!roleId && (options?.enabled ?? true),
    });
};

export const useRole = (roleId: number) => {
    return useQuery<AdminRoleResponse>({
        queryKey: queryKeys.roles.detail(roleId),
        queryFn: () => getRole(roleId),
        enabled: !!roleId,
    });
};

export const usePermissionDefinitions = () => {
    return useQuery<PermissionGroupDefinition[]>({
        queryKey: queryKeys.roles.permissionDefinitions(),
        queryFn: () => getPermissionDefinitions(),
    });
};
