import { forwardRef } from 'react';
import type { Application } from '../types';
import { FavoriteButton } from './FavoriteButton';

interface AppCardProps {
  app: Application;
  isFocused: boolean;
  showFavorite?: boolean;
  onFocus?: () => void;
  onClick?: () => void;
  onFavoriteToggle?: (appId: string, newState: boolean) => void;
}

export const AppCard = forwardRef<HTMLAnchorElement, AppCardProps>(
  ({ app, isFocused, showFavorite = false, onFocus, onClick, onFavoriteToggle }, ref) => {
    return (
      <a
        ref={ref}
        href={app.url}
        target="_blank"
        rel="noopener noreferrer"
        tabIndex={isFocused ? 0 : -1}
        onClick={onClick}
        onFocus={onFocus}
        className={`group bg-gray-50 dark:bg-gray-700 rounded-lg border p-4 hover:shadow-md transition-all duration-200 ${
          isFocused
            ? 'ring-2 ring-brand-primary border-brand-primary'
            : 'border-gray-200 dark:border-gray-600 hover:border-brand-secondary'
        }`}
      >
        <div className="flex items-start gap-3">
          <div className="flex-shrink-0 w-12 h-12 bg-white dark:bg-gray-600 rounded-lg flex items-center justify-center overflow-hidden">
            <img
              src={app.icon}
              alt={`${app.name} icon`}
              className="w-8 h-8 object-contain"
              onError={(e) => {
                (e.target as HTMLImageElement).src =
                  'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="%23398D9B"><rect width="24" height="24" rx="4"/><text x="12" y="16" text-anchor="middle" fill="white" font-size="12">' +
                  app.name.charAt(0) +
                  '</text></svg>';
              }}
            />
          </div>
          <div className="flex-1 min-w-0">
            <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 group-hover:text-brand-primary truncate">
              {app.name}
            </h3>
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400 line-clamp-2">
              {app.description}
            </p>
          </div>
          <div className="flex flex-col items-center gap-1">
            {showFavorite && (
              <FavoriteButton
                appId={app.id}
                isFavorite={app.isFavorite ?? false}
                onToggle={(newState) => onFavoriteToggle?.(app.id, newState)}
              />
            )}
            <svg
              className="w-4 h-4 text-gray-400 group-hover:text-brand-primary flex-shrink-0"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
              />
            </svg>
          </div>
        </div>
      </a>
    );
  }
);

AppCard.displayName = 'AppCard';
