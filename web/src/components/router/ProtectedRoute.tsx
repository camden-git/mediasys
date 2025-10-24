import React from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useStoreState } from '../../store/hooks';
import LoadingSpinner from '../elements/LoadingSpinner';

// Using Record<string, unknown> for props type as it's currently empty
// but might have props added later. This satisfies ESLint's no-empty-interface rule.
type ProtectedRouteProps = Record<string, unknown>;

const ProtectedRoute: React.FC<ProtectedRouteProps> = () => {
    const isAuthenticated = useStoreState((state: any) => state.auth.isAuthenticated);
    const isInitializing = useStoreState((state: any) => state.auth.isInitializing);
    const location = useLocation();

    if (isInitializing) {
        // You might want to return a loading spinner or null here
        // while checking auth status
        return <LoadingSpinner />;
    }

    if (!isAuthenticated) {
        // Redirect them to the /login page, but save the current location they were
        // trying to go to when they were redirected. This allows us to send them
        // along to that page after they login, which is a nicer user experience
        // than dropping them off on the home page.
        return <Navigate to='/auth/login' state={{ from: location }} replace />;
    }

    return <Outlet />;
};

export default ProtectedRoute;
