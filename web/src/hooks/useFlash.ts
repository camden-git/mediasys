import { useStoreActions } from '../store/hooks';

export const useFlash = () => {
    const addFlash = useStoreActions((actions: any) => actions.ui.addFlash);
    const clearFlashes = useStoreActions((actions: any) => actions.ui.clearFlashes);
    const clearAndAddHttpError = useStoreActions((actions: any) => actions.ui.clearAndAddHttpError);

    return {
        addFlash,
        clearFlashes,
        clearAndAddHttpError,
    };
};
