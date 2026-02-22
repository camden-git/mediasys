import React, { useState } from 'react';
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../../elements/Dialog';
import { Button } from '../../elements/Button';
import { Field, FieldGroup, Label } from '../../elements/Fieldset';
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
    const [filters, setFilters] = useState<Array<{ tag_key: string; tag_value: string }>>([]);
    const [isLoading, setIsLoading] = useState(false);
    const { clearFlashes, clearAndAddHttpError } = useFlash();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        clearFlashes('create-collection');
        setIsLoading(true);
        try {
            const created = await createCollection({
                name,
                slug,
                description: description || undefined,
                is_public: isPublic,
            });
            if (filters.length > 0) {
                await setCollectionFilters(created.id, filters);
            }
            onCreated();
            onClose();
            setName('');
            setSlug('');
            setDescription('');
            setFilters([]);
            setIsPublic(true);
        } catch (error: any) {
            clearAndAddHttpError({ error, key: 'create-collection' });
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <Dialog open={isOpen} onClose={onClose}>
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
                                disabled={isLoading}
                            />
                        </Field>
                        <Field>
                            <Label>Slug</Label>
                            <Input
                                type='text'
                                required
                                value={slug}
                                onChange={(e) => setSlug(e.target.value)}
                                disabled={isLoading}
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
                        <Field>
                            <Label>Tag Filters</Label>
                            <CollectionFiltersEditor filters={filters} onChange={setFilters} />
                        </Field>
                    </FieldGroup>
                </DialogBody>
                <DialogActions>
                    <Button plain onClick={onClose} type='button'>
                        Cancel
                    </Button>
                    <Button type='submit' disabled={isLoading}>
                        {isLoading ? 'Creating...' : 'Create Collection'}
                    </Button>
                </DialogActions>
            </form>
        </Dialog>
    );
};

export default CreateCollectionForm;
