import { Formik, Form } from 'formik';
import * as Yup from 'yup';
import { useAlbumData } from '../../../../store/albumContextHooks';
import { queryClient, invalidateAlbums } from '../../../../lib/queryClient';
import { queryKeys } from '../../../../lib/queryKeys';
import { useUIStore } from '../../../../store/useUIStore';
import { UpdateAlbumPayload, updateAlbum as updateAlbumAPI } from '../../../../api/admin/albums';
import { Field, FieldGroup, Label, Description } from '../../../elements/Fieldset';
import { Checkbox } from '../../../elements/Checkbox';
import SortOrderListbox from '../../shared/SortOrderListbox';
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
    const album = useAlbumData();
    const albumId = album.id;
    const addFlash = useUIStore((s) => s.addFlash);

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
                            queryClient.setQueryData(queryKeys.albums.bySlug(updated.slug), updated);
                            void invalidateAlbums();
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
                                message: err.message || 'Failed to update album',
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
