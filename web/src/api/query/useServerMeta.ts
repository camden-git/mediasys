import { useQuery } from '@tanstack/react-query';
import { getServerMeta } from '../meta';
import { queryKeys } from '../../lib/queryKeys';

// Server feature flags rarely change, so don't refetch them on every mount.
const META_STALE_TIME = 5 * 60 * 1000;

export const useServerMeta = () => {
    const { data, error, isLoading } = useQuery({
        queryKey: queryKeys.meta.all(),
        queryFn: ({ signal }) => getServerMeta(signal),
        staleTime: META_STALE_TIME,
    });
    // Until the flag is known (or if the request fails) assume enabled, so a hiccup never
    // hides working features behind a "disabled" notice.
    return { meta: data ?? null, faceRecognitionEnabled: data?.face_recognition_enabled ?? true, isLoading, error };
};
