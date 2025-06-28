import { action, Action } from 'easy-peasy';

export interface ProgressStore {
    progress: number | undefined;
    continuous: boolean;
    setProgress: Action<ProgressStore, number | undefined>;
    setContinuous: Action<ProgressStore, boolean>;
    startContinuous: Action<ProgressStore>;
    setComplete: Action<ProgressStore>;
}

const progressStore: ProgressStore = {
    progress: undefined,
    continuous: false,

    setProgress: action((state, progress) => {
        state.progress = progress;
    }),

    setContinuous: action((state, continuous) => {
        state.continuous = continuous;
    }),

    startContinuous: action((state) => {
        state.continuous = true;
        state.progress = 20;
    }),

    setComplete: action((state) => {
        state.progress = 100;
        state.continuous = false;
    }),
};

export default progressStore;
