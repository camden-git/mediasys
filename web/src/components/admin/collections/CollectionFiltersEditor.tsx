import React, { useState } from 'react';
import { Button } from '../../elements/Button';
import { Input } from '../../elements/Input';
import { PlusIcon, TrashIcon } from '@heroicons/react/20/solid';

interface CollectionFiltersEditorProps {
    filters: Array<{ tag_key: string; tag_value: string }>;
    onChange: (filters: Array<{ tag_key: string; tag_value: string }>) => void;
}

const CollectionFiltersEditor: React.FC<CollectionFiltersEditorProps> = ({ filters, onChange }) => {
    const [newKey, setNewKey] = useState('');
    const [newValue, setNewValue] = useState('');

    const addFilter = () => {
        const k = newKey.trim();
        const v = newValue.trim();
        if (!k || !v) return;
        const already = filters.some((f) => f.tag_key === k && f.tag_value === v);
        if (!already) {
            onChange([...filters, { tag_key: k, tag_value: v }]);
        }
        setNewKey('');
        setNewValue('');
    };

    const removeFilter = (idx: number) => {
        onChange(filters.filter((_, i) => i !== idx));
    };

    return (
        <div className='mt-3 space-y-3'>
            <p className='text-sm text-zinc-500 dark:text-zinc-400'>
                Collection matches images where <strong>all keys</strong> have at least one matching value (AND across
                keys, OR within same key).
            </p>

            {filters.length > 0 && (
                <div className='divide-y divide-zinc-200 rounded border border-zinc-200 dark:divide-zinc-700 dark:border-zinc-700'>
                    {filters.map((f, idx) => (
                        <div key={idx} className='flex items-center justify-between px-3 py-2 text-sm'>
                            <span>
                                <span className='font-mono text-zinc-500 dark:text-zinc-400'>{f.tag_key}</span>
                                <span className='mx-2 text-zinc-400'>=</span>
                                <span className='font-mono'>{f.tag_value}</span>
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
                <Button type='button' onClick={addFilter} plain>
                    <PlusIcon className='size-4' />
                </Button>
            </div>
        </div>
    );
};

export default CollectionFiltersEditor;
