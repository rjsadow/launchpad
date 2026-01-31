import { useState, useEffect } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from './context/AuthContext';
import { Header } from './components/Header';
import { Dashboard } from './pages/Dashboard';
import { Favorites } from './pages/Favorites';
import { LoginPage } from './pages/LoginPage';
import { AdminLayout } from './pages/admin/AdminLayout';
import { AppList } from './pages/admin/AppList';
import { AppForm } from './pages/admin/AppForm';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-primary"></div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

function App() {
  const { loading } = useAuth();
  const [search, setSearch] = useState('');
  const [darkMode, setDarkMode] = useState(() => {
    const stored = localStorage.getItem('launchpad-theme');
    if (stored) return stored === 'dark';
    return window.matchMedia('(prefers-color-scheme: dark)').matches;
  });

  useEffect(() => {
    document.documentElement.classList.toggle('dark', darkMode);
    localStorage.setItem('launchpad-theme', darkMode ? 'dark' : 'light');
  }, [darkMode]);

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex justify-center items-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-primary"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 transition-colors">
      <Header
        search={search}
        onSearchChange={setSearch}
        darkMode={darkMode}
        onDarkModeToggle={() => setDarkMode(!darkMode)}
      />

      <main className="max-w-7xl mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <Routes>
          <Route path="/" element={<Dashboard search={search} />} />
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="/favorites"
            element={
              <ProtectedRoute>
                <Favorites />
              </ProtectedRoute>
            }
          />
          <Route
            path="/admin"
            element={
              <ProtectedRoute>
                <AdminLayout />
              </ProtectedRoute>
            }
          >
            <Route index element={<AppList />} />
            <Route path="new" element={<AppForm />} />
            <Route path="edit/:id" element={<AppForm />} />
          </Route>
        </Routes>
      </main>

      <footer className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 mt-auto">
        <div className="max-w-7xl mx-auto px-4 py-4 sm:px-6 lg:px-8">
          <p className="text-center text-sm text-gray-500 dark:text-gray-400">
            Launchpad — Your centralized application launcher
          </p>
        </div>
      </footer>
    </div>
  );
}

export default App;
