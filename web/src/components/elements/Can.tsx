import React from 'react';
import { useStoreState } from '../../store/hooks';

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
    const currentUserPermissions = useStoreState((state: any) => state.auth.currentUserPermissions);

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
