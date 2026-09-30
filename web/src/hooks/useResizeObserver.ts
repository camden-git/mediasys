import { useState, useEffect, useRef, useCallback, RefCallback, RefObject } from 'react';

interface Size {
    width: number;
    height: number;
}

// Returns a ref object (for reading the element) plus a callback ref to attach to it,
// so elements that mount late are still observed.
function useResizeObserver<T extends HTMLElement>(): [RefObject<T | null>, Size, RefCallback<T>] {
    const ref = useRef<T | null>(null);
    const observerRef = useRef<ResizeObserver | null>(null);
    const [size, setSize] = useState<Size>({ width: 0, height: 0 });

    const setRef = useCallback((element: T | null) => {
        observerRef.current?.disconnect();
        observerRef.current = null;
        ref.current = element;
        if (!element) return;

        // use the same box (content box) for the initial value and for updates
        const observer = new ResizeObserver((entries) => {
            const entry = entries[entries.length - 1];
            if (entry) {
                const { width, height } = entry.contentRect;
                setSize((prev) => (prev.width === width && prev.height === height ? prev : { width, height }));
            }
        });
        observer.observe(element);
        observerRef.current = observer;
    }, []);

    useEffect(() => () => observerRef.current?.disconnect(), []);

    return [ref, size, setRef];
}

export default useResizeObserver;
