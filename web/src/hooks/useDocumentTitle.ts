import { useEffect } from 'react';

const APP_NAME = 'Mediasys';

// Sets document.title while mounted and restores the previous title on unmount.
export function useDocumentTitle(title: string | null | undefined) {
    useEffect(() => {
        if (!title) return;
        const previous = document.title;
        document.title = `${title} | ${APP_NAME}`;
        return () => {
            document.title = previous;
        };
    }, [title]);
}
