import React, { forwardRef } from 'react';
import { Field as FormikField, FieldProps } from 'formik';
import { Input } from './Input';
import { Field, Label, Description, ErrorMessage } from './Fieldset';

interface FormikFieldProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'name'> {
    name: string;
    label?: string;
    description?: string;
    validate?: (value: any) => undefined | string | Promise<any>;
    className?: string;
}

export const FormikFieldComponent = forwardRef<HTMLInputElement, FormikFieldProps>(
    ({ name, label, description, validate, className, ...inputProps }, ref) => (
        <FormikField name={name} validate={validate}>
            {({ field, form: { errors, touched, setFieldValue, setFieldTouched } }: FieldProps) => {
                const hasError = touched[name] && errors[name];

                return (
                    <Field className={className}>
                        {label && <Label htmlFor={name}>{label}</Label>}
                        <Input
                            ref={ref}
                            id={name}
                            data-invalid={hasError ? true : undefined}
                            {...field}
                            {...inputProps}
                            // handle special cases for number inputs
                            onChange={(e) => {
                                const value = e.target.value;
                                if (inputProps.type === 'number') {
                                    // allow empty string or valid numbers
                                    if (value === '' || !isNaN(Number(value))) {
                                        setFieldValue(name, value === '' ? '' : Number(value));
                                    }
                                } else {
                                    setFieldValue(name, value);
                                }
                            }}
                            onBlur={(e) => {
                                setFieldTouched(name, true);
                                field.onBlur(e);
                            }}
                        />
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
