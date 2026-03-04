import { useCallback, useEffect, useState } from 'react';
import { AlbumDefaultTag } from '../../../../types';
import { useAlbumId } from '../../../../store/albumContextHooks';
import { getAlbumDefaultTags, setAlbumDefaultTags } from '../../../../api/admin/imageTags';
import { Button } from '../../../elements/Button';
import { Input } from '../../../elements/Input';
import { PlusIcon, TrashIcon } from '@heroicons/react/20/solid';
import FlashMessageRender from '../../../elements/FlashMessageRender';
import { useFlash } from '../../../../hooks/useFlash';
import HeaderedContent from '../../../elements/HeaderedContent';
import { DescriptionList, DescriptionTerm, DescriptionDetails } from '../../../elements/DescriptionList.tsx';
import { Field, FieldGroup, Label } from '../../../elements/Fieldset.tsx';

export function DefaultTagsSection() {
    const id = useAlbumId();

    const [tags, setTags] = useState<AlbumDefaultTag[]>([]);
    const [newKey, setNewKey] = useState('');
    const [newValue, setNewValue] = useState('');
    const [isSaving, setIsSaving] = useState(false);
    const { clearFlashes, clearAndAddHttpError, addFlash } = useFlash();

    const fetchTags = useCallback(async () => {
        if (!id) return;
        try {
            const data = await getAlbumDefaultTags(id);
            setTags(data);
        } catch {
            // silently ignore initial fetch errors
        }
    }, [id]);

    useEffect(() => {
        fetchTags();
    }, [fetchTags]);

    const addTag = () => {
        const k = newKey.trim();
        const v = newValue.trim();
        if (!k || !v) return;
        const already = tags.some((t) => t.tag_key === k && t.tag_value === v);
        if (!already) {
            setTags((prev) => [...prev, { id: 0, album_id: id, tag_key: k, tag_value: v, created_at: '' }]);
        }
        setNewKey('');
        setNewValue('');
    };

    const removeTag = (idx: number) => {
        setTags((prev) => prev.filter((_, i) => i !== idx));
    };

    const save = async () => {
        clearFlashes('default-tags');
        setIsSaving(true);
        try {
            const updated = await setAlbumDefaultTags(
                id,
                tags.map((t) => ({ tag_key: t.tag_key, tag_value: t.tag_value })),
            );
            setTags(updated);
            addFlash({ key: 'default-tags', type: 'success', title: 'Saved', message: 'Default tags updated.' });
        } catch (error: any) {
            clearAndAddHttpError({ error, key: 'default-tags' });
        } finally {
            setIsSaving(false);
        }
    };

    return (
        <HeaderedContent
            title='Default Tags'
            description='These tags are automatically applied to every new image added to this album.'
            className={'mt-16 pb-8'}
        >
            <FlashMessageRender byKey='default-tags' />

            {tags.length > 0 ? (
                <DescriptionList>
                    {tags.map((t, idx) => (
                        <>
                            <DescriptionTerm>{t.tag_key}</DescriptionTerm>
                            <DescriptionDetails>
                                {t.tag_value}
                                <Button type='button' plain onClick={() => removeTag(idx)} aria-label='Remove tag'>
                                    <TrashIcon className='size-4' />
                                </Button>
                            </DescriptionDetails>
                        </>
                    ))}
                </DescriptionList>
            ) : (
                <p className='text-sm text-zinc-400'>No default tags configured.</p>
            )}

            <FieldGroup>
                <div className='grid grid-cols-1 gap-8 sm:grid-cols-2 sm:gap-4'>
                    <Field>
                        <Label>Key</Label>
                        <Input
                            type='text'
                            placeholder='tag key (e.g. event)'
                            value={newKey}
                            onChange={(e) => setNewKey(e.target.value)}
                        />
                    </Field>
                    <Field>
                        <Label>Value</Label>
                        <Input
                            type='text'
                            placeholder='tag value (e.g. spring-gala)'
                            value={newValue}
                            onChange={(e) => setNewValue(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && addTag()}
                        />
                    </Field>
                </div>
                <Button type='button' onClick={addTag} plain>
                    <PlusIcon className='size-4' />
                </Button>
            </FieldGroup>

            <div className='mt-4 flex justify-end'>
                <Button onClick={save} disabled={isSaving}>
                    {isSaving ? 'Saving...' : 'Save Default Tags'}
                </Button>
            </div>
        </HeaderedContent>
    );
}
