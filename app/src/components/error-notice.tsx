import { Pressable, StyleSheet, Text, View } from 'react-native';

/**
 * Inline error message. Used for failures the user can act on or retry — a rejected
 * login, a failed fetch — never for validation, which attaches to its own field.
 */
export function ErrorNotice({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <View style={styles.container} accessibilityRole="alert">
      <Text style={styles.message}>{message}</Text>
      {onRetry ? (
        <Pressable onPress={onRetry} accessibilityRole="button">
          <Text style={styles.retry}>Retry</Text>
        </Pressable>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    backgroundColor: '#fef2f2',
    borderColor: '#fecaca',
    borderWidth: 1,
    borderRadius: 8,
    padding: 12,
    gap: 8,
  },
  message: { color: '#991b1b', fontSize: 14 },
  retry: { color: '#991b1b', fontSize: 14, fontWeight: '600' },
});
