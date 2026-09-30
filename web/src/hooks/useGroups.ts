import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { AlbumGroup } from '../types';
import { getGroups, getGroup, getGroupPhotos } from '../api';
import { queryKeys } from '../lib/queryKeys';

const PHOTOS_PAGE_SIZE = 120;

export function useGroups() {
    const { data, error, isLoading } = useQuery<AlbumGroup[]>({
        queryKey: queryKeys.groups.list(),
        queryFn: () => getGroups(),
    });
    return { groups: data ?? [], isLoading, error };
}

export function useGroup(slug: string | undefined) {
    const { data, error, isLoading } = useQuery<AlbumGroup>({
        queryKey: queryKeys.groups.detail(slug!),
        queryFn: () => getGroup(slug!),
        enabled: !!slug,
    });
    return { group: data ?? null, isLoading, error };
}

export function useGroupPhotos(slug: string | undefined, minRating?: number) {
    return useInfiniteQuery({
        queryKey: queryKeys.groups.photos(slug!, minRating),
        queryFn: ({ pageParam, signal }) =>
            getGroupPhotos(slug!, { offset: pageParam, limit: PHOTOS_PAGE_SIZE, min_rating: minRating }, signal),
        initialPageParam: 0,
        getNextPageParam: (last, pages) =>
            last.has_more ? pages.reduce((n, p) => n + (p.files?.length ?? 0), 0) : undefined,
        enabled: !!slug,
    });
}
