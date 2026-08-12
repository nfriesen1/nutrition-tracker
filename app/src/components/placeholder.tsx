import { Link, type Href } from 'expo-router';
import { StyleSheet, Text, View } from 'react-native';

/**
 * Scaffold-only screen body (T-02). Every route renders one of these so the route
 * tree can be navigated and verified before any real screen exists.
 *
 * Each of these is replaced by its real screen in T-04 and T-15 through T-20, and
 * this component is deleted once the last one lands.
 */
export function Placeholder({ name, links = [] }: { name: string; links?: Href[] }) {
  return (
    <View style={styles.container}>
      <Text style={styles.name}>{name}</Text>
      <Text style={styles.note}>placeholder — T-02</Text>
      {links.map((href) => (
        <Link key={String(href)} href={href} style={styles.link}>
          {String(href)}
        </Link>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 8, padding: 24 },
  name: { fontSize: 24, fontWeight: '600' },
  note: { fontSize: 13, opacity: 0.5, marginBottom: 16 },
  link: { fontSize: 16, color: '#2563eb', paddingVertical: 6 },
});
