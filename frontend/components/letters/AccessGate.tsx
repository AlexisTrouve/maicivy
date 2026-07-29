'use client';

import { motion } from 'framer-motion';
import { Lock, Eye, Sparkles } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { Link } from '@/i18n/navigation';
import { useVisitCount } from '@/hooks/useVisitCount';

interface AccessGateProps {
  children: React.ReactNode;
}

export function AccessGate({ children }: AccessGateProps) {
  const t = useTranslations('letters.accessGate');
  const { status, loading } = useVisitCount();

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  // Si accès autorisé
  if (status?.hasAccess) {
    return <>{children}</>;
  }

  // Teaser (accès refusé)
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="max-w-2xl mx-auto"
    >
      <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-8 border border-slate-200 dark:border-slate-700">
        {/* Icon */}
        <div className="flex justify-center mb-6">
          <div className="relative">
            <div className="absolute inset-0 bg-blue-500/20 blur-xl rounded-full"></div>
            <div className="relative bg-gradient-to-br from-blue-500 to-purple-500 p-4 rounded-full">
              <Lock className="w-8 h-8 text-white" />
            </div>
          </div>
        </div>

        {/* Title */}
        <h2 className="text-2xl font-bold text-center mb-4 text-slate-900 dark:text-white">
          {t('title')}
        </h2>

        {/* Description */}
        <p className="text-center text-slate-600 dark:text-slate-300 mb-6">
          {t('description')}
        </p>

        {/* Progress */}
        <div className="mb-6">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
              {t('progress')}
            </span>
            <span className="text-sm font-bold text-blue-600 dark:text-blue-400">
              {t('visits', { count: status?.visitCount || 0 })}
            </span>
          </div>
          <div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-3 overflow-hidden">
            <motion.div
              initial={{ width: 0 }}
              animate={{ width: `${((status?.visitCount || 0) / 3) * 100}%` }}
              transition={{ duration: 1, ease: 'easeOut' }}
              className="bg-gradient-to-r from-blue-500 to-purple-500 h-full rounded-full"
            />
          </div>
        </div>

        {/* Remaining visits */}
        <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 mb-6">
          <div className="flex items-start gap-3">
            <Eye className="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
            <div>
              <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                {t('remaining', { count: status?.remainingVisits || 3 })}
              </p>
              <p className="text-xs text-blue-700 dark:text-blue-300 mt-1">
                {t('encourage')}
              </p>
            </div>
          </div>
        </div>

        {/* Features preview */}
        <div className="space-y-3">
          <p className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
            {t('unlockTitle')}
          </p>
          {[
            t('features.motivation'),
            t('features.anti'),
            t('features.pdf'),
            t('features.analysis'),
          ].map((feature, index) => (
            <motion.div
              key={index}
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: index * 0.1 }}
              className="flex items-center gap-2"
            >
              <Sparkles className="w-4 h-4 text-purple-500 flex-shrink-0" />
              <span className="text-sm text-slate-600 dark:text-slate-400">{feature}</span>
            </motion.div>
          ))}
        </div>

        {/* CTA */}
        <div className="mt-8 text-center">
          <Link
            href="/cv"
            className="inline-flex items-center gap-2 bg-gradient-to-r from-blue-600 to-purple-600 text-white px-6 py-3 rounded-lg font-medium hover:from-blue-700 hover:to-purple-700 transition-all duration-200 shadow-lg hover:shadow-xl"
          >
            {t('exploreCV')}
          </Link>
        </div>
      </div>
    </motion.div>
  );
}
