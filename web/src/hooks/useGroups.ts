import { useQuery } from '@tanstack/react-query';
import { AlbumGroup } from '../types';
import { getGroups, getGroup } from '../api';
import { queryKeys } from '../lib/queryKeys';

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
