import { Pressable, StyleSheet, Text } from 'react-native';

/**
 * Primary form button.
 *
 * `busyLabel` is shown while submitting. The button disables, but the fields around it do
 * not — a user who spots a typo mid-request should be able to fix it rather than wait.
 */
export function SubmitButton({
  label,
  busyLabel,
  busy,
  onPress,
}: {
  label: string;
  busyLabel: string;
  busy: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      onPress={onPress}
      disabled={busy}
      accessibilityRole="button"
      accessibilityState={{ disabled: busy, busy }}
      style={[styles.button, busy && styles.busy]}
    >
      <Text style={styles.label}>{busy ? busyLabel : label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  button: {
    backgroundColor: '#2563eb',
    borderRadius: 8,
    paddingVertical: 14,
    alignItems: 'center',
  },
  busy: { opacity: 0.6 },
  label: { color: '#ffffff', fontSize: 16, fontWeight: '600' },
});
