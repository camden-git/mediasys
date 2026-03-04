import { useQuery } from '@tanstack/react-query';
import { listRoles, getRole, getPermissionDefinitions } from '../admin/roles';
import { AdminRoleResponse, PermissionGroupDefinition } from '../../types';
import { PaginatedResult, PaginationRequest } from '../standard';
import { queryKeys } from '../../lib/queryKeys';

export const useRoles = (params?: PaginationRequest) => {
    return useQuery<PaginatedResult<AdminRoleResponse>>({
        queryKey: queryKeys.roles.list(params),
        queryFn: () => listRoles(params),
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
