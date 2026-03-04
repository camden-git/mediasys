import { useUIStore } from '../store/useUIStore';

export const useFlash = () => {
    const addFlash = useUIStore((s) => s.addFlash);
    const clearFlashes = useUIStore((s) => s.clearFlashes);
    const clearAndAddHttpError = useUIStore((s) => s.clearAndAddHttpError);

    return { addFlash, clearFlashes, clearAndAddHttpError };
};
