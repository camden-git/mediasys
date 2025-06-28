import useSWR from 'swr';
import { listRoles, getRole, getPermissionDefinitions } from '../admin/roles';
import { AdminRoleResponse, PermissionGroupDefinition } from '../../types';

export const useRoles = () => {
    return useSWR<AdminRoleResponse[]>('roles', listRoles);
};

export const useRole = (roleId: number) => {
    return useSWR<AdminRoleResponse>(roleId ? `role-${roleId}` : null, () => getRole(roleId));
};

export const usePermissionDefinitions = () => {
    return useSWR<PermissionGroupDefinition[]>('permission-definitions', getPermissionDefinitions);
};
