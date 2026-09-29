import { QueryClient } from '@tanstack/react-query';
import { queryKeys } from './queryKeys';

export const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            staleTime: 30_000,
            retry: 1,
            refetchOnWindowFocus: false,
        },
    },
});

/** Invalidate the admin album queries and the public album queries derived from them. */
export const invalidateAlbums = (refetchType: 'active' | 'none' = 'active') =>
    Promise.all(
        [queryKeys.albums.all(), queryKeys.publicAlbum.all()].map((queryKey) =>
            queryClient.invalidateQueries({ queryKey, refetchType }),
        ),
    );
