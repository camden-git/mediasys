import React, { forwardRef } from 'react';
import { Field as FormikField, FieldProps } from 'formik';
import { Input } from './Input';
import { Textarea } from './Textarea';
import { Field, Label, Description, ErrorMessage } from './Fieldset';

interface FormikFieldProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'name'> {
    name: string;
    label?: string;
    description?: string;
    validate?: (value: any) => undefined | string | Promise<any>;
    className?: string;
    type?: 'input' | 'textarea'; // TODO: this breaks input types so this needs to be refactored
    rows?: number;
}

export const FormikFieldComponent = forwardRef<HTMLInputElement | HTMLTextAreaElement, FormikFieldProps>(
    ({ name, label, description, validate, className, type = 'input', rows, ...inputProps }, ref) => (
        <FormikField name={name} validate={validate}>
            {({ field, form: { errors, touched, setFieldValue, setFieldTouched } }: FieldProps) => {
                const hasError = touched[name] && errors[name];

                const commonProps = {
                    ref,
                    id: name,
                    'data-invalid': hasError ? true : undefined,
                    ...field,
                    ...inputProps,
                    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
                        const value = e.target.value;
                        if (inputProps.type === 'number') {
                            // allow empty string or valid numbers
                            if (value === '' || !isNaN(Number(value))) {
                                setFieldValue(name, value === '' ? '' : Number(value));
                            }
                        } else {
                            setFieldValue(name, value);
                        }
                    },
                    onBlur: (e: React.FocusEvent<HTMLInputElement | HTMLTextAreaElement>) => {
                        setFieldTouched(name, true);
                        field.onBlur(e);
                    },
                };

                return (
                    <Field className={className}>
                        {label && <Label htmlFor={name}>{label}</Label>}
                        {type === 'textarea' ? <Textarea {...commonProps} rows={rows} /> : <Input {...commonProps} />}
                        {hasError ? (
                            <ErrorMessage>{errors[name] as string}</ErrorMessage>
                        ) : description ? (
                            <Description>{description}</Description>
                        ) : null}
                    </Field>
                );
            }}
        </FormikField>
    ),
);

FormikFieldComponent.displayName = 'FormikFieldComponent';

// Export as default for easier imports
export default FormikFieldComponent;
