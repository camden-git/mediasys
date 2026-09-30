import { useQuery, useInfiniteQuery } from '@tanstack/react-query';
import { getAlbumDetails, getAlbumContents } from '../albums';
import { queryKeys } from '../../lib/queryKeys';

const PAGE_SIZE = 50;
const HIGHLIGHT_MIN_RATING = 4;

export const usePublicAlbumDetail = (identifier?: string) =>
    useQuery({
        queryKey: queryKeys.publicAlbum.detail(identifier!),
        queryFn: ({ signal }) => getAlbumDetails(identifier!, signal),
        enabled: !!identifier,
    });

export const usePublicAlbumContents = (identifier?: string, enabled = true) =>
    useInfiniteQuery({
        queryKey: queryKeys.publicAlbum.contents(identifier!),
        queryFn: ({ pageParam, signal }) =>
            getAlbumContents(identifier!, { offset: pageParam as number, limit: PAGE_SIZE }, signal),
        initialPageParam: 0 as number,
        getNextPageParam: (last) => (last.has_more ? last.offset + last.files.length : undefined),
        enabled: !!identifier && enabled,
    });

export const usePublicAlbumHighlights = (identifier?: string, enabled = true) =>
    useInfiniteQuery({
        queryKey: [...queryKeys.publicAlbum.contents(identifier!), 'highlights'] as const,
        queryFn: ({ pageParam, signal }) =>
            getAlbumContents(
                identifier!,
                { offset: pageParam as number, limit: PAGE_SIZE, min_rating: HIGHLIGHT_MIN_RATING },
                signal,
            ),
        initialPageParam: 0 as number,
        getNextPageParam: (last) => (last.has_more ? last.offset + last.files.length : undefined),
        enabled: !!identifier && enabled,
    });
