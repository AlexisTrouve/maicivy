import type { Config } from 'tailwindcss';

const config: Config = {
  // Le site public est SOMBRE en permanence (plus de bascule) : son <html> porte `dark` en dur
  // (app/[locale]/layout.tsx) pour que les variantes `dark:` des pages internes, écrites du temps
  // des deux thèmes, restent actives. L'admin n'a jamais porté la classe et ne la porte toujours pas.
  darkMode: ['class'],
  content: [
    './pages/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    './app/**/*.{ts,tsx}',
    './src/**/*.{ts,tsx}',
  ],
  theme: {
    container: {
      center: true,
      padding: '2rem',
      screens: {
        '2xl': '1400px',
      },
    },
    extend: {
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))',
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
        // Or — le SIGNAL partagé par le panneau admin et l'univers de la montagne. POURQUOI une palette
        // littérale et non une variable CSS : les variables ci-dessus diffèrent entre le site public
        // et l'admin (globals.css › .admin-surface), l'or est le même des deux côtés.
        // COMMENT l'utiliser : filets, bordures, état actif, action principale. JAMAIS en couleur de
        // texte courant ni en aplat large — l'or perd son sens de signal dès qu'il couvre une surface.
        gold: {
          DEFAULT: '#d9b45f', // trait et texte d'accent sur fond sombre
          soft: '#ecd8a4', // libellés sur fond doré translucide (contraste AA sur slate-950)
          dim: '#7a6329', // bordures au repos, à peine perceptibles
          bright: '#f0d590', // haut du dégradé des boutons pleins
          deep: '#c9a24d', // bas du dégradé — l'écart des deux fait le relief
        },
        // Échelle de SURFACES du panneau admin — trois niveaux, jamais plus.
        //
        // POURQUOI une échelle nommée plutôt que des opacités choisies au cas par cas : des panneaux
        // en `bg-slate-900/50` sur un fond sombre se détachaient à peine, et deux écrans voisins ne
        // tombaient pas sur la même valeur. Résultat : tout à plat, l'œil ne sait pas ce qui est
        // posé sur quoi. Trois valeurs OPAQUES et distinctes rendent la profondeur lisible sans
        // avoir à comparer des transparences.
        //
        // base < raised < ... et `inset` REDESCEND sous la base : un champ est un creux dans le
        // panneau, pas une carte posée dessus. C'est ce qui distingue « je saisis ici » de
        // « je lis ici ».
        surface: {
          base: '#060911', // fond de page (cf. .admin-surface)
          raised: '#121b2c', // panneau posé sur le fond
          inset: '#080d18', // champ creusé dans le panneau
        },
      },
      boxShadow: {
        // Ombre de panneau : un contact net (1 px) + une diffusion large. Le contact ancre le
        // panneau, la diffusion crée la hauteur. Une seule des deux donne soit un trait sec, soit
        // un halo flou qui ne pose rien.
        panel: '0 1px 2px rgba(0,0,0,0.5), 0 10px 30px -12px rgba(0,0,0,0.8)',
        // Idem + un liseré et une lueur dorés, réservés aux panneaux qui portent LE livrable.
        'panel-gold':
          '0 1px 2px rgba(0,0,0,0.5), 0 10px 30px -12px rgba(0,0,0,0.8), 0 0 0 1px rgba(217,180,95,0.10), 0 0 44px -18px rgba(217,180,95,0.45)',
        // Relief interne des boutons pleins : une arête claire en haut (la lumière vient d'en haut)
        // et une retombée colorée en dessous.
        gold: 'inset 0 1px 0 rgba(255,255,255,0.28), 0 2px 10px -3px rgba(217,180,95,0.55)',
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      fontFamily: {
        sans: ['var(--font-inter)', 'sans-serif'],
        // Police de titre PAR SURFACE (globals.css) : serif de la montagne sur le site public,
        // Poppins sur l'admin. La pile complète (repli générique compris) vit dans la variable.
        heading: ['var(--font-heading)'],
      },
      keyframes: {
        'accordion-down': {
          from: { height: '0' },
          to: { height: 'var(--radix-accordion-content-height)' },
        },
        'accordion-up': {
          from: { height: 'var(--radix-accordion-content-height)' },
          to: { height: '0' },
        },
      },
      animation: {
        'accordion-down': 'accordion-down 0.2s ease-out',
        'accordion-up': 'accordion-up 0.2s ease-out',
      },
    },
  },
  plugins: [require('tailwindcss-animate'), require('@tailwindcss/typography')],
};

export default config;
