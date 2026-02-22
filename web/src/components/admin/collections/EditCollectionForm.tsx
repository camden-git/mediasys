import React, { useEffect, useState } from 'react';
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../../elements/Dialog';
import { Button } from '../../elements/Button';
import { Field, FieldGroup, Label } from '../../elements/Fieldset';
import { Input } from '../../elements/Input';
import { Textarea } from '../../elements/Textarea';
import { CheckboxField, Checkbox } from '../../elements/Checkbox';
import { updateCollection, setCollectionFilters } from '../../../api/admin/collections';
import { Collection } from '../../../types';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import CollectionFiltersEditor from './CollectionFiltersEditor';

interface EditCollectionFormProps {
    isOpen: boolean;
    onClose: () => void;
    onUpdated: () => void;
    collection?: Collection;
}

const EditCollectionForm: React.FC<EditCollectionFormProps> = ({ isOpen, onClose, onUpdated, collection }) => {
    const [name, setName] = useState('');
    const [slug, setSlug] = useState('');
    const [description, setDescription] = useState('');
    const [isPublic, setIsPublic] = useState(true);
    const [filters, setFilters] = useState<Array<{ tag_key: string; tag_value: string }>>([]);
    const [isLoading, setIsLoading] = useState(false);
    const { clearFlashes, clearAndAddHttpError } = useFlash();

    useEffect(() => {
        if (collection) {
            setName(collection.name);
            setSlug(collection.slug);
            setDescription(collection.description ?? '');
            setIsPublic(collection.is_public);
            setFilters(collection.filters?.map((f) => ({ tag_key: f.tag_key, tag_value: f.tag_value })) ?? []);
        }
    }, [collection]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!collection) return;
        clearFlashes('edit-collection');
        setIsLoading(true);
        try {
            await updateCollection(collection.id, {
                name,
                slug,
                description: description || undefined,
                is_public: isPublic,
            });
            await setCollectionFilters(collection.id, filters);
            onUpdated();
            onClose();
        } catch (error: any) {
            clearAndAddHttpError({ error, key: 'edit-collection' });
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <Dialog open={isOpen} onClose={onClose}>
            <form onSubmit={handleSubmit}>
                <DialogTitle>Edit Collection</DialogTitle>
                <DialogDescription>Update this collection&apos;s details and tag filters.</DialogDescription>
                <DialogBody>
                    <FlashMessageRender byKey='edit-collection' className='mb-4' />
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
                                id='edit-is-public'
                                checked={isPublic}
                                onChange={(checked) => setIsPublic(checked)}
                                disabled={isLoading}
                            />
                            <Label htmlFor='edit-is-public'>Public (visible on the collections page)</Label>
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
                        {isLoading ? 'Saving...' : 'Save Changes'}
                    </Button>
                </DialogActions>
            </form>
        </Dialog>
    );
};

export default EditCollectionForm;
