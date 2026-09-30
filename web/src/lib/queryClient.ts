import { QueryClient } from '@tanstack/react-query';
import { queryKeys } from './queryKeys';
import { isApiError } from '../api/errors';

// Retry once, and only for network failures and 5xx responses: a 4xx will fail the same way again.
const retryTransientOnce = (failureCount: number, error: unknown): boolean =>
    failureCount < 1 && !(isApiError(error) && (error.isClientError || error.isAbort));

export const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            staleTime: 30_000,
            retry: retryTransientOnce,
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
