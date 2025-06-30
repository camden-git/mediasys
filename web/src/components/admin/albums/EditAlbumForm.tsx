import React, { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Formik, Form } from 'formik';
import * as Yup from 'yup';
import { useStoreActions, useStoreState } from '../../../store/hooks';
import { UpdateAlbumPayload } from '../../../api/admin/albums';
import { Button } from '../../elements/Button';
import { Input } from '../../elements/Input';
import { Field, FieldGroup, Label, Description, ErrorMessage } from '../../elements/Fieldset';
import { Checkbox } from '../../elements/Checkbox';
import { Select } from '../../elements/Select';
import { PhotoIcon } from '@heroicons/react/24/solid';

const validationSchema = Yup.object({
    name: Yup.string().required('Name is required').min(1, 'Name must be at least 1 character'),
    description: Yup.string().optional(),
    location: Yup.string().optional(),
    sort_order: Yup.string().optional(),
    is_hidden: Yup.boolean().optional(),
});

const allowedTypes: readonly string[] = ['image/png', 'image/jpeg', 'image/webp', 'image/svg+xml'] as const;

const EditAlbumForm: React.FC = () => {
    const navigate = useNavigate();
    const album = useStoreState((state) => state.albumContext.data!);
    const albumId = useStoreState((state) => state.albumContext.data!.id);
    const { updateAlbum, uploadAlbumBanner } = useStoreActions((actions) => actions.adminAlbums);
    const { addFlash } = useStoreActions((actions) => actions.ui);

    // Banner upload state
    const [bannerFile, setBannerFile] = useState<File | null>(null);
    const [bannerBlobUrl, setBannerBlobUrl] = useState<string | null>(null);
    const [bannerUploading, setBannerUploading] = useState(false);
    const [bannerError, setBannerError] = useState('');
    const bannerInputRef = useRef<HTMLInputElement>(null);

    const initialValues: UpdateAlbumPayload = {
        name: album.name,
        description: album.description || '',
        location: album.location || '',
        sort_order: album.sort_order,
        is_hidden: album.is_hidden,
    };

    // Handle banner file selection
    const handleBannerDrop: React.DragEventHandler<HTMLButtonElement> = (event) => {
        event.preventDefault();
        if (event.dataTransfer.files.length > 0) {
            const file = event.dataTransfer.files[0];
            if (!allowedTypes.includes(file.type)) {
                setBannerError('Invalid file type. Please select a PNG, JPEG, WebP, or SVG file.');
                return;
            }
            setBannerFile(file);
            setBannerError('');
        }
    };

    // Handle banner file input change
    const handleBannerFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        if (event.target.files && event.target.files.length > 0) {
            const file = event.target.files[0];
            if (!allowedTypes.includes(file.type)) {
                setBannerError('Invalid file type. Please select a PNG, JPEG, WebP, or SVG file.');
                return;
            }
            setBannerFile(file);
            setBannerError('');
        }
    };

    // Handle banner upload
    const handleBannerUpload = () => {
        if (!bannerFile) return;
        setBannerUploading(true);
        setBannerError('');

        uploadAlbumBanner({
            id: albumId,
            file: bannerFile,
            addFlash,
            onSuccess: () => {
                setBannerFile(null);
                setBannerBlobUrl(null);
                setBannerUploading(false);
            },
        });
    };

    // Cleanup blob URL when component unmounts or file changes
    useEffect(() => {
        if (bannerBlobUrl) {
            URL.revokeObjectURL(bannerBlobUrl);
        }
        if (!bannerFile) {
            setBannerBlobUrl(null);
            return;
        }
        setBannerBlobUrl(URL.createObjectURL(bannerFile));
    }, [bannerFile]);

    return (
        <div className='mx-auto max-w-2xl'>
            <div className='mb-6'>
                <h1 className='text-2xl font-bold text-gray-900'>Edit Album</h1>
                <p className='text-gray-600'>Update album information and settings.</p>
            </div>

            <Formik
                initialValues={initialValues}
                validationSchema={validationSchema}
                enableReinitialize={true}
                onSubmit={async (values, { setSubmitting }) => {
                    updateAlbum({
                        id: albumId,
                        payload: values,
                        addFlash,
                        onSuccess: () => {
                            navigate(`/admin/albums/${albumId}`);
                        },
                    });
                    setSubmitting(false);
                }}
            >
                {({ values, handleChange, handleBlur, isSubmitting, errors, touched }) => (
                    <Form className='space-y-6'>
                        <FieldGroup>
                            <Field>
                                <Label htmlFor='name'>Name *</Label>
                                <Input
                                    id='name'
                                    name='name'
                                    value={values.name}
                                    onChange={handleChange}
                                    onBlur={handleBlur}
                                    placeholder='Enter album name'
                                />
                                {touched.name && errors.name && <ErrorMessage>{errors.name}</ErrorMessage>}
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
                                <Label htmlFor='sort_order'>Sort Order</Label>
                                <Select
                                    id='sort_order'
                                    name='sort_order'
                                    value={values.sort_order}
                                    onChange={handleChange}
                                    onBlur={handleBlur}
                                >
                                    <option value='name'>By Name</option>
                                    <option value='date'>By Date</option>
                                    <option value='size'>By Size</option>
                                    <option value='random'>Random</option>
                                </Select>
                                <Description>How images in this album should be sorted</Description>
                            </Field>

                            <Field>
                                <div className='flex items-center space-x-2'>
                                    <Checkbox
                                        id='is_hidden'
                                        name='is_hidden'
                                        checked={values.is_hidden}
                                        onChange={handleChange}
                                        onBlur={handleBlur}
                                    />
                                    <Label htmlFor='is_hidden'>Hide this album from regular users</Label>
                                </div>
                                <Description>Hidden albums are not visible to regular users</Description>
                            </Field>
                        </FieldGroup>

                        <div className='flex justify-end space-x-3'>
                            <Button type='button' outline onClick={() => navigate(`/admin/albums/${albumId}`)}>
                                Cancel
                            </Button>
                            <Button type='submit' disabled={isSubmitting}>
                                {isSubmitting ? 'Updating...' : 'Update Album'}
                            </Button>
                        </div>
                    </Form>
                )}
            </Formik>

            {/* Banner Upload Section */}
            <div className='mt-12 border-t border-gray-200 pt-8'>
                <div className='mb-6'>
                    <h2 className='text-xl font-semibold text-gray-900'>Album Banner</h2>
                    <p className='text-gray-600'>Upload a banner image for this album.</p>
                </div>

                {bannerError && (
                    <div className='mb-4 rounded-lg border border-red-200 bg-red-50 p-3'>
                        <p className='text-sm text-red-700'>{bannerError}</p>
                    </div>
                )}

                <input
                    type='file'
                    className='hidden'
                    aria-hidden
                    ref={bannerInputRef}
                    accept={allowedTypes.join(',')}
                    onChange={handleBannerFileChange}
                />

                <button
                    className='flex w-full flex-col items-center justify-center rounded-xl border-2 border-dashed border-gray-600 p-8 transition-opacity duration-100 disabled:opacity-50'
                    onDragOver={(e) => e.preventDefault()}
                    onDrop={handleBannerDrop}
                    onClick={() => bannerInputRef.current?.click()}
                    disabled={bannerUploading}
                >
                    {bannerBlobUrl ? (
                        <img src={bannerBlobUrl} className='max-h-48 max-w-full rounded-lg' alt='Banner preview' />
                    ) : (
                        <>
                            <PhotoIcon className='size-16 text-gray-500' />
                            <p className='text-gray-500'>Drag and drop a banner image here</p>
                            <p className='text-xs text-gray-500'>or click to select a file</p>
                            <p className='mt-2 text-xs text-gray-500'>Supported formats: .jpg, .png, .webp, .svg</p>
                        </>
                    )}
                </button>

                <div className='mt-4 flex justify-end space-x-3'>
                    <Button
                        type='button'
                        outline
                        onClick={() => {
                            setBannerFile(null);
                            setBannerError('');
                        }}
                        disabled={bannerUploading || !bannerFile}
                    >
                        Clear
                    </Button>
                    <Button onClick={handleBannerUpload} disabled={bannerUploading || !bannerFile}>
                        {bannerUploading ? 'Uploading...' : 'Upload Banner'}
                    </Button>
                </div>
            </div>
        </div>
    );
};

export default EditAlbumForm;
