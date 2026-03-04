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
        detail: (id: number) => [...queryKeys.roles.all(), id] as const,
        permissionDefinitions: () => ['permission-definitions'] as const,
    },
    users: {
        all: () => ['users'] as const,
        list: (p?: PaginationRequest) => [...queryKeys.users.all(), 'list', p ?? {}] as const,
        detail: (id: number) => [...queryKeys.users.all(), id] as const,
    },
    inviteCodes: {
        all: () => ['invite-codes'] as const,
        list: (p?: PaginationRequest) => [...queryKeys.inviteCodes.all(), 'list', p ?? {}] as const,
    },
    groups: {
        list: () => ['groups', 'list'] as const,
        detail: (slug: string) => ['groups', slug] as const,
    },
    collections: {
        list: () => ['collections', 'list'] as const,
        detail: (slug: string) => ['collections', slug] as const,
    },
    publicAlbum: {
        detail: (id: string) => ['public-album', id] as const,
        contents: (id: string) => ['public-album', id, 'contents'] as const,
    },
};
