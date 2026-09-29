import React, { useState } from 'react';
import { Button } from '../../elements/Button';
import { Dialog, DialogActions, DialogBody, DialogTitle, DialogDescription } from '../../elements/Dialog';
import { Field, FieldGroup, Label, ErrorMessage as FieldErrorMessage, Description } from '../../elements/Fieldset';
import { Input } from '../../elements/Input';
import { Formik, Form, ErrorMessage } from 'formik';
import * as Yup from 'yup';
import { GroupCreatePayload } from '../../../api/admin/groups';
import { useFlash } from '../../../hooks/useFlash';
import { Checkbox } from '../../elements/Checkbox';

const GroupSchema = Yup.object().shape({
    name: Yup.string().required('Name is required.'),
    slug: Yup.string()
        .required('Slug is required.')
        .matches(/^[a-z0-9]+(?:-[a-z0-9]+)*$/, 'Slug must be lowercase letters, numbers, and hyphens only.'),
    description: Yup.string(),
    is_hidden: Yup.boolean(),
});

interface GroupFormProps {
    isOpen: boolean;
    onClose: () => void;
    title: string;
    description: string;
    initialValues: GroupCreatePayload;
    enableReinitialize?: boolean;
    successMessage: string;
    failureMessage: string;
    submitLabel: string;
    submittingLabel: string;
    onSave: (values: GroupCreatePayload) => Promise<unknown>;
    onSaved: () => void;
}

const GroupForm: React.FC<GroupFormProps> = ({
    isOpen,
    onClose,
    title,
    description,
    initialValues,
    enableReinitialize,
    successMessage,
    failureMessage,
    submitLabel,
    submittingLabel,
    onSave,
    onSaved,
}) => {
    const [isSubmitting, setIsSubmitting] = useState(false);
    const { addFlash, clearFlashes } = useFlash();

    if (!isOpen) return null;

    return (
        <Dialog open={isOpen} onClose={onClose}>
            <Formik
                initialValues={initialValues}
                validationSchema={GroupSchema}
                enableReinitialize={enableReinitialize}
                onSubmit={async (values, { resetForm }) => {
                    setIsSubmitting(true);
                    clearFlashes('groups');

                    try {
                        await onSave(values);
                        addFlash({
                            key: 'groups',
                            type: 'success',
                            message: successMessage,
                        });
                        resetForm();
                        onSaved();
                        onClose();
                    } catch (error: any) {
                        addFlash({
                            key: 'groups',
                            type: 'error',
                            message: error.message || failureMessage,
                        });
                    } finally {
                        setIsSubmitting(false);
                    }
                }}
            >
                {({ values, handleChange, handleBlur, setFieldValue }) => (
                    <Form>
                        <DialogTitle>{title}</DialogTitle>
                        <DialogDescription>{description}</DialogDescription>
                        <DialogBody>
                            <FieldGroup>
                                <Field>
                                    <Label htmlFor='name'>Name</Label>
                                    <Input
                                        id='name'
                                        name='name'
                                        type='text'
                                        value={values.name}
                                        onChange={handleChange}
                                        onBlur={handleBlur}
                                        disabled={isSubmitting}
                                    />
                                    <ErrorMessage name='name' component={FieldErrorMessage} />
                                </Field>

                                <Field>
                                    <Label htmlFor='slug'>Slug</Label>
                                    <Input
                                        id='slug'
                                        name='slug'
                                        type='text'
                                        value={values.slug}
                                        onChange={handleChange}
                                        onBlur={handleBlur}
                                        disabled={isSubmitting}
                                        placeholder='my-group-slug'
                                    />
                                    <Description>URL-safe identifier, e.g. summer-2024</Description>
                                    <ErrorMessage name='slug' component={FieldErrorMessage} />
                                </Field>

                                <Field>
                                    <Label htmlFor='description'>Description</Label>
                                    <Input
                                        id='description'
                                        name='description'
                                        type='text'
                                        value={values.description}
                                        onChange={handleChange}
                                        onBlur={handleBlur}
                                        disabled={isSubmitting}
                                    />
                                    <ErrorMessage name='description' component={FieldErrorMessage} />
                                </Field>

                                <Field>
                                    <div className='flex items-center space-x-2'>
                                        <Checkbox
                                            id='is_hidden'
                                            name='is_hidden'
                                            checked={values.is_hidden}
                                            onChange={(checked) => setFieldValue('is_hidden', checked)}
                                            disabled={isSubmitting}
                                        />
                                        <Label htmlFor='is_hidden'>Hide this group from regular users</Label>
                                    </div>
                                </Field>
                            </FieldGroup>
                        </DialogBody>
                        <DialogActions>
                            <Button
                                plain
                                onClick={() => {
                                    clearFlashes('groups');
                                    onClose();
                                }}
                                disabled={isSubmitting}
                            >
                                Cancel
                            </Button>
                            <Button type='submit' disabled={isSubmitting}>
                                {isSubmitting ? submittingLabel : submitLabel}
                            </Button>
                        </DialogActions>
                    </Form>
                )}
            </Formik>
        </Dialog>
    );
};

export default GroupForm;
