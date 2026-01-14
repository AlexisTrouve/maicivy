'use client';

import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { Filter, X, Calendar } from 'lucide-react';

interface TimelineFiltersProps {
  categories: string[];
  selectedCategory: string;
  selectedType: string;
  onFiltersChange: (filters: {
    category: string;
    type: string;
    period: { from: string; to: string } | null;
  }) => void;
  onReset: () => void;
}

const TimelineFilters: React.FC<TimelineFiltersProps> = ({
  categories,
  selectedCategory,
  selectedType,
  onFiltersChange,
  onReset,
}) => {
  const [category, setCategory] = useState<string>(selectedCategory);
  const [type, setType] = useState<string>(selectedType);
  const [showPeriodPicker, setShowPeriodPicker] = useState(false);
  const [fromDate, setFromDate] = useState<string>('');
  const [toDate, setToDate] = useState<string>('');

  // Appliquer les filtres
  const applyFilters = () => {
    const period =
      fromDate && toDate
        ? {
            from: fromDate,
            to: toDate,
          }
        : null;

    onFiltersChange({ category, type, period });
  };

  // Gérer le changement de catégorie
  const handleCategoryChange = (cat: string) => {
    const newCategory = cat === category ? '' : cat;
    setCategory(newCategory);
    onFiltersChange({
      category: newCategory,
      type,
      period: fromDate && toDate ? { from: fromDate, to: toDate } : null,
    });
  };

  // Gérer le changement de type
  const handleTypeChange = (newType: string) => {
    setType(newType);
    onFiltersChange({
      category,
      type: newType,
      period: fromDate && toDate ? { from: fromDate, to: toDate } : null,
    });
  };

  // Réinitialiser
  const handleReset = () => {
    setCategory('');
    setType('all');
    setFromDate('');
    setToDate('');
    setShowPeriodPicker(false);
    onReset();
  };

  // Vérifier si des filtres sont actifs
  const hasActiveFilters = category || type !== 'all' || fromDate || toDate;

  return (
    <div className="mb-8">
      <div className="flex items-center gap-2 mb-4">
        <Filter className="w-5 h-5 text-gray-600 dark:text-gray-400" />
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
          Filtres
        </h3>
        {hasActiveFilters && (
          <button
            onClick={handleReset}
            className="ml-auto flex items-center gap-1 px-3 py-1 text-sm text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 transition-colors"
          >
            <X className="w-4 h-4" />
            Réinitialiser
          </button>
        )}
      </div>

      {/* Filtres par type */}
      <div className="mb-4">
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Type :</p>
        <div className="flex flex-wrap gap-2">
          {[
            { value: 'all', label: 'Tous' },
            { value: 'experience', label: 'Expériences' },
            { value: 'project', label: 'Projets' },
          ].map((typeOption) => (
            <button
              key={typeOption.value}
              onClick={() => handleTypeChange(typeOption.value)}
              className={`px-4 py-2 rounded-full text-sm font-medium transition-all ${
                type === typeOption.value
                  ? 'bg-blue-500 text-white shadow-md'
                  : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
              }`}
            >
              {typeOption.label}
            </button>
          ))}
        </div>
      </div>

      {/* Filtres par catégorie */}
      <div className="mb-4">
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
          Catégorie :
        </p>
        <div className="flex flex-wrap gap-2">
          {categories.map((cat) => (
            <motion.button
              key={cat}
              onClick={() => handleCategoryChange(cat)}
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              className={`px-4 py-2 rounded-full text-sm font-medium transition-all ${
                category === cat
                  ? 'bg-purple-500 text-white shadow-md'
                  : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
              }`}
            >
              {cat}
            </motion.button>
          ))}
        </div>
      </div>

      {/* Filtre par période */}
      <div>
        <button
          onClick={() => setShowPeriodPicker(!showPeriodPicker)}
          className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
        >
          <Calendar className="w-4 h-4" />
          {fromDate && toDate
            ? `Période : ${fromDate} → ${toDate}`
            : 'Filtrer par période'}
        </button>

        {showPeriodPicker && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="mt-3 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg"
          >
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  De :
                </label>
                <input
                  type="date"
                  value={fromDate}
                  onChange={(e) => setFromDate(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  À :
                </label>
                <input
                  type="date"
                  value={toDate}
                  onChange={(e) => setToDate(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>

            <div className="flex gap-2 mt-4">
              <button
                onClick={applyFilters}
                disabled={!fromDate || !toDate}
                className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
              >
                Appliquer
              </button>
              <button
                onClick={() => {
                  setFromDate('');
                  setToDate('');
                  setShowPeriodPicker(false);
                  applyFilters();
                }}
                className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
              >
                Effacer
              </button>
            </div>
          </motion.div>
        )}
      </div>
    </div>
  );
};

export default TimelineFilters;
