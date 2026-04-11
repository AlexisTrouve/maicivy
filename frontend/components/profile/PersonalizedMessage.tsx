// frontend/components/profile/PersonalizedMessage.tsx
'use client';

import React from 'react';
import { motion } from 'framer-motion';
import { useProfileDetection } from '@/hooks/useProfileDetection';

interface PersonalizedMessageProps {
  className?: string;
}

const messages = {
  recruiter: {
    title: 'Bienvenue, Recruteur !',
    message: "J'ai détecté que vous recherchez probablement des talents. Vous pouvez générer des lettres de motivation personnalisées immédiatement, sans attendre 3 visites.",
    cta: 'Générer des lettres IA',
    ctaLink: '/letters',
    icon: '👔',
  },
  cto: {
    title: 'Bonjour, CTO !',
    message: "Intéressé par mon profil technique ? Utilisez l'IA pour générer une analyse détaillée de mes compétences et de mon adéquation avec vos besoins.",
    cta: 'Explorer mes compétences',
    ctaLink: '/cv?theme=backend',
    icon: '🎯',
  },
  tech_lead: {
    title: 'Hey, Tech Lead !',
    message: "En tant que leader technique, vous apprécierez peut-être de voir comment l'IA peut analyser mon profil et générer des lettres personnalisées pour votre équipe.",
    cta: 'Voir la démo IA',
    ctaLink: '/letters',
    icon: '🚀',
  },
  ceo: {
    title: 'Bienvenue, CEO !',
    message: "Merci de votre visite. L'IA peut générer une analyse stratégique de mon profil et comment je pourrais contribuer à votre vision.",
    cta: 'Générer une analyse',
    ctaLink: '/letters',
    icon: '⭐',
  },
  developer: {
    title: 'Un Fellow Developer !',
    message: "Sympa de te voir ici ! Explore mon GitHub, ma timeline de projets, ou teste le générateur de lettres IA (c'est assez fun).",
    cta: 'Voir mes projets',
    ctaLink: '/cv?theme=projects',
    icon: '💻',
  },
  other: {
    title: 'Bienvenue !',
    message: 'Explorez mon CV interactif et découvrez comment je peux contribuer à vos projets.',
    cta: 'Explorer le CV',
    ctaLink: '/cv',
    icon: '👋',
  },
};

export function PersonalizedMessage({ className = '' }: PersonalizedMessageProps) {
  const { profileType, isDetected, enrichmentData } = useProfileDetection();

  // Ne pas afficher si pas de profil détecté
  if (!isDetected) {
    return null;
  }

  const config = messages[profileType as keyof typeof messages] || messages.other;

  // Récupérer le nom de l'entreprise si disponible
  const companyName = enrichmentData?.company_name as string | undefined;

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.6, delay: 0.3 }}
      className={`max-w-3xl mx-auto ${className}`}
    >
      <div className="bg-gradient-to-br from-blue-50 to-purple-50 rounded-2xl p-6 md:p-8 shadow-xl border border-blue-100">
        <div className="flex items-start gap-4">
          {/* Icon */}
          <div className="text-4xl md:text-5xl flex-shrink-0">
            {config.icon}
          </div>

          {/* Content */}
          <div className="flex-1">
            <h2 className="text-2xl md:text-3xl font-bold text-gray-900 mb-2">
              {config.title}
            </h2>

            {/* Company name si disponible */}
            {companyName && (
              <div className="text-sm text-blue-600 font-medium mb-3">
                {companyName}
              </div>
            )}

            <p className="text-gray-700 leading-relaxed mb-4">
              {config.message}
            </p>

            {/* CTA Button */}
            <motion.a
              href={config.ctaLink}
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              className="inline-block px-6 py-3 bg-gradient-to-r from-blue-500 to-purple-500 text-white font-semibold rounded-lg shadow-lg hover:shadow-xl transition-shadow"
            >
              {config.cta} →
            </motion.a>
          </div>
        </div>
      </div>
    </motion.div>
  );
}

// Version compacte pour sidebar
export function PersonalizedMessageCompact() {
  const { profileType, isDetected } = useProfileDetection();

  if (!isDetected || profileType === 'other') {
    return null;
  }

  const config = messages[profileType as keyof typeof messages] || messages.other;

  return (
    <motion.div
      initial={{ opacity: 0, x: 20 }}
      animate={{ opacity: 1, x: 0 }}
      className="bg-white rounded-lg p-4 shadow-md border border-gray-200"
    >
      <div className="flex items-center gap-3 mb-2">
        <span className="text-2xl">{config.icon}</span>
        <h3 className="font-semibold text-gray-900">
          {config.title}
        </h3>
      </div>
      <p className="text-sm text-gray-600 mb-3">
        {config.message}
      </p>
      <a
        href={config.ctaLink}
        className="text-sm text-blue-600 hover:text-blue-700 font-medium"
      >
        {config.cta} →
      </a>
    </motion.div>
  );
}

// Notification toast pour détection
export function ProfileDetectionToast() {
  const { profileType, isDetected, confidence } = useProfileDetection();
  const [visible, setVisible] = React.useState(false);

  React.useEffect(() => {
    if (isDetected && profileType !== 'other' && confidence >= 60) {
      setVisible(true);
      // Auto-hide après 5 secondes
      setTimeout(() => setVisible(false), 5000);
    }
  }, [isDetected, profileType, confidence]);

  if (!visible) return null;

  const config = messages[profileType as keyof typeof messages] || messages.other;

  return (
    <motion.div
      initial={{ opacity: 0, y: 50 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 50 }}
      className="fixed bottom-4 right-4 z-50 max-w-md"
    >
      <div className="bg-white rounded-lg shadow-2xl border border-gray-200 p-4">
        <div className="flex items-start gap-3">
          <span className="text-2xl">{config.icon}</span>
          <div className="flex-1">
            <h4 className="font-semibold text-gray-900 mb-1">
              Profil Détecté !
            </h4>
            <p className="text-sm text-gray-600">
              {config.title} - Accès IA débloqué immédiatement.
            </p>
          </div>
          <button
            onClick={() => setVisible(false)}
            className="text-gray-400 hover:text-gray-600"
          >
            ✕
          </button>
        </div>
      </div>
    </motion.div>
  );
}
