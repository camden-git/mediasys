import { useEffect } from 'react';
import { useBlocker } from 'react-router-dom';
import { useUnsavedChangesStore } from '../../store/useUnsavedChangesStore';

/**
 * Blocks in-app navigation while any form has unsaved changes. React Router only supports one
 * blocker at a time, so this is mounted once at the app root and forms register their dirty
 * state via UnsavedChangesBar instead of calling useBlocker themselves.
 */
export function UnsavedChangesGuard() {
    const hasUnsavedChanges = useUnsavedChangesStore((s) => s.dirtyForms.size > 0);

    const blocker = useBlocker(
        ({ currentLocation, nextLocation }) =>
            hasUnsavedChanges &&
            (currentLocation.pathname !== nextLocation.pathname || currentLocation.search !== nextLocation.search),
    );

    useEffect(() => {
        if (blocker.state !== 'blocked') return;
        if (window.confirm('You have unsaved changes. Leave this page and discard them?')) {
            blocker.proceed();
        } else {
            blocker.reset();
        }
    }, [blocker]);

    return null;
}
