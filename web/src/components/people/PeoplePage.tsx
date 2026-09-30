import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Person } from '../../types.ts';
import { getPeople } from '../../api/people';
import { errorMessage, isAbortError } from '../../api/errors';
import { useDocumentTitle } from '../../hooks/useDocumentTitle.ts';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import { UserCircleIcon } from '@heroicons/react/24/outline';

const PeoplePage: React.FC = () => {
    useDocumentTitle('People');
    const [people, setPeople] = useState<Person[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();
        (async () => {
            try {
                const data = await getPeople(controller.signal);
                setPeople(data);
            } catch (err: unknown) {
                if (!isAbortError(err)) {
                    setError(errorMessage(err, 'Failed to load people'));
                }
            } finally {
                setIsLoading(false);
            }
        })();
        return () => controller.abort();
    }, []);

    if (isLoading) return <LoadingSpinner />;
    if (error) return <p className='p-8 text-red-500'>{error}</p>;

    return (
        <div className='mx-auto max-w-6xl px-4 py-12'>
            <h1 className='mb-8 text-3xl font-bold text-gray-950 dark:text-white'>People</h1>
            {people.length === 0 && <p className='text-gray-500 dark:text-gray-400'>No people tagged yet.</p>}
            <div className='grid gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5'>
                {people.map((person) => (
                    <Link
                        key={person.id}
                        to={`/people/${person.id}`}
                        className='flex flex-col items-center gap-2 rounded-xl p-4 text-center transition-colors hover:bg-gray-100 dark:hover:bg-gray-800'
                    >
                        <UserCircleIcon className='size-16 text-gray-300 dark:text-gray-600' />
                        <span className='text-sm font-medium text-gray-950 dark:text-white'>{person.primary_name}</span>
                    </Link>
                ))}
            </div>
        </div>
    );
};

export default PeoplePage;
