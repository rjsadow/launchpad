import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import type { Application } from '../../types';
import { fetchAllApps, createApp, updateApp } from '../../api/client';

export function AppForm() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const isEditing = !!id;

  const [loading, setLoading] = useState(isEditing);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [formData, setFormData] = useState<Partial<Application>>({
    id: '',
    name: '',
    description: '',
    url: '',
    icon: '',
    category: '',
    roles: [],
    enabled: true,
    sortOrder: 0,
  });

  const [rolesInput, setRolesInput] = useState('');

  useEffect(() => {
    if (isEditing) {
      fetchAllApps()
        .then((apps) => {
          const app = apps.find((a) => a.id === id);
          if (app) {
            setFormData(app);
            setRolesInput(app.roles?.join(', ') || '');
          } else {
            setError('Application not found');
          }
        })
        .catch((err) => {
          console.error('Failed to load app:', err);
          setError('Failed to load application');
        })
        .finally(() => setLoading(false));
    }
  }, [id, isEditing]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSaving(true);

    // Parse roles from comma-separated input
    const roles = rolesInput
      .split(',')
      .map((r) => r.trim())
      .filter((r) => r.length > 0);

    const appData: Application = {
      ...formData,
      roles,
    } as Application;

    try {
      if (isEditing) {
        await updateApp(appData);
      } else {
        await createApp(appData);
      }
      navigate('/admin');
    } catch (err) {
      console.error('Failed to save app:', err);
      setError(err instanceof Error ? err.message : 'Failed to save application');
    } finally {
      setSaving(false);
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
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
      <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-6">
        {isEditing ? 'Edit Application' : 'Add New Application'}
      </h3>

      {error && (
        <div className="mb-4 p-4 bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200 rounded-lg">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              ID *
            </label>
            <input
              type="text"
              required
              disabled={isEditing}
              value={formData.id}
              onChange={(e) => setFormData({ ...formData, id: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-brand-primary focus:border-transparent disabled:opacity-50"
              placeholder="unique-app-id"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Name *
            </label>
            <input
              type="text"
              required
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-brand-primary focus:border-transparent"
              placeholder="Application Name"
            />
          </div>

          <div className="md:col-span-2">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Description
            </label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              rows={2}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-brand-primary focus:border-transparent"
              placeholder="Brief description of the application"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              URL *
            </label>
            <input
              type="url"
              required
              value={formData.url}
              onChange={(e) => setFormData({ ...formData, url: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-brand-primary focus:border-transparent"
              placeholder="https://app.example.com"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Icon URL
            </label>
            <input
              type="url"
              value={formData.icon}
              onChange={(e) => setFormData({ ...formData, icon: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-brand-primary focus:border-transparent"
              placeholder="https://example.com/icon.png"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Category *
            </label>
            <input
              type="text"
              required
              value={formData.category}
              onChange={(e) => setFormData({ ...formData, category: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-brand-primary focus:border-transparent"
              placeholder="Development, Monitoring, etc."
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Sort Order
            </label>
            <input
              type="number"
              value={formData.sortOrder}
              onChange={(e) => setFormData({ ...formData, sortOrder: parseInt(e.target.value) || 0 })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-brand-primary focus:border-transparent"
            />
          </div>

          <div className="md:col-span-2">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Required Roles (comma-separated)
            </label>
            <input
              type="text"
              value={rolesInput}
              onChange={(e) => setRolesInput(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-brand-primary focus:border-transparent"
              placeholder="admin, developer, team-lead (leave empty for all users)"
            />
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Leave empty to make this app visible to all users
            </p>
          </div>

          <div className="md:col-span-2">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={formData.enabled}
                onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                className="w-4 h-4 text-brand-primary border-gray-300 rounded focus:ring-brand-primary"
              />
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                Enabled
              </span>
            </label>
          </div>
        </div>

        <div className="flex justify-end gap-4">
          <button
            type="button"
            onClick={() => navigate('/admin')}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={saving}
            className="px-4 py-2 bg-brand-primary text-white rounded-lg hover:bg-brand-secondary transition-colors disabled:opacity-50"
          >
            {saving ? 'Saving...' : isEditing ? 'Update Application' : 'Create Application'}
          </button>
        </div>
      </form>
    </div>
  );
}
