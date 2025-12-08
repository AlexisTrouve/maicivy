import Link from 'next/link';
import { Button } from '@/components/ui/button';

export default function NotFound() {
  return (
    <div className="container flex min-h-screen flex-col items-center justify-center">
      <h1 className="text-6xl font-bold">404</h1>
      <p className="mt-4 text-xl text-muted-foreground">Page non trouvée</p>
      <Button asChild className="mt-8">
        <Link href="/">Retour à l'accueil</Link>
      </Button>
    </div>
  );
}
