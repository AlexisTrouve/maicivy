import Link from 'next/link';
import { Github, Linkedin, Mail } from 'lucide-react';

export function Footer() {
  const currentYear = new Date().getFullYear();

  return (
    <footer className="border-t bg-muted/50">
      <div className="container py-8 md:py-12">
        <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
          <div>
            <h3 className="font-heading text-lg font-semibold">maicivy</h3>
            <p className="mt-2 text-sm text-muted-foreground">
              CV interactif intelligent avec génération de lettres par IA
            </p>
          </div>

          <div>
            <h3 className="font-heading text-lg font-semibold">Navigation</h3>
            <ul className="mt-2 space-y-2 text-sm">
              <li>
                <Link href="/cv" className="text-muted-foreground hover:text-foreground">
                  CV Dynamique
                </Link>
              </li>
              <li>
                <Link href="/letters" className="text-muted-foreground hover:text-foreground">
                  Générateur de Lettres
                </Link>
              </li>
              <li>
                <Link href="/analytics" className="text-muted-foreground hover:text-foreground">
                  Analytics
                </Link>
              </li>
            </ul>
          </div>

          <div>
            <h3 className="font-heading text-lg font-semibold">Contact</h3>
            <div className="mt-2 flex gap-4">
              <a
                href="https://github.com/yourusername"
                target="_blank"
                rel="noopener noreferrer"
                className="text-muted-foreground hover:text-foreground"
              >
                <Github className="h-5 w-5" />
                <span className="sr-only">GitHub</span>
              </a>
              <a
                href="https://linkedin.com/in/yourusername"
                target="_blank"
                rel="noopener noreferrer"
                className="text-muted-foreground hover:text-foreground"
              >
                <Linkedin className="h-5 w-5" />
                <span className="sr-only">LinkedIn</span>
              </a>
              <a
                href="mailto:contact@example.com"
                className="text-muted-foreground hover:text-foreground"
              >
                <Mail className="h-5 w-5" />
                <span className="sr-only">Email</span>
              </a>
            </div>
          </div>
        </div>

        <div className="mt-8 border-t pt-8 text-center text-sm text-muted-foreground">
          <p>&copy; {currentYear} maicivy. Tous droits réservés.</p>
        </div>
      </div>
    </footer>
  );
}
