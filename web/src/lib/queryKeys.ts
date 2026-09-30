import { PaginationRequest } from '../api/standard';

export const queryKeys = {
    albums: {
        all: () => ['albums'] as const,
        list: () => [...queryKeys.albums.all(), 'list'] as const,
        detail: (id: number) => [...queryKeys.albums.all(), id] as const,
        bySlug: (slug: string) => [...queryKeys.albums.all(), 'slug', slug] as const,
        users: (id: number) => [...queryKeys.albums.detail(id), 'users'] as const,
        availableUsers: (id: number) => [...queryKeys.albums.detail(id), 'available-users'] as const,
    },
    roles: {
        all: () => ['roles'] as const,
        list: (p?: PaginationRequest) => [...queryKeys.roles.all(), 'list', p ?? {}] as const,
        allItems: () => [...queryKeys.roles.all(), 'all-items'] as const,
        detail: (id: number) => [...queryKeys.roles.all(), id] as const,
        users: (id: number, p?: PaginationRequest) => [...queryKeys.roles.detail(id), 'users', p ?? {}] as const,
        allUsers: (id: number) => [...queryKeys.roles.detail(id), 'all-users'] as const,
        permissionDefinitions: () => ['permission-definitions'] as const,
    },
    users: {
        all: () => ['users'] as const,
        list: (p?: PaginationRequest) => [...queryKeys.users.all(), 'list', p ?? {}] as const,
        allItems: () => [...queryKeys.users.all(), 'all-items'] as const,
        detail: (id: number) => [...queryKeys.users.all(), id] as const,
    },
    inviteCodes: {
        all: () => ['invite-codes'] as const,
        list: (p?: PaginationRequest) => [...queryKeys.inviteCodes.all(), 'list', p ?? {}] as const,
    },
    groups: {
        all: () => ['groups'] as const,
        list: () => [...queryKeys.groups.all(), 'list'] as const,
        detail: (slug: string) => [...queryKeys.groups.all(), slug] as const,
        photos: (slug: string, minRating?: number) =>
            [...queryKeys.groups.all(), slug, 'photos', minRating ?? null] as const,
    },
    collections: {
        all: () => ['collections'] as const,
        list: () => [...queryKeys.collections.all(), 'list'] as const,
        detail: (slug: string) => [...queryKeys.collections.all(), slug] as const,
        photos: (slug: string) => [...queryKeys.collections.all(), slug, 'photos'] as const,
    },
    publicAlbum: {
        all: () => ['public-album'] as const,
        detail: (id: string) => [...queryKeys.publicAlbum.all(), id] as const,
        contents: (id: string) => [...queryKeys.publicAlbum.all(), id, 'contents'] as const,
    },
};
