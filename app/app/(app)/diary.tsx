import { Placeholder } from '@/components/placeholder';

// Real screen in T-15. Home: the day view.
export default function Diary() {
  return (
    <Placeholder
      name="diary"
      links={['/add-food', '/settings', '/login']}
    />
  );
}
