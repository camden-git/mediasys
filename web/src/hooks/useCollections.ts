import { useQuery } from '@tanstack/react-query';
import { Collection } from '../types';
import { listPublicCollections, getPublicCollection } from '../api/collections';
import { queryKeys } from '../lib/queryKeys';

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
