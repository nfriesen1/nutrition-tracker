import { Controller, type Control, type FieldValues, type Path } from 'react-hook-form';
import { StyleSheet, Text, TextInput, View, type TextInputProps } from 'react-native';

type TextFieldProps<T extends FieldValues> = TextInputProps & {
  control: Control<T>;
  name: Path<T>;
  label: string;
};

/**
 * A labelled text input bound to react-hook-form.
 *
 * Exists so the keyboard, autofill, and accessibility props are set once rather than per
 * screen — a form field that opens the wrong keyboard is the most common way a React
 * Native app feels wrong on a phone.
 */
export function TextField<T extends FieldValues>({
  control,
  name,
  label,
  ...inputProps
}: TextFieldProps<T>) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field: { onChange, onBlur, value }, fieldState: { error } }) => (
        <View style={styles.container}>
          <Text style={styles.label}>{label}</Text>
          <TextInput
            value={value ?? ''}
            onChangeText={onChange}
            onBlur={onBlur}
            accessibilityLabel={label}
            // Announces the validation message to a screen reader instead of leaving the
            // field silently red.
            aria-invalid={error !== undefined}
            style={[styles.input, error && styles.inputError]}
            {...inputProps}
          />
          {error ? <Text style={styles.error}>{error.message}</Text> : null}
        </View>
      )}
    />
  );
}

const styles = StyleSheet.create({
  container: { gap: 6 },
  label: { fontSize: 14, fontWeight: '500' },
  input: {
    borderWidth: 1,
    borderColor: '#d4d4d8',
    borderRadius: 8,
    paddingHorizontal: 12,
    // Comfortably past the 44pt minimum touch target.
    paddingVertical: 12,
    fontSize: 16,
  },
  inputError: { borderColor: '#dc2626' },
  error: { color: '#dc2626', fontSize: 13 },
});
