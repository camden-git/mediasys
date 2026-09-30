import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FileInfo } from '../types';

interface Options {
    images: FileInfo[];
    hasMore: boolean;
    isFetchingMore: boolean;
    fetchMore: () => unknown;
    // changing this resets the selection and scroll guards (e.g. slug change)
    resetKey: string | undefined;
}

// Shared infinite-scroll + lightbox navigation for paginated photo pages.
export function usePaginatedLightbox({ images, hasMore, isFetchingMore, fetchMore, resetKey }: Options) {
    const [selectedImage, setSelectedImage] = useState<FileInfo | null>(null);
    const sentinelRef = useRef<HTMLDivElement | null>(null);
    const hasUserScrolledRef = useRef(false);
    const prefillCountRef = useRef(0);
    const pendingAdvanceRef = useRef(false);

    const loadMore = useCallback(() => {
        if (isFetchingMore || !hasMore) return;
        void fetchMore();
    }, [isFetchingMore, hasMore, fetchMore]);

    useEffect(() => {
        prefillCountRef.current = 0;
        hasUserScrolledRef.current = false;
        pendingAdvanceRef.current = false;
        setSelectedImage(null);
    }, [resetKey]);

    useEffect(() => {
        const onScroll = () => {
            if (window.scrollY > 0) hasUserScrolledRef.current = true;
        };
        window.addEventListener('scroll', onScroll, { passive: true });
        return () => window.removeEventListener('scroll', onScroll);
    }, []);

    useEffect(() => {
        const el = sentinelRef.current;
        if (!el) return;
        const observer = new IntersectionObserver(
            (entries) => {
                if (entries[0].isIntersecting && hasUserScrolledRef.current) loadMore();
            },
            { rootMargin: '0px 0px 300px 0px', threshold: 0.01 },
        );
        observer.observe(el);
        return () => observer.disconnect();
    }, [loadMore, images.length]);

    // if content doesn't fill the viewport, load one more page without requiring a scroll
    useEffect(() => {
        if (!hasMore) return;
        const viewportH = window.innerHeight || 0;
        const contentH = document.documentElement.scrollHeight || 0;
        if (contentH <= viewportH + 40 && prefillCountRef.current < 1) {
            prefillCountRef.current += 1;
            loadMore();
        }
    }, [images.length, hasMore, loadMore]);

    const selectedIndex = useMemo(() => {
        if (!selectedImage) return -1;
        return images.findIndex((f) => f.path === selectedImage.path);
    }, [selectedImage, images]);

    const onPrev = useCallback(() => {
        if (selectedIndex > 0) setSelectedImage(images[selectedIndex - 1]);
    }, [selectedIndex, images]);

    const onNext = useCallback(() => {
        if (selectedIndex < 0) return;
        if (selectedIndex < images.length - 1) {
            setSelectedImage(images[selectedIndex + 1]);
        } else if (hasMore) {
            pendingAdvanceRef.current = true;
            loadMore();
        }
    }, [selectedIndex, images, hasMore, loadMore]);

    // preload the next page when near the end of the loaded images
    useEffect(() => {
        if (selectedImage && selectedIndex >= images.length - 20) loadMore();
    }, [selectedImage, selectedIndex, images.length, loadMore]);

    // flush a pending "next" once new images arrive
    useEffect(() => {
        if (pendingAdvanceRef.current && selectedIndex >= 0 && images.length > selectedIndex + 1) {
            pendingAdvanceRef.current = false;
            setSelectedImage(images[selectedIndex + 1]);
        }
    }, [images, selectedIndex]);

    return {
        selectedImage,
        selectedIndex,
        sentinelRef,
        onImageClick: setSelectedImage,
        onClose: () => setSelectedImage(null),
        onPrev,
        onNext,
        canPrev: selectedIndex > 0,
        canNext: selectedIndex >= 0 && (selectedIndex < images.length - 1 || hasMore),
    };
}
