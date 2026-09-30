import { create } from 'zustand';

interface UnsavedChangesState {
    // ids of mounted forms that currently have unsaved changes
    dirtyForms: Set<string>;
    setDirty: (id: string, dirty: boolean) => void;
}

export const useUnsavedChangesStore = create<UnsavedChangesState>()((set) => ({
    dirtyForms: new Set(),

    setDirty: (id, dirty) =>
        set((state) => {
            if (state.dirtyForms.has(id) === dirty) return state;
            const dirtyForms = new Set(state.dirtyForms);
            if (dirty) dirtyForms.add(id);
            else dirtyForms.delete(id);
            return { dirtyForms };
        }),
}));
