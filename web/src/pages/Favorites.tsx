import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import type { Application } from '../types';
import { fetchFavorites } from '../api/client';
import { AppCard } from '../components/AppCard';

export function Favorites() {
  const [apps, setApps] = useState<Application[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchFavorites()
      .then(setApps)
      .catch((err) => console.error('Failed to load favorites:', err))
      .finally(() => setLoading(false));
  }, []);

  const handleFavoriteToggle = (appId: string, newState: boolean) => {
    if (!newState) {
      // Remove from list when unfavorited
      setApps((prev) => prev.filter((app) => app.id !== appId));
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-primary"></div>
      </div>
    );
  }

  if (apps.length === 0) {
    return (
      <div className="text-center py-12">
        <svg
          className="mx-auto h-12 w-12 text-gray-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
          />
        </svg>
        <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
          No favorites yet
        </h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Star your favorite applications to see them here.
        </p>
        <Link
          to="/"
          className="mt-4 inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-brand-primary hover:bg-brand-secondary"
        >
          Browse Applications
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          Your Favorites
        </h2>
        <span className="text-sm text-gray-500 dark:text-gray-400">
          {apps.length} application{apps.length !== 1 ? 's' : ''}
        </span>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {apps.map((app) => (
          <AppCard
            key={app.id}
            app={app}
            isFocused={false}
            showFavorite={true}
            onFavoriteToggle={handleFavoriteToggle}
          />
        ))}
      </div>
    </div>
  );
}
