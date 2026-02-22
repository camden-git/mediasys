import { Formik, Form } from 'formik';
import * as Yup from 'yup';
import { useStoreActions, useStoreState } from '../../../../store/hooks';
import { UpdateAlbumPayload } from '../../../../api/admin/albums';
import { Button } from '../../../elements/Button';
import { Field, FieldGroup, Label, Description } from '../../../elements/Fieldset';
import { Checkbox } from '../../../elements/Checkbox';
import { Select } from '../../../elements/Select';
import HeaderedContent from '../../../elements/HeaderedContent.tsx';
import FormikFieldComponent from '../../../elements/FormikField';

const validationSchema = Yup.object({
    name: Yup.string().required('Name is required').min(1, 'Name must be at least 1 character'),
    description: Yup.string().optional(),
    location: Yup.string().optional(),
    sort_order: Yup.string().optional(),
    is_hidden: Yup.boolean().optional(),
});

export function UpdateAlbumForm() {
    const album = useStoreState((state) => state.albumContext.data!);
    const albumId = useStoreState((state) => state.albumContext.data!.id);
    const { updateAlbum } = useStoreActions((actions) => actions.adminAlbums);
    const { addFlash } = useStoreActions((actions) => actions.ui);
    const { setAlbum } = useStoreActions((actions) => actions.albumContext);

    const initialValues: UpdateAlbumPayload = {
        name: album.name,
        description: album.description || '',
        location: album.location || '',
        sort_order: album.sort_order,
        is_hidden: album.is_hidden,
    };

    return (
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
                    updateAlbum({
                        id: albumId,
                        payload: values,
                        addFlash,
                        setAlbum,
                        // onSuccess: () => {
                        //     navigate(`/admin/albums/${albumId}`);
                        // },
                    });
                    setSubmitting(false);
                }}
            >
                {({ values, handleChange, handleBlur, setFieldValue, isSubmitting }) => (
                    <Form className='space-y-6'>
                        <FieldGroup>
                            <FormikFieldComponent name='name' label='Name' placeholder='Enter album name' required />

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
                                <Label htmlFor='sort_order'>Sort Order</Label>
                                <Select
                                    id='sort_order'
                                    name='sort_order'
                                    value={values.sort_order}
                                    onChange={handleChange}
                                    onBlur={handleBlur}
                                >
                                    <optgroup label='Filename'>
                                        <option value='filename_asc'>Filename (A–Z)</option>
                                        <option value='filename_desc'>Filename (Z–A)</option>
                                        <option value='filename_nat'>Filename (Natural)</option>
                                    </optgroup>
                                    <optgroup label='Capture Date'>
                                        <option value='date_desc'>Capture Date (Newest First)</option>
                                        <option value='date_asc'>Capture Date (Oldest First)</option>
                                    </optgroup>
                                    <optgroup label='File'>
                                        <option value='mod_time_desc'>Modified Time (Newest First)</option>
                                        <option value='mod_time_asc'>Modified Time (Oldest First)</option>
                                        <option value='file_size_desc'>File Size (Largest First)</option>
                                        <option value='file_size_asc'>File Size (Smallest First)</option>
                                    </optgroup>
                                    <optgroup label='Camera Settings'>
                                        <option value='iso_asc'>ISO (Low to High)</option>
                                        <option value='iso_desc'>ISO (High to Low)</option>
                                        <option value='aperture_asc'>Aperture (Small to Large)</option>
                                        <option value='aperture_desc'>Aperture (Large to Small)</option>
                                        <option value='shutter_speed_desc'>Shutter Speed (Fast to Slow)</option>
                                        <option value='shutter_speed_asc'>Shutter Speed (Slow to Fast)</option>
                                        <option value='focal_length_asc'>Focal Length (Short to Long)</option>
                                        <option value='focal_length_desc'>Focal Length (Long to Short)</option>
                                    </optgroup>
                                    <optgroup label='Equipment'>
                                        <option value='camera_asc'>Camera (A–Z)</option>
                                    </optgroup>
                                </Select>
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

                        <div className='flex justify-end space-x-3'>
                            <Button type='submit' disabled={isSubmitting}>
                                {isSubmitting ? 'Updating...' : 'Update Album'}
                            </Button>
                        </div>
                    </Form>
                )}
            </Formik>
        </HeaderedContent>
    );
}
