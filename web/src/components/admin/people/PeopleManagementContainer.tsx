import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Heading } from '../../elements/Heading';
import { Text } from '../../elements/Text';
import { Button } from '../../elements/Button';
import { Input } from '../../elements/Input';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../elements/Table';
import { Dialog, DialogActions, DialogDescription, DialogTitle } from '../../elements/Dialog';
import PageContentBlock from '../../elements/PageContentBlock.tsx';
import LoadingSpinner from '../../elements/LoadingSpinner';
import { Person } from '../../../types';
import { getPeople, createPerson, deletePerson } from '../../../api/people';

const PeopleManagementContainer: React.FC = () => {
    const navigate = useNavigate();
    const [people, setPeople] = useState<Person[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const [showCreate, setShowCreate] = useState(false);
    const [newName, setNewName] = useState('');
    const [creating, setCreating] = useState(false);
    const [createError, setCreateError] = useState<string | null>(null);

    const [personForDelete, setPersonForDelete] = useState<Person | null>(null);
    const [deleting, setDeleting] = useState(false);

    const load = () => {
        setLoading(true);
        setError(null);
        getPeople()
            .then(setPeople)
            .catch((e) => setError(e.message))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
    }, []);

    const handleCreate = async () => {
        if (!newName.trim()) return;
        setCreating(true);
        setCreateError(null);
        try {
            const person = await createPerson(newName.trim());
            setPeople((prev) => [...prev, person]);
            setNewName('');
            setShowCreate(false);
        } catch (e: any) {
            setCreateError(e.message);
        } finally {
            setCreating(false);
        }
    };

    const handleDelete = async () => {
        if (!personForDelete) return;
        setDeleting(true);
        try {
            await deletePerson(personForDelete.id);
            setPeople((prev) => prev.filter((p) => p.id !== personForDelete.id));
            setPersonForDelete(null);
        } catch (e: any) {
            setError(e.message);
            setPersonForDelete(null);
        } finally {
            setDeleting(false);
        }
    };

    if (loading) return <LoadingSpinner />;

    return (
        <PageContentBlock title='People Management'>
            <div className='mb-6 flex w-full flex-wrap items-end justify-between gap-4'>
                <div>
                    <Heading>People</Heading>
                    <Text className='mt-1'>Manage people for face recognition tagging.</Text>
                </div>
                <Button onClick={() => setShowCreate(true)}>Create Person</Button>
            </div>

            {error && <p className='mb-4 text-sm text-red-600'>{error}</p>}

            {showCreate && (
                <div className='mb-6 flex items-end gap-3 rounded-lg border border-zinc-200 p-4 dark:border-zinc-700'>
                    <div className='flex-1'>
                        <Input
                            placeholder='Person name'
                            value={newName}
                            onChange={(e) => setNewName(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
                            autoFocus
                        />
                        {createError && <p className='mt-1 text-xs text-red-600'>{createError}</p>}
                    </div>
                    <Button onClick={handleCreate} disabled={creating || !newName.trim()}>
                        {creating ? 'Saving…' : 'Save'}
                    </Button>
                    <Button
                        plain
                        onClick={() => {
                            setShowCreate(false);
                            setNewName('');
                            setCreateError(null);
                        }}
                    >
                        Cancel
                    </Button>
                </div>
            )}

            {people.length === 0 ? (
                <p className='text-sm text-gray-500'>No people found. Create one to get started.</p>
            ) : (
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableHeader>Name</TableHeader>
                            <TableHeader>ID</TableHeader>
                            <TableHeader>Actions</TableHeader>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {people.map((person) => (
                            <TableRow key={person.id} href={`/admin/people/${person.id}`}>
                                <TableCell className='font-medium'>{person.primary_name}</TableCell>
                                <TableCell className='text-zinc-500'>{person.id}</TableCell>
                                <TableCell>
                                    <div className='flex gap-2' onClick={(e) => e.stopPropagation()}>
                                        <Button plain onClick={() => navigate(`/admin/people/${person.id}`)}>
                                            Edit
                                        </Button>
                                        <Button
                                            plain
                                            className='text-red-600'
                                            onClick={() => setPersonForDelete(person)}
                                        >
                                            Delete
                                        </Button>
                                    </div>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            )}

            <Dialog open={!!personForDelete} onClose={() => setPersonForDelete(null)} size='sm'>
                <DialogTitle>Delete Person</DialogTitle>
                <DialogDescription>
                    Are you sure you want to delete &ldquo;{personForDelete?.primary_name}&rdquo;? This will remove all
                    face tags associated with this person.
                </DialogDescription>
                <DialogActions>
                    <Button plain onClick={() => setPersonForDelete(null)}>
                        Cancel
                    </Button>
                    <Button color='red' onClick={handleDelete} disabled={deleting}>
                        {deleting ? 'Deleting…' : 'Delete'}
                    </Button>
                </DialogActions>
            </Dialog>
        </PageContentBlock>
    );
};

export default PeopleManagementContainer;
