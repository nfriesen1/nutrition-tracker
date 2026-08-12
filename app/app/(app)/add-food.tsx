import { Placeholder } from '@/components/placeholder';

// Real screen in T-17. Reached from a meal's "+", carrying ?date= and ?meal=.
export default function AddFood() {
  return <Placeholder name="add-food" links={['/create-food', '/diary']} />;
}
