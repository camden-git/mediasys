import React from 'react';
import { Listbox, ListboxLabel, ListboxOption } from '../../elements/Listbox';

const SORT_ORDER_OPTIONS = [
    { value: 'filename_asc', label: 'Filename (A–Z)' },
    { value: 'filename_desc', label: 'Filename (Z–A)' },
    { value: 'filename_nat', label: 'Filename (Natural)' },
    { value: 'date_asc', label: 'Capture Date (Oldest First)' },
    { value: 'date_desc', label: 'Capture Date (Newest First)' },
    { value: 'mod_time_desc', label: 'Modified Time (Newest First)' },
    { value: 'mod_time_asc', label: 'Modified Time (Oldest First)' },
    { value: 'file_size_desc', label: 'File Size (Largest First)' },
    { value: 'file_size_asc', label: 'File Size (Smallest First)' },
    { value: 'iso_asc', label: 'ISO (Low to High)' },
    { value: 'iso_desc', label: 'ISO (High to Low)' },
    { value: 'aperture_asc', label: 'Aperture (Small to Large)' },
    { value: 'aperture_desc', label: 'Aperture (Large to Small)' },
    { value: 'shutter_speed_desc', label: 'Shutter Speed (Fast to Slow)' },
    { value: 'shutter_speed_asc', label: 'Shutter Speed (Slow to Fast)' },
    { value: 'focal_length_asc', label: 'Focal Length (Short to Long)' },
    { value: 'focal_length_desc', label: 'Focal Length (Long to Short)' },
    { value: 'camera_asc', label: 'Camera (A–Z)' },
];

interface SortOrderListboxProps {
    value?: string;
    onChange: (value: string) => void;
}

const SortOrderListbox: React.FC<SortOrderListboxProps> = ({ value, onChange }) => (
    <Listbox value={value} onChange={onChange}>
        {SORT_ORDER_OPTIONS.map((option) => (
            <ListboxOption key={option.value} value={option.value}>
                <ListboxLabel>{option.label}</ListboxLabel>
            </ListboxOption>
        ))}
    </Listbox>
);

export default SortOrderListbox;
