// frontend/components/profile/ProfileBadge.tsx
'use client';

import React, { useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useProfileDetection } from '@/hooks/useProfileDetection';

interface ProfileBadgeProps {
  className?: string;
}

const profileConfig = {
  recruiter: {
    icon: '👔',
    label: 'Profil Recruteur Détecté',
    color: 'from-blue-500 to-purple-500',
    bgColor: 'bg-blue-50',
    borderColor: 'border-blue-500',
  },
  cto: {
    icon: '🎯',
    label: 'Profil CTO Détecté',
    color: 'from-purple-500 to-pink-500',
    bgColor: 'bg-purple-50',
    borderColor: 'border-purple-500',
  },
  tech_lead: {
    icon: '🚀',
    label: 'Profil Tech Lead Détecté',
    color: 'from-green-500 to-teal-500',
    bgColor: 'bg-green-50',
    borderColor: 'border-green-500',
  },
  ceo: {
    icon: '⭐',
    label: 'Profil CEO Détecté',
    color: 'from-yellow-500 to-orange-500',
    bgColor: 'bg-yellow-50',
    borderColor: 'border-yellow-500',
  },
  developer: {
    icon: '💻',
    label: 'Fellow Developer!',
    color: 'from-indigo-500 to-blue-500',
    bgColor: 'bg-indigo-50',
    borderColor: 'border-indigo-500',
  },
  other: {
    icon: '👋',
    label: 'Bienvenue',
    color: 'from-gray-500 to-gray-600',
    bgColor: 'bg-gray-50',
    borderColor: 'border-gray-500',
  },
};

export function ProfileBadge({ className = '' }: ProfileBadgeProps) {
  const { profileType, confidence, isDetected, bypassEnabled } = useProfileDetection();
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    // Afficher le badge avec un délai pour l'animation
    if (isDetected && profileType !== 'other') {
      setTimeout(() => setVisible(true), 500);
    }
  }, [isDetected, profileType]);

  // Ne pas afficher pour les profils "other" ou non détectés
  if (!isDetected || profileType === 'other') {
    return null;
  }

  const config = profileConfig[profileType as keyof typeof profileConfig] || profileConfig.other;

  return (
    <AnimatePresence>
      {visible && (
        <motion.div
          initial={{ opacity: 0, y: -20, scale: 0.9 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: -20, scale: 0.9 }}
          transition={{ duration: 0.5, ease: 'easeOut' }}
          className={`relative ${className}`}
        >
          <div
            className={`
              ${config.bgColor}
              ${config.borderColor}
              border-l-4
              rounded-lg
              p-4
              shadow-lg
              backdrop-blur-sm
            `}
          >
            {/* Gradient overlay */}
            <div className={`absolute inset-0 bg-gradient-to-r ${config.color} opacity-5 rounded-lg`} />

            <div className="relative flex items-center gap-3">
              {/* Icon avec pulse animation */}
              <motion.div
                animate={{
                  scale: [1, 1.2, 1],
                }}
                transition={{
                  duration: 2,
                  repeat: Infinity,
                  ease: 'easeInOut',
                }}
                className="text-3xl"
              >
                {config.icon}
              </motion.div>

              {/* Text content */}
              <div className="flex-1">
                <div className="font-semibold text-gray-900">
                  {config.label}
                </div>

                {/* Confidence indicator */}
                {confidence > 0 && (
                  <div className="mt-1 flex items-center gap-2">
                    <div className="flex-1 h-2 bg-gray-200 rounded-full overflow-hidden">
                      <motion.div
                        initial={{ width: 0 }}
                        animate={{ width: `${confidence}%` }}
                        transition={{ duration: 1, ease: 'easeOut' }}
                        className={`h-full bg-gradient-to-r ${config.color}`}
                      />
                    </div>
                    <span className="text-xs text-gray-600">
                      {confidence}% confiance
                    </span>
                  </div>
                )}

                {/* Bypass notice */}
                {bypassEnabled && (
                  <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ delay: 0.5 }}
                    className="mt-2 text-sm text-green-700 font-medium"
                  >
                    ✓ Accès IA Débloqué
                  </motion.div>
                )}
              </div>

              {/* Close button */}
              <button
                onClick={() => setVisible(false)}
                className="text-gray-400 hover:text-gray-600 transition-colors"
                aria-label="Fermer"
              >
                <svg
                  className="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            </div>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

// Compact version pour le header
export function ProfileBadgeCompact() {
  const { profileType, isDetected, bypassEnabled } = useProfileDetection();

  if (!isDetected || profileType === 'other') {
    return null;
  }

  const config = profileConfig[profileType as keyof typeof profileConfig] || profileConfig.other;

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.8 }}
      animate={{ opacity: 1, scale: 1 }}
      className="flex items-center gap-2 px-3 py-1.5 bg-white rounded-full shadow-md border border-gray-200"
    >
      <span className="text-lg">{config.icon}</span>
      <span className="text-sm font-medium text-gray-700">
        {config.label.split(' ')[0]}
      </span>
      {bypassEnabled && (
        <span className="text-xs text-green-600">✓</span>
      )}
    </motion.div>
  );
}
