import { ActivityIndicator, StyleSheet, View } from 'react-native';

/**
 * Shown while the persisted session is being read from storage.
 *
 * Deliberately blank apart from the spinner: this frame is on screen for a few
 * milliseconds on a warm start, and anything with text in it reads as a flash.
 */
export function SessionSplash() {
  return (
    <View style={styles.container}>
      <ActivityIndicator />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, alignItems: 'center', justifyContent: 'center' },
});
