import React from 'react';
import { useAuthStore } from '../../store/useAuthStore';

interface CanProps {
    permission: string | string[] | null;
    requireAll?: boolean;
    // when provided, permissions granted for this specific album (directly or via a role,
    // whether album-specific or "for all albums") are also considered a match. This lets
    // users who only have per-album rights (no global permission) see the relevant controls.
    albumId?: number | string | null;
    // when true, a user who holds ANY album-scoped permission for ANY album (or "for all
    // albums") also satisfies the check, even without albumId or a matching global permission.
    // Useful for nav entries that link to an album list/section rather than a specific album.
    allowAnyAlbumAccess?: boolean;
    // rendered instead of the children when the check fails (defaults to nothing)
    fallback?: React.ReactNode;
    children: React.ReactNode;
}

const matchesPermission = (perm: string, userPerms: string[]): boolean => {
    // direct match
    if (userPerms.includes(perm)) return true;

    // wildcard match
    if (perm.includes('*')) {
        const escaped = perm
            .split('*')
            .map((part) => part.replace(/[.+?^${}()|[\]\\]/g, '\\$&'))
            .join('.*');
        const regex = new RegExp(`^${escaped}$`);
        return userPerms.some((up) => regex.test(up));
    }

    return false;
};

export const Can: React.FC<CanProps> = ({
    permission,
    requireAll = false,
    albumId,
    allowAnyAlbumAccess = false,
    fallback = null,
    children,
}) => {
    const user = useAuthStore((s) => s.user);
    const currentUserPermissions = React.useMemo(() => {
        if (!user) return [];
        const perms = new Set<string>();
        user.global_permissions?.forEach((p) => perms.add(p));
        user.roles?.forEach((role) => role.global_permissions?.forEach((p) => perms.add(p)));

        if (albumId !== undefined && albumId !== null) {
            const albumGrants = user.effective_permissions?.album;
            albumGrants?.for_all?.forEach((p) => perms.add(p));
            const scoped = albumGrants?.by_album?.[String(albumId)];
            scoped?.forEach((p) => perms.add(p));
        }

        return Array.from(perms);
    }, [user, albumId]);

    const hasAnyAlbumPermission = React.useMemo(() => {
        const albumGrants = user?.effective_permissions?.album;
        if (!albumGrants) return false;
        if (albumGrants.for_all && albumGrants.for_all.length > 0) return true;
        return Object.values(albumGrants.by_album ?? {}).some((perms) => perms.length > 0);
    }, [user]);

    if (allowAnyAlbumAccess && hasAnyAlbumPermission) {
        return <>{children}</>;
    }

    // null or undefined means "allow access"
    if (permission == null) {
        return <>{children}</>;
    }

    if (!currentUserPermissions || currentUserPermissions.length === 0) {
        return <>{fallback}</>;
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

    return hasPermission ? <>{children}</> : <>{fallback}</>;
};
