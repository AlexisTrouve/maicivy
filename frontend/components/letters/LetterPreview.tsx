'use client';

import { useState } from 'react';
import { motion } from 'framer-motion';
import { Download, RotateCcw, Copy, Check } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { lettersApi } from '@/lib/api';
import type { GeneratedLetters } from '@/lib/types';

interface LetterPreviewProps {
  letters: GeneratedLetters;
  onReset: () => void;
}

export function LetterPreview({ letters, onReset }: LetterPreviewProps) {
  const t = useTranslations('letters.preview');
  const [downloadingMotivation, setDownloadingMotivation] = useState(false);
  const [downloadingAnti, setDownloadingAnti] = useState(false);
  const [downloadingBoth, setDownloadingBoth] = useState(false);
  const [copiedMotivation, setCopiedMotivation] = useState(false);
  const [copiedAnti, setCopiedAnti] = useState(false);

  const downloadPDF = async (type: 'motivation' | 'anti' | 'both') => {
    const setLoading = {
      motivation: setDownloadingMotivation,
      anti: setDownloadingAnti,
      both: setDownloadingBoth,
    }[type];

    setLoading(true);

    try {
      const blob = await lettersApi.downloadPDF(letters.id, type);

      // Create download link
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `lettre-${letters.companyName}-${type}.pdf`);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error('PDF download failed:', error);
      alert(t('downloadError'));
    } finally {
      setLoading(false);
    }
  };

  const copyToClipboard = async (text: string, type: 'motivation' | 'anti') => {
    try {
      await navigator.clipboard.writeText(text);
      if (type === 'motivation') {
        setCopiedMotivation(true);
        setTimeout(() => setCopiedMotivation(false), 2000);
      } else {
        setCopiedAnti(true);
        setTimeout(() => setCopiedAnti(false), 2000);
      }
    } catch (error) {
      console.error('Copy failed:', error);
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="space-y-6"
    >
      {/* Header avec actions */}
      <div className="bg-white dark:bg-slate-800 rounded-xl shadow-lg p-6 border border-slate-200 dark:border-slate-700">
        <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
          <div>
            <h2 className="text-2xl font-bold text-slate-900 dark:text-white mb-1">
              {t('title', { company: letters.companyName })}
            </h2>
            {letters.companyInfo?.industry && (
              <p className="text-sm text-slate-600 dark:text-slate-400">
                {t('sector')} {letters.companyInfo.industry}
              </p>
            )}
          </div>

          <div className="flex flex-wrap gap-2">
            <button
              onClick={() => downloadPDF('both')}
              disabled={downloadingBoth}
              className="inline-flex items-center gap-2 bg-gradient-to-r from-blue-600 to-purple-600 text-white px-4 py-2 rounded-lg font-medium hover:from-blue-700 hover:to-purple-700 transition-all disabled:opacity-50 text-sm"
            >
              {downloadingBoth ? (
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
              ) : (
                <Download className="w-4 h-4" />
              )}
              {t('downloadDual')}
            </button>

            <button
              onClick={onReset}
              className="inline-flex items-center gap-2 bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300 px-4 py-2 rounded-lg font-medium hover:bg-slate-300 dark:hover:bg-slate-600 transition-all text-sm"
            >
              <RotateCcw className="w-4 h-4" />
              {t('newGeneration')}
            </button>
          </div>
        </div>
      </div>

      {/* Dual Display */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Lettre de Motivation */}
        <motion.div
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.1 }}
          className="bg-white dark:bg-slate-800 rounded-xl shadow-lg border border-slate-200 dark:border-slate-700 overflow-hidden"
        >
          {/* Header */}
          <div className="bg-gradient-to-r from-green-500 to-emerald-500 p-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <span className="text-2xl">✅</span>
                {t('motivation')}
              </h3>
              <div className="flex gap-2">
                <button
                  onClick={() => copyToClipboard(letters.motivationLetter, 'motivation')}
                  className="p-2 hover:bg-white/20 rounded-lg transition-colors"
                  title={t('copyText')}
                >
                  {copiedMotivation ? (
                    <Check className="w-4 h-4 text-white" />
                  ) : (
                    <Copy className="w-4 h-4 text-white" />
                  )}
                </button>
                <button
                  onClick={() => downloadPDF('motivation')}
                  disabled={downloadingMotivation}
                  className="p-2 hover:bg-white/20 rounded-lg transition-colors disabled:opacity-50"
                  title={t('downloadPDF')}
                >
                  {downloadingMotivation ? (
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  ) : (
                    <Download className="w-4 h-4 text-white" />
                  )}
                </button>
              </div>
            </div>
          </div>

          {/* Content */}
          <div className="p-6 max-h-[600px] overflow-y-auto">
            <div className="prose prose-slate dark:prose-invert max-w-none">
              <div className="whitespace-pre-wrap text-slate-700 dark:text-slate-300">
                {letters.motivationLetter}
              </div>
            </div>
          </div>
        </motion.div>

        {/* Lettre d'Anti-Motivation */}
        <motion.div
          initial={{ opacity: 0, x: 20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.2 }}
          className="bg-white dark:bg-slate-800 rounded-xl shadow-lg border border-slate-200 dark:border-slate-700 overflow-hidden"
        >
          {/* Header */}
          <div className="bg-gradient-to-r from-orange-500 to-red-500 p-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <span className="text-2xl">❌</span>
                {t('antiMotivation')}
              </h3>
              <div className="flex gap-2">
                <button
                  onClick={() => copyToClipboard(letters.antiMotivationLetter, 'anti')}
                  className="p-2 hover:bg-white/20 rounded-lg transition-colors"
                  title={t('copyText')}
                >
                  {copiedAnti ? (
                    <Check className="w-4 h-4 text-white" />
                  ) : (
                    <Copy className="w-4 h-4 text-white" />
                  )}
                </button>
                <button
                  onClick={() => downloadPDF('anti')}
                  disabled={downloadingAnti}
                  className="p-2 hover:bg-white/20 rounded-lg transition-colors disabled:opacity-50"
                  title={t('downloadPDF')}
                >
                  {downloadingAnti ? (
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  ) : (
                    <Download className="w-4 h-4 text-white" />
                  )}
                </button>
              </div>
            </div>
          </div>

          {/* Content */}
          <div className="p-6 max-h-[600px] overflow-y-auto">
            <div className="prose prose-slate dark:prose-invert max-w-none">
              <div className="whitespace-pre-wrap text-slate-700 dark:text-slate-300">
                {letters.antiMotivationLetter}
              </div>
            </div>
          </div>
        </motion.div>
      </div>

      {/* Info Footer */}
      <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-4">
        <p className="text-sm text-amber-800 dark:text-amber-200">
          {t('warning')}
        </p>
      </div>
    </motion.div>
  );
}
