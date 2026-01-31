import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

interface HeaderProps {
  search: string;
  onSearchChange: (value: string) => void;
  darkMode: boolean;
  onDarkModeToggle: () => void;
}

export function Header({ search, onSearchChange, darkMode, onDarkModeToggle }: HeaderProps) {
  const { user, isAuthenticated, isAdmin, login, logout } = useAuth();

  return (
    <header className="bg-brand-primary text-white shadow-lg">
      <div className="max-w-7xl mx-auto px-4 py-6 sm:px-6 lg:px-8">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div className="flex items-center gap-3">
            <Link to="/" className="flex items-center gap-3">
              <div className="w-10 h-10 bg-white rounded-lg flex items-center justify-center">
                <svg
                  className="w-6 h-6 text-brand-primary"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"
                  />
                </svg>
              </div>
              <h1 className="text-2xl font-bold">Launchpad</h1>
            </Link>
          </div>

          <div className="flex items-center gap-3">
            <div className="relative">
              <input
                type="text"
                placeholder="Search applications..."
                value={search}
                onChange={(e) => onSearchChange(e.target.value)}
                className="w-full sm:w-80 px-4 py-2 pl-10 rounded-lg text-gray-900 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-brand-secondary"
              />
              <svg
                className="absolute left-3 top-2.5 w-5 h-5 text-gray-400"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
            </div>

            {/* Nav links */}
            {isAuthenticated && (
              <nav className="hidden sm:flex items-center gap-2">
                <Link
                  to="/"
                  className="px-3 py-2 rounded-lg hover:bg-white/10 transition-colors"
                >
                  Apps
                </Link>
                <Link
                  to="/favorites"
                  className="px-3 py-2 rounded-lg hover:bg-white/10 transition-colors"
                >
                  Favorites
                </Link>
                {isAdmin && (
                  <Link
                    to="/admin"
                    className="px-3 py-2 rounded-lg hover:bg-white/10 transition-colors"
                  >
                    Admin
                  </Link>
                )}
              </nav>
            )}

            {/* Dark mode toggle */}
            <button
              onClick={onDarkModeToggle}
              className="p-2 rounded-lg bg-white/10 hover:bg-white/20 transition-colors"
              aria-label={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}
            >
              {darkMode ? (
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
                  />
                </svg>
              ) : (
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
                  />
                </svg>
              )}
            </button>

            {/* User menu */}
            {isAuthenticated ? (
              <div className="flex items-center gap-2">
                <span className="hidden sm:inline text-sm">{user?.name || user?.email}</span>
                <button
                  onClick={logout}
                  className="p-2 rounded-lg bg-white/10 hover:bg-white/20 transition-colors"
                  aria-label="Logout"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
                    />
                  </svg>
                </button>
              </div>
            ) : (
              <button
                onClick={login}
                className="px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 transition-colors"
              >
                Login
              </button>
            )}
          </div>
        </div>
      </div>
    </header>
  );
}
