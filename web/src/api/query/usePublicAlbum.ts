import { useQuery, useInfiniteQuery } from '@tanstack/react-query';
import { getAlbumDetails, getAlbumContents } from '../../api';
import { queryKeys } from '../../lib/queryKeys';

export const usePublicAlbumDetail = (identifier?: string) =>
    useQuery({
        queryKey: queryKeys.publicAlbum.detail(identifier!),
        queryFn: ({ signal }) => getAlbumDetails(identifier!, signal),
        enabled: !!identifier,
    });

export const usePublicAlbumContents = (identifier?: string) =>
    useInfiniteQuery({
        queryKey: queryKeys.publicAlbum.contents(identifier!),
        queryFn: ({ pageParam, signal }) =>
            getAlbumContents(identifier!, { offset: pageParam as number, limit: 50 }, signal),
        initialPageParam: 0 as number,
        getNextPageParam: (last) =>
            last.has_more ? (last.offset ?? 0) + (last.files?.length ?? 0) : undefined,
        enabled: !!identifier,
    });
