import React, { useEffect, useRef, useState } from 'react';
import { Button } from '../../../elements/Button';
import { Input } from '../../../elements/Input';
import { Combobox, ComboboxLabel, ComboboxOption } from '../../../elements/Combobox';
import { UntaggedFaceResult, Person } from '../../../../types';
import { tagFace, deleteFace, suggestFace } from '../../../../api/faces';
import { createPerson, searchPeople } from '../../../../api/people';
import { getPersonKeyPhotoUrl } from '../../../../api/media';
import { isAbortError } from '../../../../api/errors';
import { Avatar } from '../../../elements/Avatar';

type CreateOption = { __create: true; name: string };
type ComboOption = Person | CreateOption;

function isCreate(opt: ComboOption): opt is CreateOption {
    return (opt as CreateOption).__create === true;
}

interface FaceTaggingPanelProps {
    face: UntaggedFaceResult;
    onTagged: (faceId: number) => void;
    onDeleted: (faceId: number) => void;
    onSuggestionUpdated: (
        faceId: number,
        personId: number | null,
        personName: string | null,
        suggestionCount: number,
    ) => void;
}

const FaceTaggingPanel: React.FC<FaceTaggingPanelProps> = ({ face, onTagged, onDeleted, onSuggestionUpdated }) => {
    const [comboValue, setComboValue] = useState<ComboOption | null>(null);
    const [personOptions, setPersonOptions] = useState<Person[]>([]);
    const [currentQuery, setCurrentQuery] = useState('');
    const [newPersonName, setNewPersonName] = useState('');
    const [showNewPerson, setShowNewPerson] = useState(false);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const searchAbortRef = useRef<AbortController | null>(null);
    const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    const handleAcceptSuggestion = async () => {
        if (!face.suggested_person_id) return;
        setBusy(true);
        setError(null);
        try {
            await tagFace(face.face_id, face.suggested_person_id);
            onTagged(face.face_id);
        } catch (e: any) {
            setError(e.message);
            setBusy(false);
        }
    };

    useEffect(() => {
        const handleKey = (e: KeyboardEvent) => {
            // Cmd/Ctrl+Enter accepts the suggestion (Cmd+A stays "select all")
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey) && !e.shiftKey && !e.altKey && !e.repeat) {
                if (face.suggested_person_id && !busy) {
                    e.preventDefault();
                    handleAcceptSuggestion();
                }
            }
        };
        window.addEventListener('keydown', handleKey);
        return () => window.removeEventListener('keydown', handleKey);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [face.suggested_person_id, busy]);

    const handleQueryChange = (q: string) => {
        setCurrentQuery(q);
        if (debounceRef.current) clearTimeout(debounceRef.current);
        if (!q.trim()) {
            setPersonOptions([]);
            return;
        }
        debounceRef.current = setTimeout(() => {
            if (searchAbortRef.current) searchAbortRef.current.abort();
            const ctrl = new AbortController();
            searchAbortRef.current = ctrl;
            searchPeople(q, 5, ctrl.signal)
                .then((results) => setPersonOptions(results))
                .catch((e) => {
                    if (!isAbortError(e)) console.error('People search error:', e);
                });
        }, 100);
    };

    // Show a "Create" sentinel when there's a query but no results came back
    const comboOptions: ComboOption[] =
        currentQuery.trim() && personOptions.length === 0
            ? [{ __create: true, name: currentQuery.trim() }]
            : personOptions;

    const handleSelect = async (opt: ComboOption | null) => {
        if (!opt) return;
        setComboValue(opt);
        setBusy(true);
        setError(null);
        try {
            if (isCreate(opt)) {
                const person = await createPerson(opt.name);
                await tagFace(face.face_id, person.id);
            } else {
                await tagFace(face.face_id, opt.id);
            }
            onTagged(face.face_id);
        } catch (e: any) {
            setError(e.message);
            setComboValue(null);
            setBusy(false);
        }
    };

    const handleCreateAndTag = async () => {
        if (!newPersonName.trim()) return;
        setBusy(true);
        setError(null);
        try {
            const person = await createPerson(newPersonName.trim());
            await tagFace(face.face_id, person.id);
            onTagged(face.face_id);
        } catch (e: any) {
            setError(e.message);
            setBusy(false);
        }
    };

    const handleRecalculate = async () => {
        setBusy(true);
        setError(null);
        try {
            const result = await suggestFace(face.face_id);
            onSuggestionUpdated(
                face.face_id,
                result.suggested_person_id,
                result.suggested_person_name,
                result.suggestion_count,
            );
        } catch (e: any) {
            setError(e.message);
        } finally {
            setBusy(false);
        }
    };

    const handleDelete = async () => {
        if (!window.confirm('Delete this face? This cannot be undone.')) return;
        setBusy(true);
        setError(null);
        try {
            await deleteFace(face.face_id);
            onDeleted(face.face_id);
        } catch (e: any) {
            setError(e.message);
            setBusy(false);
        }
    };

    const confidencePct = Math.round(face.detection_confidence * 100);

    return (
        <div className='flex flex-col gap-4 p-5'>
            {/* Meta info */}
            <div>
                <p className='truncate text-xs text-zinc-500 dark:text-zinc-400' title={face.image_path}>
                    {face.image_path}
                </p>
                <p className='mt-1 text-xs text-zinc-500 dark:text-zinc-400'>
                    Confidence: {confidencePct}%
                    {face.quality_score != null && (
                        <span className='ml-2'>Quality: {Math.round(face.quality_score * 100)}%</span>
                    )}
                </p>
            </div>

            {error && <p className='text-xs text-red-600'>{error}</p>}

            {/* Accept suggestion */}
            {face.suggested_person_name && face.suggested_person_id && (
                <Button onClick={handleAcceptSuggestion} disabled={busy} className='w-full'>
                    Accept: {face.suggested_person_name}
                    {face.suggestion_count > 1 && (
                        <span className='ml-1 text-xs opacity-70'>({face.suggestion_count})</span>
                    )}
                </Button>
            )}

            {/* Tag existing person via search, or create new inline */}
            {!showNewPerson && (
                <Combobox
                    options={comboOptions}
                    value={comboValue}
                    onChange={handleSelect}
                    displayValue={(opt) => {
                        if (!opt) return '';
                        return isCreate(opt) ? opt.name : (opt as Person).primary_name;
                    }}
                    filter={() => true}
                    onInputChange={handleQueryChange}
                    placeholder='Search people…'
                    disabled={busy}
                    autoFocus
                >
                    {(opt) =>
                        isCreate(opt) ? (
                            <ComboboxOption value={opt}>
                                <ComboboxLabel>Create "{opt.name}"</ComboboxLabel>
                            </ComboboxOption>
                        ) : (
                            <ComboboxOption value={opt}>
                                <Avatar
                                    src={
                                        (opt as Person).key_photo_face_id
                                            ? getPersonKeyPhotoUrl(
                                                  (opt as Person).id,
                                                  (opt as Person).key_photo_face_id ?? 0,
                                              )
                                            : undefined
                                    }
                                    initials={(opt as Person).primary_name[0]?.toUpperCase()}
                                    className='rounded'
                                    style={{
                                        backgroundColor: `hsl(${((opt as Person).id * 137.508) % 360}, 55%, 40%)`,
                                    }}
                                    alt=''
                                />
                                <ComboboxLabel>{(opt as Person).primary_name}</ComboboxLabel>
                            </ComboboxOption>
                        )
                    }
                </Combobox>
            )}

            {/* New person form (manual fallback) */}
            {showNewPerson && (
                <div className='flex gap-2'>
                    <Input
                        placeholder='New person name'
                        value={newPersonName}
                        onChange={(e) => setNewPersonName(e.target.value)}
                        onKeyDown={(e) => e.key === 'Enter' && handleCreateAndTag()}
                        className='flex-1'
                        autoFocus
                        disabled={busy}
                    />
                    <Button onClick={handleCreateAndTag} disabled={busy || !newPersonName.trim()} plain>
                        Create &amp; Tag
                    </Button>
                </div>
            )}

            {/* Toggle / Recalculate / Delete row */}
            <div className='flex gap-2'>
                <Button plain onClick={() => setShowNewPerson((v) => !v)} className='flex-1 text-xs' disabled={busy}>
                    {showNewPerson ? 'Cancel' : 'New person…'}
                </Button>
                <Button
                    plain
                    onClick={handleRecalculate}
                    className='text-xs'
                    disabled={busy}
                    title='Recalculate suggestion using updated face embeddings'
                >
                    Recalculate
                </Button>
                <Button plain onClick={handleDelete} className='text-xs text-red-600' disabled={busy}>
                    Delete
                </Button>
            </div>
        </div>
    );
};

export default FaceTaggingPanel;
