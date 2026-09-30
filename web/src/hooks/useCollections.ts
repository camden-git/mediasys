import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { Collection } from '../types';
import { listPublicCollections, getPublicCollection, getCollectionPhotos } from '../api/collections';
import { queryKeys } from '../lib/queryKeys';

const PHOTOS_PAGE_SIZE = 120;

export function useCollections() {
    const { data, error, isLoading } = useQuery<Collection[]>({
        queryKey: queryKeys.collections.list(),
        queryFn: () => listPublicCollections(),
    });
    return { collections: data ?? [], isLoading, error };
}

export function useCollection(slug: string | undefined) {
    const { data, error, isLoading } = useQuery<Collection>({
        queryKey: queryKeys.collections.detail(slug!),
        queryFn: () => getPublicCollection(slug!),
        enabled: !!slug,
    });
    return { collection: data ?? null, isLoading, error };
}

export function useCollectionPhotos(slug: string | undefined) {
    return useInfiniteQuery({
        queryKey: queryKeys.collections.photos(slug!),
        queryFn: ({ pageParam, signal }) => getCollectionPhotos(slug!, pageParam, PHOTOS_PAGE_SIZE, signal),
        initialPageParam: 0,
        getNextPageParam: (last, pages) =>
            last.has_more ? pages.reduce((n, p) => n + (p.files?.length ?? 0), 0) : undefined,
        enabled: !!slug,
    });
}
