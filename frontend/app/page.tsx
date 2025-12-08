import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { FileText, Sparkles, BarChart3 } from 'lucide-react';

export default function HomePage() {
  return (
    <div className="container py-12 md:py-24">
      <div className="mx-auto max-w-4xl text-center">
        <h1 className="font-heading text-4xl font-bold tracking-tight sm:text-5xl md:text-6xl">
          CV Interactif Intelligent
        </h1>
        <p className="mt-6 text-lg text-muted-foreground">
          Découvrez mon parcours professionnel adaptatif et générez des lettres de motivation
          personnalisées grâce à l'intelligence artificielle.
        </p>

        <div className="mt-10 flex flex-wrap justify-center gap-4">
          <Button asChild size="lg">
            <Link href="/cv">Voir mon CV</Link>
          </Button>
          <Button asChild size="lg" variant="outline">
            <Link href="/letters">Générer une lettre</Link>
          </Button>
        </div>
      </div>

      <div className="mx-auto mt-24 grid max-w-5xl gap-8 md:grid-cols-3">
        <Card>
          <CardHeader>
            <FileText className="h-10 w-10 text-primary" />
            <CardTitle>CV Dynamique</CardTitle>
            <CardDescription>
              Un CV qui s'adapte automatiquement selon le contexte : backend, frontend, DevOps...
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full">
              <Link href="/cv">Explorer</Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <Sparkles className="h-10 w-10 text-primary" />
            <CardTitle>Lettres IA</CardTitle>
            <CardDescription>
              Génération de lettres de motivation et anti-motivation personnalisées par IA.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full">
              <Link href="/letters">Essayer</Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <BarChart3 className="h-10 w-10 text-primary" />
            <CardTitle>Analytics Publiques</CardTitle>
            <CardDescription>
              Dashboard temps réel des statistiques de visite et d'utilisation.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full">
              <Link href="/analytics">Voir les stats</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
