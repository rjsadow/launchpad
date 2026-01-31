import { useState, useEffect } from 'react';
import type { Application, ApplicationFormData, ValidationErrors, LaunchType } from '../types';

interface AdminPanelProps {
  apps: Application[];
  onAppsChange: () => void;
}

const emptyForm: ApplicationFormData = {
  id: '',
  name: '',
  description: '',
  url: '',
  icon: '',
  category: '',
  launch_type: 'url',
  container_image: '',
};

export function AdminPanel({ apps, onAppsChange }: AdminPanelProps) {
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [editingApp, setEditingApp] = useState<Application | null>(null);
  const [formData, setFormData] = useState<ApplicationFormData>(emptyForm);
  const [errors, setErrors] = useState<ValidationErrors>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<Application | null>(null);
  const [apiError, setApiError] = useState<string | null>(null);

  // Reset form when modal opens/closes
  useEffect(() => {
    if (isFormOpen && editingApp) {
      setFormData({
        id: editingApp.id,
        name: editingApp.name,
        description: editingApp.description,
        url: editingApp.url,
        icon: editingApp.icon,
        category: editingApp.category,
        launch_type: editingApp.launch_type,
        container_image: editingApp.container_image || '',
      });
    } else if (isFormOpen) {
      setFormData(emptyForm);
    }
    setErrors({});
    setApiError(null);
  }, [isFormOpen, editingApp]);

  const validateForm = (): boolean => {
    const newErrors: ValidationErrors = {};

    if (!formData.id.trim()) {
      newErrors.id = 'Application ID is required';
    } else if (!/^[a-z0-9-]+$/.test(formData.id)) {
      newErrors.id = 'ID must contain only lowercase letters, numbers, and hyphens';
    } else if (!editingApp && apps.some(app => app.id === formData.id)) {
      newErrors.id = 'An application with this ID already exists';
    }

    if (!formData.name.trim()) {
      newErrors.name = 'Application name is required';
    } else if (formData.name.length > 100) {
      newErrors.name = 'Name must be 100 characters or less';
    }

    if (!formData.url.trim()) {
      newErrors.url = 'URL is required';
    } else if (formData.launch_type === 'url') {
      try {
        new URL(formData.url);
      } catch {
        newErrors.url = 'Please enter a valid URL (e.g., https://example.com)';
      }
    }

    if (!formData.category.trim()) {
      newErrors.category = 'Category is required';
    }

    if (formData.launch_type === 'container' && !formData.container_image?.trim()) {
      newErrors.container_image = 'Container image is required for container apps';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);
    setApiError(null);

    try {
      const url = editingApp ? `/api/apps/${editingApp.id}` : '/api/apps';
      const method = editingApp ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...formData,
          icon: formData.icon || '/icons/default.svg',
        }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        // Try to parse as JSON, fall back to text
        try {
          const errorJson = JSON.parse(errorText);
          throw new Error(errorJson.error || errorText);
        } catch {
          throw new Error(errorText || `Failed to ${editingApp ? 'update' : 'create'} application`);
        }
      }

      setIsFormOpen(false);
      setEditingApp(null);
      onAppsChange();
    } catch (err) {
      setApiError(err instanceof Error ? err.message : 'An unexpected error occurred');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDelete = async (app: Application) => {
    setIsSubmitting(true);
    setApiError(null);

    try {
      const response = await fetch(`/api/apps/${app.id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || 'Failed to delete application');
      }

      setDeleteConfirm(null);
      onAppsChange();
    } catch (err) {
      setApiError(err instanceof Error ? err.message : 'An unexpected error occurred');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleInputChange = (field: keyof ApplicationFormData, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[field];
        return newErrors;
      });
    }
  };

  const categories = [...new Set(apps.map(app => app.category))];

  return (
    <div className="space-y-6">
      {/* Header with Add button */}
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
          Manage Applications
        </h2>
        <button
          onClick={() => {
            setEditingApp(null);
            setIsFormOpen(true);
          }}
          className="px-4 py-2 bg-brand-primary text-white rounded-lg hover:bg-brand-primary/90 transition-colors flex items-center gap-2"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add Application
        </button>
      </div>

      {/* Apps Table */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead className="bg-gray-50 dark:bg-gray-750">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Application
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Category
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Type
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {apps.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-6 py-8 text-center text-gray-500 dark:text-gray-400">
                  No applications found. Click "Add Application" to create one.
                </td>
              </tr>
            ) : (
              apps.map((app) => (
                <tr key={app.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-3">
                      <img
                        src={app.icon}
                        alt={`${app.name} icon`}
                        className="w-8 h-8 rounded object-contain bg-gray-100 dark:bg-gray-600"
                        onError={(e) => {
                          (e.target as HTMLImageElement).src =
                            'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="%23398D9B"><rect width="24" height="24" rx="4"/></svg>';
                        }}
                      />
                      <div>
                        <div className="font-medium text-gray-900 dark:text-gray-100">{app.name}</div>
                        <div className="text-sm text-gray-500 dark:text-gray-400 truncate max-w-xs">
                          {app.description}
                        </div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-900 dark:text-gray-100">
                    {app.category}
                  </td>
                  <td className="px-6 py-4">
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                      app.launch_type === 'container'
                        ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
                        : 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                    }`}>
                      {app.launch_type === 'container' ? 'Container' : 'URL'}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => {
                          setEditingApp(app);
                          setIsFormOpen(true);
                        }}
                        className="p-2 text-gray-500 hover:text-brand-primary hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                        title="Edit application"
                      >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                        </svg>
                      </button>
                      <button
                        onClick={() => setDeleteConfirm(app)}
                        className="p-2 text-gray-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
                        title="Delete application"
                      >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Create/Edit Modal */}
      {isFormOpen && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:p-0">
            <div
              className="fixed inset-0 bg-gray-500/75 dark:bg-gray-900/75 transition-opacity"
              onClick={() => !isSubmitting && setIsFormOpen(false)}
            />
            <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-lg w-full mx-auto z-10">
              <form onSubmit={handleSubmit}>
                <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                    {editingApp ? 'Edit Application' : 'Add New Application'}
                  </h3>
                </div>

                <div className="px-6 py-4 space-y-4 max-h-[60vh] overflow-y-auto">
                  {apiError && (
                    <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                      <p className="text-sm text-red-600 dark:text-red-400">{apiError}</p>
                    </div>
                  )}

                  {/* ID Field */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Application ID *
                    </label>
                    <input
                      type="text"
                      value={formData.id}
                      onChange={(e) => handleInputChange('id', e.target.value.toLowerCase())}
                      disabled={!!editingApp}
                      className={`w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-brand-primary focus:border-brand-primary dark:bg-gray-700 dark:text-gray-100 ${
                        errors.id ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'
                      } ${editingApp ? 'bg-gray-100 dark:bg-gray-600 cursor-not-allowed' : ''}`}
                      placeholder="my-app-id"
                    />
                    {errors.id && (
                      <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.id}</p>
                    )}
                    {!editingApp && (
                      <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                        Unique identifier (lowercase, numbers, hyphens only)
                      </p>
                    )}
                  </div>

                  {/* Name Field */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Name *
                    </label>
                    <input
                      type="text"
                      value={formData.name}
                      onChange={(e) => handleInputChange('name', e.target.value)}
                      className={`w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-brand-primary focus:border-brand-primary dark:bg-gray-700 dark:text-gray-100 ${
                        errors.name ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'
                      }`}
                      placeholder="My Application"
                    />
                    {errors.name && (
                      <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.name}</p>
                    )}
                  </div>

                  {/* Description Field */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Description
                    </label>
                    <textarea
                      value={formData.description}
                      onChange={(e) => handleInputChange('description', e.target.value)}
                      rows={2}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-brand-primary focus:border-brand-primary dark:bg-gray-700 dark:text-gray-100"
                      placeholder="Brief description of the application"
                    />
                  </div>

                  {/* Category Field */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Category *
                    </label>
                    <input
                      type="text"
                      value={formData.category}
                      onChange={(e) => handleInputChange('category', e.target.value)}
                      list="categories"
                      className={`w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-brand-primary focus:border-brand-primary dark:bg-gray-700 dark:text-gray-100 ${
                        errors.category ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'
                      }`}
                      placeholder="Development"
                    />
                    <datalist id="categories">
                      {categories.map((cat) => (
                        <option key={cat} value={cat} />
                      ))}
                    </datalist>
                    {errors.category && (
                      <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.category}</p>
                    )}
                  </div>

                  {/* Launch Type Field */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Launch Type *
                    </label>
                    <select
                      value={formData.launch_type}
                      onChange={(e) => handleInputChange('launch_type', e.target.value as LaunchType)}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-brand-primary focus:border-brand-primary dark:bg-gray-700 dark:text-gray-100"
                    >
                      <option value="url">URL (opens in new tab)</option>
                      <option value="container">Container (launches session)</option>
                    </select>
                  </div>

                  {/* URL Field */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      URL *
                    </label>
                    <input
                      type="text"
                      value={formData.url}
                      onChange={(e) => handleInputChange('url', e.target.value)}
                      className={`w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-brand-primary focus:border-brand-primary dark:bg-gray-700 dark:text-gray-100 ${
                        errors.url ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'
                      }`}
                      placeholder={formData.launch_type === 'container' ? '/app/path' : 'https://example.com'}
                    />
                    {errors.url && (
                      <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.url}</p>
                    )}
                  </div>

                  {/* Container Image Field (conditional) */}
                  {formData.launch_type === 'container' && (
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Container Image *
                      </label>
                      <input
                        type="text"
                        value={formData.container_image || ''}
                        onChange={(e) => handleInputChange('container_image', e.target.value)}
                        className={`w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-brand-primary focus:border-brand-primary dark:bg-gray-700 dark:text-gray-100 ${
                          errors.container_image ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'
                        }`}
                        placeholder="docker.io/myorg/myimage:latest"
                      />
                      {errors.container_image && (
                        <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.container_image}</p>
                      )}
                    </div>
                  )}

                  {/* Icon Field */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Icon URL
                    </label>
                    <input
                      type="text"
                      value={formData.icon}
                      onChange={(e) => handleInputChange('icon', e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-brand-primary focus:border-brand-primary dark:bg-gray-700 dark:text-gray-100"
                      placeholder="/icons/app.svg or https://example.com/icon.png"
                    />
                    <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      Leave empty for default icon
                    </p>
                  </div>
                </div>

                <div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
                  <button
                    type="button"
                    onClick={() => setIsFormOpen(false)}
                    disabled={isSubmitting}
                    className="px-4 py-2 text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors disabled:opacity-50"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={isSubmitting}
                    className="px-4 py-2 bg-brand-primary text-white rounded-lg hover:bg-brand-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
                  >
                    {isSubmitting && (
                      <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                      </svg>
                    )}
                    {editingApp ? 'Save Changes' : 'Create Application'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {deleteConfirm && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:p-0">
            <div
              className="fixed inset-0 bg-gray-500/75 dark:bg-gray-900/75 transition-opacity"
              onClick={() => !isSubmitting && setDeleteConfirm(null)}
            />
            <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-auto z-10 p-6">
              <div className="flex items-center gap-4 mb-4">
                <div className="w-12 h-12 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
                  <svg className="w-6 h-6 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                </div>
                <div className="text-left">
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Delete Application
                  </h3>
                  <p className="text-sm text-gray-500 dark:text-gray-400">
                    This action cannot be undone.
                  </p>
                </div>
              </div>

              <p className="text-gray-700 dark:text-gray-300 mb-6">
                Are you sure you want to delete <strong>{deleteConfirm.name}</strong>?
              </p>

              {apiError && (
                <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                  <p className="text-sm text-red-600 dark:text-red-400">{apiError}</p>
                </div>
              )}

              <div className="flex justify-end gap-3">
                <button
                  onClick={() => setDeleteConfirm(null)}
                  disabled={isSubmitting}
                  className="px-4 py-2 text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors disabled:opacity-50"
                >
                  Cancel
                </button>
                <button
                  onClick={() => handleDelete(deleteConfirm)}
                  disabled={isSubmitting}
                  className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 flex items-center gap-2"
                >
                  {isSubmitting && (
                    <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                    </svg>
                  )}
                  Delete
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
