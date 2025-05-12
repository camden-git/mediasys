import { useState, useEffect, useRef, RefObject } from 'react';

interface Size {
    width: number;
    height: number;
}

function useResizeObserver<T extends HTMLElement>(): [RefObject<T | null>, Size] {
    const ref = useRef<T>(null);
    const [size, setSize] = useState<Size>({ width: 0, height: 0 });

    useEffect(() => {
        const element = ref.current;
        if (!element) return;

        const observer = new ResizeObserver((entries) => {
            if (entries[0]) {
                const { width, height } = entries[0].contentRect;
                setSize({ width, height });
            }
        });

        observer.observe(element);

        const { width, height } = element.getBoundingClientRect();
        setSize({ width, height });

        return () => {
            observer.unobserve(element);
        };
    }, [ref]);

    return [ref, size];
}

export default useResizeObserver;
