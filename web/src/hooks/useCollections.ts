import useSWR from 'swr';
import { Collection } from '../types';
import { listPublicCollections, getPublicCollection } from '../api/collections';

const collectionsFetcher = () => listPublicCollections();
const collectionFetcher = ([, slug]: [string, string]) => getPublicCollection(slug);

export function useCollections() {
    const { data, error, isLoading } = useSWR<Collection[]>('/collections', collectionsFetcher, {
        revalidateOnFocus: false,
    });
    return { collections: data ?? [], isLoading, error };
}

export function useCollection(slug: string | undefined) {
    const { data, error, isLoading } = useSWR<Collection>(slug ? ['/collections', slug] : null, collectionFetcher, {
        revalidateOnFocus: false,
    });
    return { collection: data ?? null, isLoading, error };
}
