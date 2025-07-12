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
                {({ values, handleChange, handleBlur, isSubmitting }) => (
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
                                    <option value='filename_asc'>By Filename (A-Z)</option>
                                    <option value='filename_nat'>By Filename (Natural)</option>
                                    <option value='date_desc'>By Date (Newest First)</option>
                                    <option value='date_asc'>By Date (Oldest First)</option>
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
