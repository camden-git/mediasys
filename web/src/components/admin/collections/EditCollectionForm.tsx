import React, { useEffect, useState } from 'react';
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../../elements/Dialog';
import { Button } from '../../elements/Button';
import { Field, Fieldset, FieldGroup, Label, Description, Legend } from '../../elements/Fieldset';
import { RadioGroup, RadioField, Radio } from '../../elements/Radio';
import { Input } from '../../elements/Input';
import { Textarea } from '../../elements/Textarea';
import { CheckboxField, Checkbox } from '../../elements/Checkbox';
import SortOrderListbox from '../shared/SortOrderListbox';
import {
    updateCollection,
    setCollectionFilters,
    addCollectionBanner,
    deleteCollectionBanner,
    reorderCollectionBanners,
    setCollectionInheritBanners,
    AdminCollectionResponse,
} from '../../../api/admin/collections';
import { CollectionBanner } from '../../../types';
import { queryClient } from '../../../lib/queryClient';
import { queryKeys } from '../../../lib/queryKeys';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender';
import CollectionFiltersEditor from './CollectionFiltersEditor';
import { BannerManager } from '../shared/BannerManager';

interface EditCollectionFormProps {
    isOpen: boolean;
    onClose: () => void;
    onUpdated: () => void;
    collection?: AdminCollectionResponse;
}

const invalidateCollections = () => void queryClient.invalidateQueries({ queryKey: queryKeys.collections.all() });

const EditCollectionForm: React.FC<EditCollectionFormProps> = ({ isOpen, onClose, onUpdated, collection }) => {
    const [name, setName] = useState('');
    const [slug, setSlug] = useState('');
    const [description, setDescription] = useState('');
    const [isPublic, setIsPublic] = useState(true);
    const [filterMatch, setFilterMatch] = useState<'all' | 'any'>('all');
    const [sortOrder, setSortOrder] = useState('filename_asc');
    const [filters, setFilters] = useState<Array<{ tag_key: string; tag_value: string; negate: boolean }>>([]);
    const [isLoading, setIsLoading] = useState(false);
    const [inheritBanners, setInheritBanners] = useState(false);
    const [banners, setBanners] = useState<CollectionBanner[]>([]);
    const { clearFlashes, clearAndAddHttpError, addFlash } = useFlash();

    useEffect(() => {
        if (collection) {
            setName(collection.name);
            setSlug(collection.slug);
            setDescription(collection.description ?? '');
            setIsPublic(collection.is_public);
            setFilterMatch((collection.filter_match as 'all' | 'any') ?? 'all');
            setSortOrder(collection.sort_order ?? 'filename_asc');
            setFilters(
                collection.filters?.map((f) => ({ tag_key: f.tag_key, tag_value: f.tag_value, negate: f.negate })) ??
                    [],
            );
            setInheritBanners(collection.inherit_banners_from_albums ?? false);
            setBanners(collection.banners ?? []);
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
                filter_match: filterMatch,
                sort_order: sortOrder,
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

    const handleInheritToggle = async (checked: boolean) => {
        if (!collection) return;
        setInheritBanners(checked);
        try {
            await setCollectionInheritBanners(collection.id, checked);
            invalidateCollections();
        } catch {
            setInheritBanners(!checked);
            addFlash({ key: 'edit-collection', type: 'error', message: 'Failed to update inherit setting.' });
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
                            <Label>Sort Order</Label>
                            <SortOrderListbox value={sortOrder} onChange={(val) => setSortOrder(val)} />
                            <Description>How images in this collection should be sorted</Description>
                        </Field>

                        <Field>
                            <Label>Tag Filters</Label>
                            <CollectionFiltersEditor filters={filters} onChange={setFilters} />
                        </Field>

                        {/* Banner management */}
                        <Field>
                            <Label>Banners</Label>
                            <div className='mt-2 space-y-3'>
                                <CheckboxField>
                                    <Checkbox
                                        id='edit-inherit-banners'
                                        checked={inheritBanners}
                                        onChange={handleInheritToggle}
                                    />
                                    <Label htmlFor='edit-inherit-banners'>
                                        Inherit banners from contributing albums
                                    </Label>
                                </CheckboxField>

                                {!inheritBanners && (
                                    <BannerManager
                                        banners={banners}
                                        onAdd={async (file) => {
                                            if (!collection) return;
                                            try {
                                                const newBanner = await addCollectionBanner(collection.id, file);
                                                setBanners((prev) => [...prev, newBanner]);
                                                invalidateCollections();
                                            } catch {
                                                addFlash({
                                                    key: 'edit-collection',
                                                    type: 'error',
                                                    message: 'Failed to upload banner.',
                                                });
                                            }
                                        }}
                                        onDelete={async (bannerId) => {
                                            if (!collection) return;
                                            try {
                                                await deleteCollectionBanner(collection.id, bannerId);
                                                setBanners((prev) => prev.filter((b) => b.id !== bannerId));
                                                invalidateCollections();
                                            } catch {
                                                addFlash({
                                                    key: 'edit-collection',
                                                    type: 'error',
                                                    message: 'Failed to remove banner.',
                                                });
                                            }
                                        }}
                                        onReorder={async (ids) => {
                                            if (!collection) return;
                                            const newOrder = ids
                                                .map((id) => banners.find((b) => b.id === id))
                                                .filter((b): b is CollectionBanner => b !== undefined);
                                            setBanners(newOrder);
                                            try {
                                                await reorderCollectionBanners(collection.id, ids);
                                                invalidateCollections();
                                            } catch {
                                                addFlash({
                                                    key: 'edit-collection',
                                                    type: 'error',
                                                    message: 'Failed to reorder banners.',
                                                });
                                            }
                                        }}
                                    />
                                )}

                                {inheritBanners && (
                                    <p className='text-xs text-zinc-500 dark:text-zinc-400'>
                                        Banners will be pulled from albums that contribute images to this collection.
                                    </p>
                                )}
                            </div>
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
