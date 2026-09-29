import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Formik, Form } from 'formik';
import * as Yup from 'yup';
import { CreateAlbumPayload, createAlbum as createAlbumAPI } from '../../../api/admin/albums';
import { useUIStore } from '../../../store/useUIStore';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../lib/queryKeys';
import { Button } from '../../elements/Button';
import { Input } from '../../elements/Input';
import { Field, FieldGroup, Label, Description, ErrorMessage } from '../../elements/Fieldset';
import { Checkbox } from '../../elements/Checkbox';
import { Listbox, ListboxLabel, ListboxOption } from '../../elements/Listbox';

const validationSchema = Yup.object({
    name: Yup.string().required('Name is required').min(1, 'Name must be at least 1 character'),
    slug: Yup.string()
        .required('Slug is required')
        .matches(/^[a-z0-9-]+$/, 'Slug can only contain lowercase letters, numbers, and hyphens')
        .min(1, 'Slug must be at least 1 character'),
    folder_path: Yup.string().optional(),
    description: Yup.string().optional(),
    location: Yup.string().optional(),
    sort_order: Yup.string().optional(),
    is_hidden: Yup.boolean().optional(),
});

const CreateAlbumForm: React.FC = () => {
    const navigate = useNavigate();
    const addFlash = useUIStore((s) => s.addFlash);
    const queryClient = useQueryClient();

    const initialValues: CreateAlbumPayload = {
        name: '',
        slug: '',
        folder_path: '',
        description: '',
        location: '',
        sort_order: 'date_asc',
        is_hidden: false,
    };

    return (
        <div className='mx-auto max-w-2xl'>
            <div className='mb-6'>
                <h1 className='text-2xl font-bold text-gray-900'>Create Album</h1>
                <p className='text-gray-600'>Create a new album to organize your media files.</p>
            </div>

            <Formik
                initialValues={initialValues}
                validationSchema={validationSchema}
                onSubmit={async (values, { setSubmitting }) => {
                    try {
                        await createAlbumAPI(values);
                        queryClient.invalidateQueries({ queryKey: queryKeys.albums.all() });
                        addFlash({ key: 'album-created', type: 'success', title: 'Created', message: 'Album created successfully' });
                        navigate('/admin/albums');
                    } catch (err: any) {
                        addFlash({ key: 'album-created-error', type: 'error', title: 'Error', message: err.message || 'Failed to create album' });
                    } finally {
                        setSubmitting(false);
                    }
                }}
            >
                {({
                    values,
                    handleChange,
                    handleBlur,
                    setFieldValue,
                    setFieldTouched,
                    isSubmitting,
                    errors,
                    touched,
                }) => (
                    <Form className='space-y-6'>
                        <FieldGroup>
                            <Field>
                                <Label htmlFor='name'>Name</Label>
                                <Input
                                    id='name'
                                    name='name'
                                    value={values.name}
                                    onChange={(e) => {
                                        const name = e.target.value;
                                        setFieldValue('name', name);

                                        // auto-generate slug from name
                                        const slug = name
                                            .toLowerCase()
                                            .replace(/[^a-z0-9\s-]/g, '')
                                            .replace(/\s+/g, '-')
                                            .replace(/-+/g, '-')
                                            .trim();
                                        setFieldValue('slug', slug);
                                    }}
                                    onBlur={handleBlur}
                                    placeholder='Enter album name'
                                />
                                {touched.name && errors.name && <ErrorMessage>{errors.name}</ErrorMessage>}
                            </Field>

                            <Field>
                                <Label htmlFor='slug'>Slug</Label>
                                <Input
                                    id='slug'
                                    name='slug'
                                    value={values.slug}
                                    onChange={handleChange}
                                    onBlur={handleBlur}
                                    placeholder='album-slug'
                                />
                                <Description>URL-friendly identifier for the album</Description>
                                {touched.slug && errors.slug && <ErrorMessage>{errors.slug}</ErrorMessage>}
                            </Field>

                            <Field>
                                <Label htmlFor='folder_path'>Storage Folder</Label>
                                <Input
                                    id='folder_path'
                                    name='folder_path'
                                    value={values.folder_path}
                                    onChange={handleChange}
                                    onBlur={handleBlur}
                                    placeholder='photos/2024/summer'
                                />
                                <Description>
                                    Optional storage prefix for this album's files. Defaults to the slug.
                                </Description>
                                {touched.folder_path && errors.folder_path && (
                                    <ErrorMessage>{errors.folder_path}</ErrorMessage>
                                )}
                            </Field>

                            <Field>
                                <Label htmlFor='description'>Description</Label>
                                <Input
                                    id='description'
                                    name='description'
                                    value={values.description}
                                    onChange={handleChange}
                                    onBlur={handleBlur}
                                    placeholder='Optional description of the album'
                                />
                            </Field>

                            <Field>
                                <Label htmlFor='location'>Location</Label>
                                <Input
                                    id='location'
                                    name='location'
                                    value={values.location}
                                    onChange={handleChange}
                                    onBlur={handleBlur}
                                    placeholder='e.g., Paris, France'
                                />
                                <Description>Optional location information for the album</Description>
                            </Field>

                            <Field>
                                <Label>Sort Order</Label>
                                <Listbox value={values.sort_order} onChange={(val) => setFieldValue('sort_order', val)}>
                                    <ListboxOption value='filename_asc'>
                                        <ListboxLabel>Filename (A–Z)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='filename_desc'>
                                        <ListboxLabel>Filename (Z–A)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='filename_nat'>
                                        <ListboxLabel>Filename (Natural)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='date_asc'>
                                        <ListboxLabel>Capture Date (Oldest First)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='date_desc'>
                                        <ListboxLabel>Capture Date (Newest First)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='mod_time_desc'>
                                        <ListboxLabel>Modified Time (Newest First)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='mod_time_asc'>
                                        <ListboxLabel>Modified Time (Oldest First)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='file_size_desc'>
                                        <ListboxLabel>File Size (Largest First)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='file_size_asc'>
                                        <ListboxLabel>File Size (Smallest First)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='iso_asc'>
                                        <ListboxLabel>ISO (Low to High)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='iso_desc'>
                                        <ListboxLabel>ISO (High to Low)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='aperture_asc'>
                                        <ListboxLabel>Aperture (Small to Large)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='aperture_desc'>
                                        <ListboxLabel>Aperture (Large to Small)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='shutter_speed_desc'>
                                        <ListboxLabel>Shutter Speed (Fast to Slow)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='shutter_speed_asc'>
                                        <ListboxLabel>Shutter Speed (Slow to Fast)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='focal_length_asc'>
                                        <ListboxLabel>Focal Length (Short to Long)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='focal_length_desc'>
                                        <ListboxLabel>Focal Length (Long to Short)</ListboxLabel>
                                    </ListboxOption>
                                    <ListboxOption value='camera_asc'>
                                        <ListboxLabel>Camera (A–Z)</ListboxLabel>
                                    </ListboxOption>
                                </Listbox>
                                <Description>How images in this album should be sorted</Description>
                            </Field>

                            <Field>
                                <div className='flex items-center space-x-2'>
                                    <Checkbox
                                        id='is_hidden'
                                        checked={values.is_hidden}
                                        onChange={(checked: boolean) => setFieldValue('is_hidden', checked)}
                                        onBlur={() => setFieldTouched('is_hidden', true)}
                                    />
                                    <Label htmlFor='is_hidden'>Hide this album from regular users</Label>
                                </div>
                                <Description>Hidden albums are not visible to regular users</Description>
                            </Field>
                        </FieldGroup>

                        <div className='flex justify-end space-x-3'>
                            <Button type='button' plain onClick={() => navigate('/admin/albums')}>
                                Cancel
                            </Button>
                            <Button type='submit' disabled={isSubmitting}>
                                {isSubmitting ? 'Creating...' : 'Create Album'}
                            </Button>
                        </div>
                    </Form>
                )}
            </Formik>
        </div>
    );
};

export default CreateAlbumForm;
