import { create } from 'zustand';

interface ProgressState {
    progress: number | undefined;
    continuous: boolean;
    setProgress: (progress: number | undefined) => void;
    setContinuous: (continuous: boolean) => void;
    startContinuous: () => void;
    setComplete: () => void;
}

export const useProgressStore = create<ProgressState>()((set) => ({
    progress: undefined,
    continuous: false,

    setProgress: (progress) => set({ progress }),
    setContinuous: (continuous) => set({ continuous }),

    startContinuous: () => set({ continuous: true, progress: 20 }),

    setComplete: () => set({ progress: 100, continuous: false }),
}));
