import React from 'react';
import { useAuthStore } from '../../store/useAuthStore';

interface CanProps {
    permission: string | string[] | null;
    requireAll?: boolean;
    children: React.ReactNode;
}

const matchesPermission = (perm: string, userPerms: string[]): boolean => {
    // direct match
    if (userPerms.includes(perm)) return true;

    // wildcard match
    if (perm.includes('*')) {
        const regex = new RegExp(`^${perm.replace(/\./g, '\\.').replace(/\*/g, '.*')}$`);
        return userPerms.some((up) => regex.test(up));
    }

    return false;
};

export const Can: React.FC<CanProps> = ({ permission, requireAll = false, children }) => {
    const user = useAuthStore((s) => s.user);
    const currentUserPermissions = React.useMemo(() => {
        if (!user) return [];
        const perms = new Set<string>();
        user.global_permissions?.forEach((p) => perms.add(p));
        user.roles?.forEach((role) => role.global_permissions?.forEach((p) => perms.add(p)));
        return Array.from(perms);
    }, [user]);

    // null or undefined means "allow access"
    if (permission == null) {
        return <>{children}</>;
    }

    if (!currentUserPermissions || currentUserPermissions.length === 0) {
        return null;
    }

    let hasPermission = false;

    if (typeof permission === 'string') {
        hasPermission = matchesPermission(permission, currentUserPermissions);
    } else if (Array.isArray(permission)) {
        if (permission.length === 0) {
            hasPermission = true;
        } else if (requireAll) {
            hasPermission = permission.every((p) => matchesPermission(p, currentUserPermissions));
        } else {
            hasPermission = permission.some((p) => matchesPermission(p, currentUserPermissions));
        }
    }

    return hasPermission ? <>{children}</> : null;
};
