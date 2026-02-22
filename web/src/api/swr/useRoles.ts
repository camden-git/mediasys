import useSWR from 'swr';
import { listRoles, getRole, getPermissionDefinitions } from '../admin/roles';
import { AdminRoleResponse, PermissionGroupDefinition } from '../../types';
import { PaginatedResult, PaginationRequest } from '../standard';

const rolesKey = (params?: PaginationRequest) => ['roles', params?.page ?? 1, params?.perPage ?? undefined];

export const useRoles = (params?: PaginationRequest) => {
    return useSWR<PaginatedResult<AdminRoleResponse>>(rolesKey(params), () => listRoles(params));
};

export const useRole = (roleId: number) => {
    return useSWR<AdminRoleResponse>(roleId ? `role-${roleId}` : null, () => getRole(roleId));
};

export const usePermissionDefinitions = () => {
    return useSWR<PermissionGroupDefinition[]>('permission-definitions', getPermissionDefinitions);
};
