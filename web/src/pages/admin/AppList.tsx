import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import type { Application } from '../../types';
import { fetchAllApps, deleteApp } from '../../api/client';

export function AppList() {
  const [apps, setApps] = useState<Application[]>([]);
  const [loading, setLoading] = useState(true);
  const [deleting, setDeleting] = useState<string | null>(null);

  useEffect(() => {
    loadApps();
  }, []);

  const loadApps = async () => {
    try {
      const data = await fetchAllApps();
      setApps(data);
    } catch (err) {
      console.error('Failed to load apps:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this application?')) {
      return;
    }

    setDeleting(id);
    try {
      await deleteApp(id);
      setApps((prev) => prev.filter((app) => app.id !== id));
    } catch (err) {
      console.error('Failed to delete app:', err);
      alert('Failed to delete application');
    } finally {
      setDeleting(null);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-primary"></div>
      </div>
    );
  }

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead className="bg-gray-50 dark:bg-gray-900">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Application
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Category
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Roles
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Status
            </th>
            <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
          {apps.map((app) => (
            <tr key={app.id} className="hover:bg-gray-50 dark:hover:bg-gray-750">
              <td className="px-6 py-4 whitespace-nowrap">
                <div className="flex items-center">
                  <img
                    src={app.icon}
                    alt=""
                    className="w-8 h-8 rounded"
                    onError={(e) => {
                      (e.target as HTMLImageElement).src =
                        'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="%23398D9B"><rect width="24" height="24" rx="4"/></svg>';
                    }}
                  />
                  <div className="ml-4">
                    <div className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {app.name}
                    </div>
                    <div className="text-sm text-gray-500 dark:text-gray-400 truncate max-w-xs">
                      {app.url}
                    </div>
                  </div>
                </div>
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {app.category}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {app.roles && app.roles.length > 0 ? (
                  <div className="flex flex-wrap gap-1">
                    {app.roles.map((role) => (
                      <span
                        key={role}
                        className="px-2 py-1 text-xs bg-gray-200 dark:bg-gray-700 rounded"
                      >
                        {role}
                      </span>
                    ))}
                  </div>
                ) : (
                  <span className="text-gray-400">All users</span>
                )}
              </td>
              <td className="px-6 py-4 whitespace-nowrap">
                <span
                  className={`px-2 py-1 text-xs rounded-full ${
                    app.enabled
                      ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                      : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                  }`}
                >
                  {app.enabled ? 'Enabled' : 'Disabled'}
                </span>
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <Link
                  to={`/admin/edit/${app.id}`}
                  className="text-brand-primary hover:text-brand-secondary mr-4"
                >
                  Edit
                </Link>
                <button
                  onClick={() => handleDelete(app.id)}
                  disabled={deleting === app.id}
                  className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300 disabled:opacity-50"
                >
                  {deleting === app.id ? 'Deleting...' : 'Delete'}
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {apps.length === 0 && (
        <div className="text-center py-12">
          <p className="text-gray-500 dark:text-gray-400">No applications found.</p>
          <Link
            to="/admin/new"
            className="mt-4 inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-brand-primary hover:bg-brand-secondary"
          >
            Add First Application
          </Link>
        </div>
      )}
    </div>
  );
}
