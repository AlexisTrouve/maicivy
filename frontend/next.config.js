const withNextIntl = require('next-intl/plugin')('./i18n/request.ts');

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',

  // Ignore ESLint during production builds (warnings are checked in CI)
  eslint: {
    ignoreDuringBuilds: true,
  },

  // Ignore TypeScript errors during builds (checked separately in CI)
  typescript: {
    ignoreBuildErrors: true,
  },

  // API Backend
  async rewrites() {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    return [
      {
        source: '/api/:path*',
        destination: `${apiUrl}/api/:path*`,
      },
    ];
  },

  // QUOI : /[locale]/montagne → /[locale], en PERMANENT (308).
  // POURQUOI : depuis la refonte de 2026-11, l'accueil EST la montagne. L'ancienne adresse a été
  //   partagée (liens envoyés, captures, éventuellement indexée) : elle doit mener à la page, et dire
  //   aux moteurs que l'adresse a changé pour de bon — un 308 transfère le classement, un 307 non.
  // COMMENT : ici plutôt que dans une page `redirect()` : les redirections de next.config passent
  //   AVANT le middleware i18n et le rendu, sans route fantôme à maintenir. Les locales sont celles
  //   de i18n/config.ts (ce fichier CommonJS ne peut pas importer le module TS) — une locale ajoutée
  //   là doit l'être ici ; e2e site-chrome.spec.ts vérifie la redirection.
  async redirects() {
    return [
      {
        source: '/:locale(fr|en|de|it|zh)/montagne',
        destination: '/:locale',
        permanent: true,
      },
      // L'atelier de lettres public est retiré (09/11/2026) : le dossier vivant fait mieux le même
      // travail. Les liens qui traînent encore ailleurs retombent sur l'accueil plutôt qu'en 404.
      {
        source: '/:locale(fr|en|de|it|zh)/letters',
        destination: '/:locale',
        permanent: true,
      },
    ];
  },

  // Performance optimizations
  compress: true,

  // Image optimization
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'avatars.githubusercontent.com',
      },
      // Couvertures du blog (servies par maiProFiles) : redimensionnées et converties en
      // AVIF/WebP par l'optimiseur de Next au lieu des PNG de ~2 MB (cf. BlogCard).
      {
        protocol: 'https',
        hostname: 'maiprofiles.etheryale.com',
        pathname: '/images/**',
      },
      // Captures du portfolio (servies par maiProFiles) : même recette que le blog, pour
      // ne plus envoyer la pleine définition sur l'index et les fiches projet.
      {
        protocol: 'https',
        hostname: 'maiprofiles.etheryale.com',
        pathname: '/portfolio-images/**',
      },
    ],
    // Modern image formats
    formats: ['image/avif', 'image/webp'],
    // Device sizes for responsive images
    deviceSizes: [640, 750, 828, 1080, 1200, 1920, 2048, 3840],
    imageSizes: [16, 32, 48, 64, 96, 128, 256, 384],
    // Minimize layout shift
    // 7 jours : les images de maiProFiles ont un nom dérivé de leur contenu (img_<hash>), une
    // variante optimisée ne devient jamais fausse. À 60 s, chaque visite ré-encodait les images.
    minimumCacheTTL: 604800,
  },

  // Experimental features for performance
  experimental: {
    // Optimize package imports for better tree-shaking
    optimizePackageImports: [
      'lucide-react',
      '@radix-ui/react-select',
      '@radix-ui/react-dialog',
      '@radix-ui/react-dropdown-menu',
      '@radix-ui/react-slot',
      // Tree-shake framer-motion pour ne bundler que les exports utilisés
      'framer-motion',
    ],
  },

  // Headers de sécurité
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
          {
            key: 'Referrer-Policy',
            value: 'strict-origin-when-cross-origin',
          },
        ],
      },
    ];
  },

  // Webpack optimizations
  webpack: (config, { dev, isServer }) => {
    // WSL performance optimization - reduce file watching overhead
    if (dev) {
      config.watchOptions = {
        poll: 3000,
        aggregateTimeout: 1000,
        ignored: /node_modules|\.next|coverage|__tests__|__mocks__|playwright-report|test-results/,
      };
    }

    // Production optimizations
    if (!dev && !isServer) {
      // Enable tree shaking
      config.optimization = {
        ...config.optimization,
        usedExports: true,
        sideEffects: false,
      };
    }

    return config;
  },
};

module.exports = withNextIntl(nextConfig);
