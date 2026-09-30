import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Heading } from '../../elements/Heading';
import { Text } from '../../elements/Text';
import { Button } from '../../elements/Button';
import { Input } from '../../elements/Input';
import PageContentBlock from '../../elements/PageContentBlock.tsx';
import LoadingSpinner from '../../elements/LoadingSpinner';
import FaceThumbnail from '../faces/shared/FaceThumbnail';
import { Person, Alias } from '../../../types';
import {
    getPersonByIdAdmin,
    updatePerson,
    getPersonAliases,
    addPersonAlias,
    deletePersonAlias,
    setPersonKeyPhoto,
    getPersonKeyPhotoUrl,
} from '../../../api';

const PersonAdminView: React.FC = () => {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const personId = id ? parseInt(id, 10) : 0;

    const [person, setPerson] = useState<Person | null>(null);
    const [aliases, setAliases] = useState<Alias[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const [editingName, setEditingName] = useState(false);
    const [nameValue, setNameValue] = useState('');
    const [savingName, setSavingName] = useState(false);
    const [nameError, setNameError] = useState<string | null>(null);

    const [keyPhotoLoading, setKeyPhotoLoading] = useState(false);
    const [keyPhotoError, setKeyPhotoError] = useState<string | null>(null);

    const [newAlias, setNewAlias] = useState('');
    const [addingAlias, setAddingAlias] = useState(false);
    const [aliasError, setAliasError] = useState<string | null>(null);

    useEffect(() => {
        if (!personId) return;
        setLoading(true);
        setError(null);
        Promise.all([getPersonByIdAdmin(personId), getPersonAliases(personId)])
            .then(([p, a]) => {
                setPerson(p);
                setNameValue(p.primary_name);
                setAliases(a);
            })
            .catch((e) => setError(e.message))
            .finally(() => setLoading(false));
    }, [personId]);

    const handleSaveName = async () => {
        if (!nameValue.trim() || !person) return;
        setSavingName(true);
        setNameError(null);
        try {
            const updated = await updatePerson(person.id, nameValue.trim());
            setPerson(updated);
            setEditingName(false);
        } catch (e: any) {
            setNameError(e.message);
        } finally {
            setSavingName(false);
        }
    };

    const handleSetKeyPhoto = async (faceId: number | null) => {
        if (!person) return;
        setKeyPhotoLoading(true);
        setKeyPhotoError(null);
        try {
            const updated = await setPersonKeyPhoto(person.id, faceId);
            setPerson(updated);
        } catch (e: any) {
            setKeyPhotoError(e.message);
        } finally {
            setKeyPhotoLoading(false);
        }
    };

    const handleAddAlias = async () => {
        if (!newAlias.trim() || !person) return;
        setAddingAlias(true);
        setAliasError(null);
        try {
            const alias = await addPersonAlias(person.id, newAlias.trim());
            setAliases((prev) => [...prev, alias]);
            setNewAlias('');
        } catch (e: any) {
            setAliasError(e.message);
        } finally {
            setAddingAlias(false);
        }
    };

    const handleDeleteAlias = async (alias: Alias) => {
        if (!person) return;
        try {
            await deletePersonAlias(person.id, alias.id);
            setAliases((prev) => prev.filter((a) => a.id !== alias.id));
        } catch (e: any) {
            setAliasError(e.message);
        }
    };

    if (loading) return <LoadingSpinner />;
    if (error) return <p className='text-red-600'>{error}</p>;
    if (!person) return null;

    return (
        <PageContentBlock title={`Person: ${person.primary_name}`}>
            <Button plain onClick={() => navigate('/admin/people')} className='mb-4'>
                ← Back to People
            </Button>

            <div className='mb-8'>
                <Heading level={2}>Name</Heading>
                {editingName ? (
                    <div className='mt-2 flex items-center gap-3'>
                        <Input
                            value={nameValue}
                            onChange={(e) => setNameValue(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleSaveName()}
                            autoFocus
                            className='max-w-xs'
                        />
                        <Button onClick={handleSaveName} disabled={savingName || !nameValue.trim()}>
                            {savingName ? 'Saving…' : 'Save'}
                        </Button>
                        <Button
                            plain
                            onClick={() => {
                                setEditingName(false);
                                setNameValue(person.primary_name);
                                setNameError(null);
                            }}
                        >
                            Cancel
                        </Button>
                    </div>
                ) : (
                    <div className='mt-2 flex items-center gap-3'>
                        <Text>{person.primary_name}</Text>
                        <Button plain onClick={() => setEditingName(true)}>
                            Edit
                        </Button>
                    </div>
                )}
                {nameError && <p className='mt-1 text-xs text-red-600'>{nameError}</p>}
            </div>

            <div className='mb-8'>
                <Heading level={2}>Key Photo</Heading>
                <Text className='mt-1 mb-4 text-sm text-zinc-500'>
                    Select a face crop to use as the profile thumbnail.
                </Text>

                <div className='flex items-start gap-6'>
                    {/* Current key photo preview */}
                    <div className='flex-shrink-0'>
                        {person.key_photo_face_id ? (
                            <img
                                src={getPersonKeyPhotoUrl(person.id, person.key_photo_face_id)}
                                alt={`${person.primary_name} key photo`}
                                className='h-24 w-24 rounded object-cover ring-2 ring-zinc-300 dark:ring-zinc-600'
                            />
                        ) : (
                            <div className='flex h-24 w-24 items-center justify-center rounded bg-zinc-200 text-sm text-zinc-400 dark:bg-zinc-800'>
                                None
                            </div>
                        )}
                        {person.key_photo_face_id && (
                            <Button
                                plain
                                className='mt-1 text-xs text-zinc-500'
                                onClick={() => handleSetKeyPhoto(null)}
                                disabled={keyPhotoLoading}
                            >
                                Clear
                            </Button>
                        )}
                    </div>

                    {/* Face grid */}
                    <div className='flex flex-wrap gap-2'>
                        {(person.faces ?? [])
                            .filter((f) => f.confirmed)
                            .map((face) => (
                                <div key={face.id} className='relative'>
                                    <button
                                        type='button'
                                        aria-label='Use this face as key photo'
                                        aria-pressed={person.key_photo_face_id === face.id}
                                        className={`block h-16 w-16 cursor-pointer overflow-hidden rounded ring-2 transition-all ${
                                            person.key_photo_face_id === face.id
                                                ? 'ring-blue-500'
                                                : 'ring-zinc-300 hover:ring-zinc-500 dark:ring-zinc-600'
                                        }`}
                                        disabled={keyPhotoLoading}
                                        onClick={() => handleSetKeyPhoto(face.id)}
                                    >
                                        <FaceThumbnail faceId={face.id} />
                                    </button>
                                    {person.key_photo_face_id === face.id && (
                                        <div className='absolute right-0.5 bottom-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-blue-500 text-[9px] text-white'>
                                            ✓
                                        </div>
                                    )}
                                </div>
                            ))}
                        {(person.faces ?? []).filter((f) => f.confirmed).length === 0 && (
                            <p className='text-sm text-zinc-400'>No confirmed faces yet.</p>
                        )}
                    </div>
                </div>
                {keyPhotoError && <p className='mt-2 text-xs text-red-600'>{keyPhotoError}</p>}
            </div>

            <div>
                <Heading level={2}>Aliases</Heading>
                <Text className='mt-1 mb-4 text-sm text-zinc-500'>Alternate names this person may be known by.</Text>

                {aliases.length === 0 ? (
                    <p className='mb-4 text-sm text-zinc-400'>No aliases yet.</p>
                ) : (
                    <ul className='mb-4 space-y-2'>
                        {aliases.map((alias) => (
                            <li
                                key={alias.id}
                                className='flex items-center justify-between rounded-md border border-zinc-200 px-3 py-2 dark:border-zinc-700'
                            >
                                <span className='text-sm'>{alias.name}</span>
                                <Button plain className='text-sm text-red-600' onClick={() => handleDeleteAlias(alias)}>
                                    Remove
                                </Button>
                            </li>
                        ))}
                    </ul>
                )}

                <div className='flex items-center gap-3'>
                    <Input
                        placeholder='New alias'
                        value={newAlias}
                        onChange={(e) => setNewAlias(e.target.value)}
                        onKeyDown={(e) => e.key === 'Enter' && handleAddAlias()}
                        className='max-w-xs'
                    />
                    <Button onClick={handleAddAlias} disabled={addingAlias || !newAlias.trim()}>
                        {addingAlias ? 'Adding…' : 'Add Alias'}
                    </Button>
                </div>
                {aliasError && <p className='mt-1 text-xs text-red-600'>{aliasError}</p>}
            </div>
        </PageContentBlock>
    );
};

export default PersonAdminView;
