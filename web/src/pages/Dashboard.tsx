import { useState, useEffect, useRef, useCallback } from 'react';
import type { Application } from '../types';
import { fetchApps } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { AppCard } from '../components/AppCard';

interface DashboardProps {
  search: string;
}

export function Dashboard({ search }: DashboardProps) {
  const { isAuthenticated } = useAuth();
  const [apps, setApps] = useState<Application[]>([]);
  const [loading, setLoading] = useState(true);
  const [collapsedCategories, setCollapsedCategories] = useState<Set<string>>(() => {
    const stored = localStorage.getItem('launchpad-collapsed');
    return stored ? new Set(JSON.parse(stored)) : new Set();
  });
  const [focusedIndex, setFocusedIndex] = useState(-1);
  const appRefs = useRef<(HTMLAnchorElement | null)[]>([]);

  useEffect(() => {
    fetchApps()
      .then(setApps)
      .catch((err) => console.error('Failed to load apps:', err))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    localStorage.setItem('launchpad-collapsed', JSON.stringify([...collapsedCategories]));
  }, [collapsedCategories]);

  const filteredApps = apps.filter(
    (app) =>
      app.name.toLowerCase().includes(search.toLowerCase()) ||
      app.description.toLowerCase().includes(search.toLowerCase()) ||
      app.category.toLowerCase().includes(search.toLowerCase())
  );

  const categories = [...new Set(filteredApps.map((app) => app.category))];

  const visibleApps = filteredApps.filter(
    (app) => !collapsedCategories.has(app.category)
  );

  const toggleCategory = (category: string) => {
    setCollapsedCategories((prev) => {
      const next = new Set(prev);
      if (next.has(category)) {
        next.delete(category);
      } else {
        next.add(category);
      }
      return next;
    });
  };

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (visibleApps.length === 0) return;

      const columnsMap: Record<string, number> = {
        xl: 4,
        lg: 3,
        sm: 2,
        default: 1,
      };

      const getColumns = () => {
        if (window.innerWidth >= 1280) return columnsMap.xl;
        if (window.innerWidth >= 1024) return columnsMap.lg;
        if (window.innerWidth >= 640) return columnsMap.sm;
        return columnsMap.default;
      };

      const columns = getColumns();

      switch (e.key) {
        case 'ArrowRight':
          e.preventDefault();
          setFocusedIndex((prev) =>
            prev < visibleApps.length - 1 ? prev + 1 : prev
          );
          break;
        case 'ArrowLeft':
          e.preventDefault();
          setFocusedIndex((prev) => (prev > 0 ? prev - 1 : prev));
          break;
        case 'ArrowDown':
          e.preventDefault();
          setFocusedIndex((prev) =>
            prev + columns < visibleApps.length ? prev + columns : prev
          );
          break;
        case 'ArrowUp':
          e.preventDefault();
          setFocusedIndex((prev) => (prev - columns >= 0 ? prev - columns : prev));
          break;
        case 'Enter':
          if (focusedIndex >= 0 && focusedIndex < visibleApps.length) {
            e.preventDefault();
            window.open(visibleApps[focusedIndex].url, '_blank', 'noopener,noreferrer');
          }
          break;
        case 'Escape':
          setFocusedIndex(-1);
          break;
      }
    },
    [visibleApps, focusedIndex]
  );

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  useEffect(() => {
    if (focusedIndex >= 0 && appRefs.current[focusedIndex]) {
      appRefs.current[focusedIndex]?.focus();
    }
  }, [focusedIndex]);

  useEffect(() => {
    setFocusedIndex(-1);
  }, [search]);

  const handleFavoriteToggle = (appId: string, newState: boolean) => {
    setApps((prev) =>
      prev.map((app) =>
        app.id === appId ? { ...app, isFavorite: newState } : app
      )
    );
  };

  let appIndex = 0;

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-primary"></div>
      </div>
    );
  }

  if (filteredApps.length === 0) {
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
            d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
          No applications found
        </h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Try adjusting your search terms.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <p className="text-xs text-gray-500 dark:text-gray-400">
        Use arrow keys to navigate, Enter to launch, Escape to clear focus
      </p>

      {categories.map((category) => {
        const categoryApps = filteredApps.filter((app) => app.category === category);
        const isCollapsed = collapsedCategories.has(category);

        return (
          <div
            key={category}
            className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden"
          >
            <button
              onClick={() => toggleCategory(category)}
              className="w-full flex items-center justify-between px-4 py-3 bg-gray-50 dark:bg-gray-750 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            >
              <div className="flex items-center gap-2">
                <svg
                  className={`w-4 h-4 text-gray-500 dark:text-gray-400 transition-transform ${
                    isCollapsed ? '' : 'rotate-90'
                  }`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
                <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{category}</h2>
                <span className="text-sm text-gray-500 dark:text-gray-400">({categoryApps.length})</span>
              </div>
            </button>

            {!isCollapsed && (
              <div className="p-4">
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                  {categoryApps.map((app) => {
                    const currentIndex = appIndex++;
                    return (
                      <AppCard
                        key={app.id}
                        ref={(el) => {
                          appRefs.current[currentIndex] = el;
                        }}
                        app={app}
                        isFocused={focusedIndex === currentIndex}
                        showFavorite={isAuthenticated}
                        onFocus={() => setFocusedIndex(currentIndex)}
                        onClick={() => setFocusedIndex(currentIndex)}
                        onFavoriteToggle={handleFavoriteToggle}
                      />
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
