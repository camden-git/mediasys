import React from 'react';
import clsx from 'clsx';

export const DescriptionList: React.FC<React.HTMLAttributes<HTMLDListElement>> = ({
    className,
    children,
    ...props
}) => {
    return (
        <dl className={clsx(className, 'divide-y divide-gray-200 dark:divide-white/10')} {...props}>
            {children}
        </dl>
    );
};

export const DescriptionTerm: React.FC<React.HTMLAttributes<HTMLElement>> = ({ className, children, ...props }) => {
    return (
        <dt
            className={clsx(
                className,
                'inline-block w-1/3 py-3 pr-4 text-sm font-medium text-gray-900 dark:text-white',
            )}
            {...props}
        >
            {children}
        </dt>
    );
};

export const DescriptionDetails: React.FC<React.HTMLAttributes<HTMLElement>> = ({ className, children, ...props }) => {
    return (
        <dd className={clsx(className, 'inline-block w-2/3 py-3 text-sm text-gray-700 dark:text-gray-300')} {...props}>
            {children}
        </dd>
    );
};

// Wrapper for a row in the description list
export const DescriptionItem: React.FC<{
    term: React.ReactNode;
    details: React.ReactNode;
    termClassName?: string;
    detailsClassName?: string;
    itemClassName?: string;
}> = ({ term, details, termClassName, detailsClassName, itemClassName }) => {
    return (
        <div className={clsx(itemClassName, 'pt-3 first:pt-0')}>
            {' '}
            {/* Adjusted for block layout */}
            <DescriptionTerm className={termClassName}>{term}</DescriptionTerm>
            <DescriptionDetails className={detailsClassName}>{details}</DescriptionDetails>
        </div>
    );
};
