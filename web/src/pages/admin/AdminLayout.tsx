import { Outlet, Link, useLocation } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';

export function AdminLayout() {
  const { isAdmin } = useAuth();
  const location = useLocation();

  if (!isAdmin) {
    return (
      <div className="text-center py-12">
        <svg
          className="mx-auto h-12 w-12 text-red-500"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          />
        </svg>
        <h3 className="mt-2 text-lg font-medium text-gray-900 dark:text-gray-100">
          Access Denied
        </h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          You do not have permission to access the admin panel.
        </p>
        <Link
          to="/"
          className="mt-4 inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-brand-primary hover:bg-brand-secondary"
        >
          Return to Dashboard
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          Admin Panel
        </h2>
        <nav className="flex gap-2">
          <Link
            to="/admin"
            className={`px-4 py-2 rounded-lg transition-colors ${
              location.pathname === '/admin'
                ? 'bg-brand-primary text-white'
                : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
            }`}
          >
            Applications
          </Link>
          <Link
            to="/admin/new"
            className={`px-4 py-2 rounded-lg transition-colors ${
              location.pathname === '/admin/new'
                ? 'bg-brand-primary text-white'
                : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
            }`}
          >
            Add New
          </Link>
        </nav>
      </div>

      <Outlet />
    </div>
  );
}
