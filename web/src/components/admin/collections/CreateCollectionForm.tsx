import React, { useState } from 'react';
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../../elements/Dialog';
import { Button } from '../../elements/Button';
import { Field, Fieldset, FieldGroup, Label, Description, Legend } from '../../elements/Fieldset';
import { RadioGroup, RadioField, Radio } from '../../elements/Radio';
import { Input } from '../../elements/Input';
import { Textarea } from '../../elements/Textarea';
import { CheckboxField, Checkbox } from '../../elements/Checkbox';
import { createCollection } from '../../../api/admin/collections';
import { setCollectionFilters } from '../../../api/admin/collections';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import CollectionFiltersEditor from './CollectionFiltersEditor';

interface CreateCollectionFormProps {
    isOpen: boolean;
    onClose: () => void;
    onCreated: () => void;
}

const CreateCollectionForm: React.FC<CreateCollectionFormProps> = ({ isOpen, onClose, onCreated }) => {
    const [name, setName] = useState('');
    const [slug, setSlug] = useState('');
    const [description, setDescription] = useState('');
    const [isPublic, setIsPublic] = useState(true);
    const [filterMatch, setFilterMatch] = useState<'all' | 'any'>('all');
    const [filters, setFilters] = useState<Array<{ tag_key: string; tag_value: string; negate: boolean }>>([]);
    const [isLoading, setIsLoading] = useState(false);
    // Creating and setting filters are two requests. Once the collection exists, a retry must only
    // redo the filters instead of creating a duplicate (which would fail on the slug).
    const [createdId, setCreatedId] = useState<number | null>(null);
    const { clearFlashes, clearAndAddHttpError, addFlash } = useFlash();

    const resetForm = () => {
        setName('');
        setSlug('');
        setDescription('');
        setFilterMatch('all');
        setFilters([]);
        setIsPublic(true);
        setCreatedId(null);
    };

    const handleClose = () => {
        clearFlashes('create-collection');
        if (createdId !== null) {
            // the collection exists even though the filters failed; let the list pick it up
            onCreated();
            resetForm();
        }
        onClose();
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        clearFlashes('create-collection');
        setIsLoading(true);
        let id = createdId;
        try {
            if (id === null) {
                const created = await createCollection({
                    name,
                    slug,
                    description: description || undefined,
                    is_public: isPublic,
                    filter_match: filterMatch,
                });
                id = created.id;
                setCreatedId(id);
            }
            if (filters.length > 0) {
                await setCollectionFilters(id, filters);
            }
            onCreated();
            onClose();
            resetForm();
        } catch (error: any) {
            clearAndAddHttpError({ error, key: 'create-collection' });
            if (id !== null) {
                addFlash({
                    key: 'create-collection',
                    type: 'error',
                    message: 'The collection was created, but its filters could not be saved. Submit again to retry.',
                });
            }
        } finally {
            setIsLoading(false);
        }
    };

    const detailsLocked = isLoading || createdId !== null;

    return (
        <Dialog open={isOpen} onClose={handleClose}>
            <form onSubmit={handleSubmit}>
                <DialogTitle>Create Collection</DialogTitle>
                <DialogDescription>
                    Collections are virtual, tag-filter-based groupings that work across all albums.
                </DialogDescription>
                <DialogBody>
                    <FlashMessageRender byKey='create-collection' className='mb-4' />
                    <FieldGroup>
                        <Field>
                            <Label>Name</Label>
                            <Input
                                type='text'
                                required
                                value={name}
                                onChange={(e) => setName(e.target.value)}
                                disabled={detailsLocked}
                            />
                        </Field>
                        <Field>
                            <Label>Slug</Label>
                            <Input
                                type='text'
                                required
                                value={slug}
                                onChange={(e) => setSlug(e.target.value)}
                                disabled={detailsLocked}
                                className='font-mono'
                            />
                        </Field>
                        <Field>
                            <Label>Description</Label>
                            <Textarea
                                value={description}
                                onChange={(e) => setDescription(e.target.value)}
                                rows={2}
                                disabled={isLoading}
                            />
                        </Field>
                        <CheckboxField>
                            <Checkbox
                                id='is-public'
                                checked={isPublic}
                                onChange={(checked) => setIsPublic(checked)}
                                disabled={isLoading}
                            />
                            <Label htmlFor='is-public'>Public (visible on the collections page)</Label>
                        </CheckboxField>
                        <Fieldset>
                            <Legend>Filter Mode</Legend>
                            <RadioGroup value={filterMatch} onChange={(v) => setFilterMatch(v as 'all' | 'any')}>
                                <RadioField>
                                    <Radio value='all' disabled={isLoading} />
                                    <Label>All (AND)</Label>
                                    <Description>Image must satisfy every inclusion group.</Description>
                                </RadioField>
                                <RadioField>
                                    <Radio value='any' disabled={isLoading} />
                                    <Label>Any (OR)</Label>
                                    <Description>Image must satisfy at least one inclusion group.</Description>
                                </RadioField>
                            </RadioGroup>
                        </Fieldset>
                        <Field>
                            <Label>Tag Filters</Label>
                            <CollectionFiltersEditor filters={filters} onChange={setFilters} />
                        </Field>
                    </FieldGroup>
                </DialogBody>
                <DialogActions>
                    <Button plain onClick={handleClose} type='button'>
                        Cancel
                    </Button>
                    <Button type='submit' disabled={isLoading}>
                        {isLoading ? 'Creating...' : createdId !== null ? 'Retry Saving Filters' : 'Create Collection'}
                    </Button>
                </DialogActions>
            </form>
        </Dialog>
    );
};

export default CreateCollectionForm;
