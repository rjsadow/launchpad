import { useEffect } from 'react';
import { useAuth } from '../context/AuthContext';

export function LoginPage() {
  const { login, isAuthenticated } = useAuth();

  useEffect(() => {
    if (!isAuthenticated) {
      login();
    }
  }, [isAuthenticated, login]);

  return (
    <div className="flex justify-center items-center h-64">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-primary mx-auto"></div>
        <p className="mt-4 text-gray-600 dark:text-gray-400">Redirecting to login...</p>
      </div>
    </div>
  );
}
