import { Placeholder } from '@/components/placeholder';

// Real screen in T-18. Reached from add-food's empty search state, carrying ?name=.
export default function CreateFood() {
  return <Placeholder name="create-food" links={['/add-food']} />;
}
