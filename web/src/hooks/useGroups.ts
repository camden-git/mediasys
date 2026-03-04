import useSWR from 'swr';
import { AlbumGroup } from '../types';
import { getGroups, getGroup } from '../api';

const groupsFetcher = () => getGroups();
const groupFetcher = ([, slug]: [string, string]) => getGroup(slug);

export function useGroups() {
    const { data, error, isLoading } = useSWR<AlbumGroup[]>('/groups', groupsFetcher, {
        revalidateOnFocus: false,
    });
    return { groups: data ?? [], isLoading, error };
}

export function useGroup(slug: string | undefined) {
    const { data, error, isLoading } = useSWR<AlbumGroup>(slug ? ['/groups', slug] : null, groupFetcher, {
        revalidateOnFocus: false,
    });
    return { group: data ?? null, isLoading, error };
}
