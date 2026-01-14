'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { motion } from 'framer-motion';
import { Loader2, Sparkles } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { lettersApi } from '@/lib/api';
import { LetterPreview } from './LetterPreview';
import type { GeneratedLetters } from '@/lib/types';

// Validation schema
const formSchema = z.object({
  companyName: z.string()
    .min(2, 'Le nom doit contenir au moins 2 caractères')
    .max(100, 'Le nom ne peut pas dépasser 100 caractères')
    .regex(/^[a-zA-Z0-9\s\-&.,'À-ÿ]+$/, 'Caractères invalides détectés'),
});

type FormData = z.infer<typeof formSchema>;

export function LetterGenerator() {
  const t = useTranslations('letters');
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const _tValidation = useTranslations('validation');
  const [letters, setLetters] = useState<GeneratedLetters | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [progress, setProgress] = useState(0);

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<FormData>({
    resolver: zodResolver(formSchema),
  });

  const onSubmit = async (data: FormData) => {
    setIsGenerating(true);
    setError(null);
    setProgress(0);

    try {
      const response = await lettersApi.generateAndWait(
        { company_name: data.companyName },
        (p) => setProgress(p)
      );

      setLetters(response);
      setProgress(100);

      // Sauvegarder dans localStorage (historique)
      saveToHistory(response);
    } catch (err: any) {
      handleError(err);
    } finally {
      setIsGenerating(false);
    }
  };

  const handleError = (err: any) => {
    const status = err.statusCode;
    const message = err.message;

    switch (status) {
      case 403:
        setError(t('errors.accessDenied'));
        break;
      case 429:
        setError(t('errors.rateLimit'));
        break;
      case 500:
        setError(t('errors.serverError'));
        break;
      default:
        setError(message || t('errors.generic'));
    }
  };

  const saveToHistory = (data: GeneratedLetters) => {
    try {
      const history = JSON.parse(localStorage.getItem('letters_history') || '[]');
      history.unshift({
        id: data.id,
        companyName: data.companyName,
        createdAt: data.createdAt,
      });
      // Garder seulement les 10 dernières
      localStorage.setItem('letters_history', JSON.stringify(history.slice(0, 10)));
    } catch (e) {
      console.error('Failed to save to history:', e);
    }
  };

  const handleReset = () => {
    setLetters(null);
    setError(null);
    setProgress(0);
    reset();
  };

  return (
    <div className="max-w-7xl mx-auto">
      {/* Form */}
      {!letters && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="max-w-2xl mx-auto"
        >
          <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-8 border border-slate-200 dark:border-slate-700">
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
              {/* Input */}
              <div>
                <label
                  htmlFor="companyName"
                  className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
                >
                  {t('form.companyName')}
                </label>
                <input
                  {...register('companyName')}
                  id="companyName"
                  type="text"
                  placeholder={t('form.placeholder')}
                  disabled={isGenerating}
                  className="w-full px-4 py-3 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400 focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                />
                {errors.companyName && (
                  <p className="mt-2 text-sm text-red-600 dark:text-red-400">
                    {errors.companyName.message}
                  </p>
                )}
              </div>

              {/* Submit Button */}
              <button
                type="submit"
                disabled={isGenerating}
                className="w-full bg-gradient-to-r from-blue-600 to-purple-600 text-white px-6 py-4 rounded-lg font-medium hover:from-blue-700 hover:to-purple-700 transition-all duration-200 shadow-lg hover:shadow-xl disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
              >
                {isGenerating ? (
                  <>
                    <Loader2 className="w-5 h-5 animate-spin" />
                    {t('form.generating')}
                  </>
                ) : (
                  <>
                    <Sparkles className="w-5 h-5" />
                    {t('form.submit')}
                  </>
                )}
              </button>

              {/* Progress Bar */}
              {isGenerating && (
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-slate-600 dark:text-slate-400">
                      {progress < 30 && t('progress.analyzing')}
                      {progress >= 30 && progress < 60 && t('progress.writingMotivation')}
                      {progress >= 60 && progress < 90 && t('progress.writingAnti')}
                      {progress >= 90 && t('progress.finalizing')}
                    </span>
                    <span className="font-medium text-blue-600 dark:text-blue-400">
                      {progress}%
                    </span>
                  </div>
                  <div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-2 overflow-hidden">
                    <motion.div
                      initial={{ width: 0 }}
                      animate={{ width: `${progress}%` }}
                      className="bg-gradient-to-r from-blue-500 to-purple-500 h-full"
                    />
                  </div>
                </div>
              )}

              {/* Error Display */}
              {error && (
                <motion.div
                  initial={{ opacity: 0, scale: 0.95 }}
                  animate={{ opacity: 1, scale: 1 }}
                  className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4"
                >
                  <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
                </motion.div>
              )}

              {/* Info */}
              <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                <p className="text-xs text-blue-800 dark:text-blue-200">
                  {t('form.info')}
                </p>
              </div>
            </form>
          </div>
        </motion.div>
      )}

      {/* Preview */}
      {letters && (
        <LetterPreview letters={letters} onReset={handleReset} />
      )}
    </div>
  );
}
