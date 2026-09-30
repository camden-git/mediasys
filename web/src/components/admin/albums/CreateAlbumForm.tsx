import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Formik, Form } from 'formik';
import * as Yup from 'yup';
import { CreateAlbumPayload, createAlbum as createAlbumAPI } from '../../../api/admin/albums';
import { useUIStore } from '../../../store/useUIStore';
import { invalidateAlbums } from '../../../lib/queryClient';
import FlashMessageRender from '../../elements/FlashMessageRender';
import { Button } from '../../elements/Button';
import { Input } from '../../elements/Input';
import { Field, FieldGroup, Label, Description, ErrorMessage } from '../../elements/Fieldset';
import { Checkbox } from '../../elements/Checkbox';
import SortOrderListbox from '../shared/SortOrderListbox';

const validationSchema = Yup.object({
    name: Yup.string().required('Name is required').min(1, 'Name must be at least 1 character'),
    slug: Yup.string()
        .required('Slug is required')
        .matches(
            /^[a-z0-9]+(-[a-z0-9]+)*$/,
            'Slug can only contain lowercase letters, numbers, and single hyphens between them',
        )
        // an all-digit slug would be ambiguous with a numeric album id
        .test('not-numeric', 'Slug cannot be only digits', (v) => !v || !/^\d+$/.test(v))
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
            <FlashMessageRender byKey='album-create-error' />
            <div className='mb-6'>
                <h1 className='text-2xl font-bold text-gray-900'>Create Album</h1>
                <p className='text-gray-600'>Create a new album to organize your media files.</p>
            </div>

            <Formik
                initialValues={initialValues}
                validationSchema={validationSchema}
                onSubmit={async (values, { setSubmitting, setFieldError, setFieldTouched }) => {
                    try {
                        await createAlbumAPI(values);
                        void invalidateAlbums();
                        addFlash({
                            key: 'album-created',
                            type: 'success',
                            title: 'Created',
                            message: 'Album created successfully',
                        });
                        navigate('/admin/albums');
                    } catch (err: any) {
                        if (err?.status === 409) {
                            setFieldTouched('slug', true, false);
                            setFieldError('slug', 'An album with this slug already exists.');
                            setSubmitting(false);
                            return;
                        }
                        addFlash({
                            key: 'album-create-error',
                            type: 'error',
                            title: 'Error',
                            message: err.message || 'Failed to create album',
                        });
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
                                            .replace(/^-+|-+$/g, '');
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
                                <SortOrderListbox
                                    value={values.sort_order}
                                    onChange={(val) => setFieldValue('sort_order', val)}
                                />
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
