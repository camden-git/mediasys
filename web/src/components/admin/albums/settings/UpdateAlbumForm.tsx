import { Formik, Form } from 'formik';
import * as Yup from 'yup';
import { useAlbumContextStore } from '../../../../store/useAlbumContextStore';
import { useUIStore } from '../../../../store/useUIStore';
import { UpdateAlbumPayload, updateAlbum as updateAlbumAPI } from '../../../../api/admin/albums';
import { Field, FieldGroup, Label, Description } from '../../../elements/Fieldset';
import { Checkbox } from '../../../elements/Checkbox';
import { Listbox, ListboxLabel, ListboxOption } from '../../../elements/Listbox';
import HeaderedContent from '../../../elements/HeaderedContent.tsx';
import FormikFieldComponent from '../../../elements/FormikField';
import FlashMessageRender from '../../../elements/FlashMessageRender';
import { UnsavedChangesBar } from '../../../elements/UnsavedChangesBar';

const validationSchema = Yup.object({
    name: Yup.string().required('Name is required').min(1, 'Name must be at least 1 character'),
    description: Yup.string().optional(),
    location: Yup.string().optional(),
    sort_order: Yup.string().optional(),
    is_hidden: Yup.boolean().optional(),
});

export function UpdateAlbumForm() {
    const album = useAlbumContextStore((s) => s.data!);
    const albumId = useAlbumContextStore((s) => s.data!.id);
    const addFlash = useUIStore((s) => s.addFlash);
    const setAlbum = useAlbumContextStore((s) => s.setAlbum);

    const initialValues: UpdateAlbumPayload = {
        name: album.name,
        description: album.description || '',
        location: album.location || '',
        sort_order: album.sort_order,
        is_hidden: album.is_hidden,
    };

    return (
        <>
            <FlashMessageRender byKey='album-update' />
            <HeaderedContent
                title={'Edit Album'}
                description={'Update album information and settings.'}
                className={'mt-16 pb-8'}
            >
                <Formik
                    initialValues={initialValues}
                    validationSchema={validationSchema}
                    enableReinitialize={true}
                    onSubmit={async (values, { setSubmitting }) => {
                        try {
                            const updated = await updateAlbumAPI(albumId, values);
                            setAlbum(updated);
                            addFlash({
                                key: 'album-update',
                                type: 'success',
                                title: 'Saved',
                                message: 'Album updated successfully',
                            });
                        } catch (err: any) {
                            addFlash({
                                key: 'album-update',
                                type: 'error',
                                title: 'Error',
                                message: err?.response?.data?.error || 'Failed to update album',
                            });
                        } finally {
                            setSubmitting(false);
                        }
                    }}
                >
                    {({ values, setFieldValue, isSubmitting, dirty, resetForm }) => (
                        <Form className='space-y-6'>
                            <FieldGroup>
                                <FormikFieldComponent
                                    name='name'
                                    label='Name'
                                    placeholder='Enter album name'
                                    required
                                />

                                <FormikFieldComponent
                                    name='description'
                                    label='Description'
                                    fieldType='textarea'
                                    rows={4}
                                    placeholder='Optional description of the album'
                                />

                                <FormikFieldComponent
                                    name='location'
                                    label='Location'
                                    placeholder='e.g., Paris, France'
                                    description='Optional location information for the album'
                                />

                                <Field>
                                    <Label>Sort Order</Label>
                                    <Listbox
                                        value={values.sort_order}
                                        onChange={(val) => setFieldValue('sort_order', val)}
                                    >
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
                                            name='is_hidden'
                                            checked={values.is_hidden}
                                            onChange={(checked) => setFieldValue('is_hidden', checked)}
                                        />
                                        <Label htmlFor='is_hidden'>Hide this album from regular users</Label>
                                    </div>
                                    <Description>Hidden albums are not visible to regular users</Description>
                                </Field>
                            </FieldGroup>

                            <UnsavedChangesBar isDirty={dirty} isSubmitting={isSubmitting} onDiscard={resetForm} />
                        </Form>
                    )}
                </Formik>
            </HeaderedContent>
        </>
    );
}
