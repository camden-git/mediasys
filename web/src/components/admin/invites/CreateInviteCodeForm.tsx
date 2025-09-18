import React, { useState } from 'react';
import { useStoreActions } from '../../../store/hooks';
import { InviteCodeCreatePayload } from '../../../types';
import { Button } from '../../elements/Button.tsx';
import { Dialog, DialogActions, DialogBody, DialogTitle, DialogDescription } from '../../elements/Dialog.tsx';
import { FieldGroup } from '../../elements/Fieldset.tsx';
import FormikFieldComponent from '../../elements/FormikField.tsx';
import { Formik, Form } from 'formik';
import * as Yup from 'yup';
import { createInviteCode } from '../../../api/admin/inviteCodes';
import { useFlash } from '../../../hooks/useFlash';
import { useInviteCodes } from '../../../api/swr/useInviteCodes';

const CreateInviteCodeSchema = Yup.object().shape({
    expiresAt: Yup.string()
        .nullable()
        .transform((value) => (value === '' ? null : value))
        .test('is-valid-date', 'Invalid date format', (value) => {
            return value === null || !isNaN(new Date(value as string).getTime());
        }),
    maxUses: Yup.string()
        .nullable()
        .test('is-valid-number', 'Please enter a valid number', (value) => {
            if (!value || value.trim() === '') return true;

            const num = Number(value);
            return !isNaN(num) && Number.isInteger(num) && num >= 0;
        }),
});

export const CreateInviteCodeForm: React.FC = () => {
    const [isOpen, setIsOpen] = useState(false);
    const [isSubmitting, setIsSubmitting] = useState(false);

    const { addFlash, clearFlashes } = useFlash();
    const { mutate } = useInviteCodes();
    useStoreActions((actions) => actions.inviteCodes.appendInviteCode);
    const initialValues = {
        expiresAt: '',
        maxUses: '',
    };

    return (
        <>
            <Button onClick={() => setIsOpen(true)}>Create</Button>
            <Dialog open={isOpen} onClose={() => setIsOpen(false)}>
                <Formik
                    initialValues={initialValues}
                    validationSchema={CreateInviteCodeSchema}
                    onSubmit={async (values, { resetForm }) => {
                        setIsSubmitting(true);
                        clearFlashes('invite-codes');

                        const payload: InviteCodeCreatePayload = {
                            is_active: true,
                        };

                        if (values.expiresAt) {
                            payload.expires_at = new Date(values.expiresAt).toISOString();
                        }

                        if (values.maxUses && values.maxUses.trim() !== '') {
                            const numValue = Number(values.maxUses);
                            if (!isNaN(numValue) && numValue > 0) {
                                payload.max_uses = numValue;
                            }
                        }

                        try {
                            const newInviteCode = await createInviteCode(payload);
                            mutate((currentData) => {
                                if (!currentData) return [newInviteCode];
                                return [newInviteCode, ...currentData];
                            }, false);
                            addFlash({
                                key: 'invite-codes',
                                type: 'success',
                                message: 'Invite code created successfully!',
                            });
                            resetForm();
                            setIsOpen(false);
                        } catch (error: any) {
                            addFlash({
                                key: 'invite-codes',
                                type: 'error',
                                message: error.message || 'Failed to create invite code.',
                            });
                        } finally {
                            setIsSubmitting(false);
                        }
                    }}
                >
                    {({ isSubmitting: formikSubmitting }) => (
                        <Form>
                            <DialogTitle>Create Invite Code</DialogTitle>
                            <DialogDescription>Invite codes are 6-digit PINs required for users to register.</DialogDescription>
                            <DialogBody>
                                <FieldGroup>
                                    <FormikFieldComponent
                                        name='expiresAt'
                                        label='Expires At'
                                        type='datetime-local'
                                        disabled={isSubmitting || formikSubmitting}
                                    />
                                    <FormikFieldComponent
                                        name='maxUses'
                                        label='Max Uses'
                                        type='number'
                                        min={0}
                                        step={1}
                                        inputMode='numeric'
                                        pattern='[0-9]*'
                                        placeholder='Leave blank for unlimited'
                                        description='Leave blank or 0 to allow for unlimited uses.'
                                        disabled={isSubmitting || formikSubmitting}
                                    />
                                </FieldGroup>
                            </DialogBody>
                            <DialogActions>
                                <Button
                                    plain
                                    onClick={() => {
                                        setIsOpen(false);
                                        clearFlashes('invite-codes');
                                    }}
                                >
                                    Cancel
                                </Button>
                                <Button type='submit' disabled={isSubmitting || formikSubmitting}>
                                    {isSubmitting || formikSubmitting ? 'Creating...' : 'Create Invite Code'}
                                </Button>
                            </DialogActions>
                        </Form>
                    )}
                </Formik>
            </Dialog>
        </>
    );
};
