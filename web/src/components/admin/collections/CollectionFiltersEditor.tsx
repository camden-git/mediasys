import React, { useState } from 'react';
import { Button } from '../../elements/Button';
import { Input } from '../../elements/Input';
import { PlusIcon, TrashIcon } from '@heroicons/react/20/solid';

interface CollectionFiltersEditorProps {
    filters: Array<{ tag_key: string; tag_value: string; negate: boolean }>;
    onChange: (filters: Array<{ tag_key: string; tag_value: string; negate: boolean }>) => void;
}

const CollectionFiltersEditor: React.FC<CollectionFiltersEditorProps> = ({ filters, onChange }) => {
    const [newKey, setNewKey] = useState('');
    const [newValue, setNewValue] = useState('');
    const [newNegate, setNewNegate] = useState(false);

    const addFilter = () => {
        const k = newKey.trim();
        const v = newValue.trim();
        if (!k || !v) return;
        const already = filters.some((f) => f.tag_key === k && f.tag_value === v && f.negate === newNegate);
        if (!already) {
            onChange([...filters, { tag_key: k, tag_value: v, negate: newNegate }]);
        }
        setNewKey('');
        setNewValue('');
        setNewNegate(false);
    };

    const removeFilter = (idx: number) => {
        onChange(filters.filter((_, i) => i !== idx));
    };

    return (
        <div className='mt-3 space-y-3'>
            <p className='text-sm text-zinc-500 dark:text-zinc-400'>
                Inclusion rules match images with the given tag. Exclusion rules remove images that have the tag. Use{' '}
                <strong>Filter Mode</strong> above to control whether images must match <strong>all</strong> inclusion
                groups or <strong>any</strong> one.
            </p>

            {filters.length > 0 && (
                <div className='divide-y divide-zinc-200 rounded border border-zinc-200 dark:divide-zinc-700 dark:border-zinc-700'>
                    {filters.map((f, idx) => (
                        <div key={idx} className='flex items-center justify-between px-3 py-2 text-sm'>
                            <span className='flex items-center gap-1.5'>
                                <span className='font-mono text-zinc-500 dark:text-zinc-400'>{f.tag_key}</span>
                                {f.negate ? (
                                    <span className='mx-1 font-semibold text-red-500 dark:text-red-400'>≠</span>
                                ) : (
                                    <span className='mx-1 text-zinc-400'>=</span>
                                )}
                                <span className='font-mono'>{f.tag_value}</span>
                                {f.negate && (
                                    <span className='ml-1 rounded bg-red-100 px-1.5 py-0.5 text-xs font-medium text-red-600 dark:bg-red-900/30 dark:text-red-400'>
                                        exclude
                                    </span>
                                )}
                            </span>
                            <Button
                                type='button'
                                plain
                                onClick={() => removeFilter(idx)}
                                aria-label='Remove filter'
                                className='text-red-500 data-hover:text-red-700 dark:text-red-400 dark:data-hover:text-red-300'
                            >
                                <TrashIcon className='size-4' />
                            </Button>
                        </div>
                    ))}
                </div>
            )}

            <div className='flex gap-2'>
                <Input type='text' placeholder='tag key' value={newKey} onChange={(e) => setNewKey(e.target.value)} />
                <Input
                    type='text'
                    placeholder='tag value'
                    value={newValue}
                    onChange={(e) => setNewValue(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && addFilter()}
                />
                <button
                    type='button'
                    onClick={() => setNewNegate((n) => !n)}
                    title={newNegate ? 'Exclude (click to switch to include)' : 'Include (click to switch to exclude)'}
                    className={`shrink-0 rounded border px-2 py-1 text-xs font-medium transition-colors ${
                        newNegate
                            ? 'border-red-400 bg-red-50 text-red-600 dark:border-red-600 dark:bg-red-900/20 dark:text-red-400'
                            : 'border-zinc-300 bg-white text-zinc-600 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-300'
                    }`}
                >
                    {newNegate ? '≠ excl' : '= incl'}
                </button>
                <Button type='button' onClick={addFilter} plain>
                    <PlusIcon className='size-4' />
                </Button>
            </div>
        </div>
    );
};

export default CollectionFiltersEditor;
