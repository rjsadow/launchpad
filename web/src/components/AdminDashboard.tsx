import { useState, useEffect, useCallback } from 'react';

// Types for admin API responses
interface SystemMetrics {
  timestamp: string;
  uptime: string;
  active_sessions: number;
  total_apps: number;
  total_launches: number;
  recent_activity: ActivityEntry[];
  sessions_by_state: Record<string, number>;
}

interface ActivityEntry {
  timestamp: string;
  action: string;
  user: string;
  details: string;
}

interface SessionInfo {
  id: string;
  user_id: string;
  app_id: string;
  app_name: string;
  status: string;
  created_at: string;
  updated_at: string;
  duration: string;
}

interface HealthResponse {
  status: 'healthy' | 'degraded' | 'unhealthy';
  timestamp: string;
  version: string;
  components: Record<string, { status: string; message?: string; latency?: string }>;
}

interface DiagnosticsReport {
  timestamp: string;
  version: string;
  system: {
    go_version: string;
    goos: string;
    goarch: string;
    num_cpu: number;
    num_goroutine: number;
    hostname: string;
    start_time: string;
  };
  memory: {
    alloc_bytes: number;
    heap_alloc_bytes: number;
    sys_bytes: number;
    num_gc: number;
  };
  database: {
    app_count: number;
    session_count: number;
    audit_count: number;
  };
}

interface AdminDashboardProps {
  darkMode?: boolean; // Passed for consistency; dark mode is handled via Tailwind's dark: classes on <html>
  onClose: () => void;
}

export function AdminDashboard({ onClose }: AdminDashboardProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'sessions' | 'diagnostics'>('overview');
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null);
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [sessions, setSessions] = useState<SessionInfo[]>([]);
  const [diagnostics, setDiagnostics] = useState<DiagnosticsReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const [metricsRes, healthRes] = await Promise.all([
        fetch('/api/admin/metrics'),
        fetch('/api/health'),
      ]);

      if (!metricsRes.ok || !healthRes.ok) {
        throw new Error('Failed to fetch admin data');
      }

      const [metricsData, healthData] = await Promise.all([
        metricsRes.json(),
        healthRes.json(),
      ]);

      setMetrics(metricsData);
      setHealth(healthData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchSessions = useCallback(async () => {
    try {
      const res = await fetch('/api/admin/sessions');
      if (!res.ok) throw new Error('Failed to fetch sessions');
      const data = await res.json();
      setSessions(data);
    } catch (err) {
      console.error('Failed to fetch sessions:', err);
    }
  }, []);

  const fetchDiagnostics = useCallback(async () => {
    try {
      const res = await fetch('/api/admin/diagnostics');
      if (!res.ok) throw new Error('Failed to fetch diagnostics');
      const data = await res.json();
      setDiagnostics(data);
    } catch (err) {
      console.error('Failed to fetch diagnostics:', err);
    }
  }, []);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, [fetchData]);

  useEffect(() => {
    if (activeTab === 'sessions') {
      fetchSessions();
    } else if (activeTab === 'diagnostics') {
      fetchDiagnostics();
    }
  }, [activeTab, fetchSessions, fetchDiagnostics]);

  const downloadSupportBundle = async () => {
    try {
      const res = await fetch('/api/admin/support-bundle');
      if (!res.ok) throw new Error('Failed to download support bundle');
      const blob = await res.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `launchpad-support-bundle-${new Date().toISOString().slice(0, 10)}.zip`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      console.error('Failed to download support bundle:', err);
    }
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'healthy':
      case 'running':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'degraded':
      case 'creating':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      case 'unhealthy':
      case 'failed':
      case 'stopped':
      case 'expired':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200';
    }
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="min-h-screen bg-gray-100 dark:bg-gray-900">
        {/* Header */}
        <header className="bg-white dark:bg-gray-800 shadow">
          <div className="max-w-7xl mx-auto px-4 py-4 sm:px-6 lg:px-8">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <svg className="w-8 h-8 text-brand-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7m0 10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2h-2a2 2 0 00-2 2" />
                </svg>
                <h1 className="text-xl font-bold text-gray-900 dark:text-white">Admin Dashboard</h1>
              </div>
              <button
                onClick={onClose}
                className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            {/* Tabs */}
            <nav className="mt-4 flex space-x-4">
              {(['overview', 'sessions', 'diagnostics'] as const).map((tab) => (
                <button
                  key={tab}
                  onClick={() => setActiveTab(tab)}
                  className={`px-3 py-2 text-sm font-medium rounded-md ${
                    activeTab === tab
                      ? 'bg-brand-primary text-white'
                      : 'text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700'
                  }`}
                >
                  {tab.charAt(0).toUpperCase() + tab.slice(1)}
                </button>
              ))}
            </nav>
          </div>
        </header>

        {/* Main Content */}
        <main className="max-w-7xl mx-auto px-4 py-6 sm:px-6 lg:px-8">
          {loading && !metrics ? (
            <div className="flex justify-center items-center h-64">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-primary"></div>
            </div>
          ) : error ? (
            <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
              <p className="text-red-800 dark:text-red-200">{error}</p>
            </div>
          ) : (
            <>
              {/* Overview Tab */}
              {activeTab === 'overview' && metrics && health && (
                <div className="space-y-6">
                  {/* Health Status */}
                  <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">System Health</h2>
                    <div className="flex items-center gap-4 mb-4">
                      <span className={`px-3 py-1 rounded-full text-sm font-medium ${getStatusColor(health.status)}`}>
                        {health.status.toUpperCase()}
                      </span>
                      <span className="text-sm text-gray-500 dark:text-gray-400">
                        Version: {health.version}
                      </span>
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      {Object.entries(health.components).map(([name, comp]) => (
                        <div key={name} className="bg-gray-50 dark:bg-gray-700 rounded-lg p-3">
                          <div className="flex items-center justify-between">
                            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{name}</span>
                            <span className={`px-2 py-0.5 rounded text-xs ${getStatusColor(comp.status)}`}>
                              {comp.status}
                            </span>
                          </div>
                          {comp.latency && (
                            <span className="text-xs text-gray-500 dark:text-gray-400">{comp.latency}</span>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Stats Grid */}
                  <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                      <p className="text-sm text-gray-500 dark:text-gray-400">Uptime</p>
                      <p className="text-2xl font-bold text-gray-900 dark:text-white">{metrics.uptime}</p>
                    </div>
                    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                      <p className="text-sm text-gray-500 dark:text-gray-400">Active Sessions</p>
                      <p className="text-2xl font-bold text-gray-900 dark:text-white">{metrics.active_sessions}</p>
                    </div>
                    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                      <p className="text-sm text-gray-500 dark:text-gray-400">Total Apps</p>
                      <p className="text-2xl font-bold text-gray-900 dark:text-white">{metrics.total_apps}</p>
                    </div>
                    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                      <p className="text-sm text-gray-500 dark:text-gray-400">Total Launches</p>
                      <p className="text-2xl font-bold text-gray-900 dark:text-white">{metrics.total_launches}</p>
                    </div>
                  </div>

                  {/* Recent Activity */}
                  <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Recent Activity</h2>
                    <div className="overflow-x-auto">
                      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                        <thead>
                          <tr>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Time</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">User</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Action</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Details</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                          {metrics.recent_activity?.map((entry, i) => (
                            <tr key={i}>
                              <td className="px-4 py-2 text-sm text-gray-600 dark:text-gray-300">
                                {new Date(entry.timestamp).toLocaleString()}
                              </td>
                              <td className="px-4 py-2 text-sm text-gray-600 dark:text-gray-300">{entry.user}</td>
                              <td className="px-4 py-2 text-sm text-gray-600 dark:text-gray-300">{entry.action}</td>
                              <td className="px-4 py-2 text-sm text-gray-600 dark:text-gray-300">{entry.details}</td>
                            </tr>
                          ))}
                          {(!metrics.recent_activity || metrics.recent_activity.length === 0) && (
                            <tr>
                              <td colSpan={4} className="px-4 py-4 text-center text-sm text-gray-500 dark:text-gray-400">
                                No recent activity
                              </td>
                            </tr>
                          )}
                        </tbody>
                      </table>
                    </div>
                  </div>
                </div>
              )}

              {/* Sessions Tab */}
              {activeTab === 'sessions' && (
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                  <div className="flex items-center justify-between mb-4">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white">All Sessions</h2>
                    <button
                      onClick={fetchSessions}
                      className="px-3 py-1 text-sm bg-brand-primary text-white rounded hover:bg-brand-primary/90"
                    >
                      Refresh
                    </button>
                  </div>
                  <div className="overflow-x-auto">
                    <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                      <thead>
                        <tr>
                          <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">ID</th>
                          <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">User</th>
                          <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">App</th>
                          <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Status</th>
                          <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Duration</th>
                          <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Created</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                        {sessions.map((session) => (
                          <tr key={session.id}>
                            <td className="px-4 py-2 text-sm text-gray-600 dark:text-gray-300 font-mono">
                              {session.id.slice(0, 8)}...
                            </td>
                            <td className="px-4 py-2 text-sm text-gray-600 dark:text-gray-300">{session.user_id}</td>
                            <td className="px-4 py-2 text-sm text-gray-600 dark:text-gray-300">{session.app_name}</td>
                            <td className="px-4 py-2">
                              <span className={`px-2 py-0.5 rounded text-xs ${getStatusColor(session.status)}`}>
                                {session.status}
                              </span>
                            </td>
                            <td className="px-4 py-2 text-sm text-gray-600 dark:text-gray-300">{session.duration}</td>
                            <td className="px-4 py-2 text-sm text-gray-600 dark:text-gray-300">
                              {new Date(session.created_at).toLocaleString()}
                            </td>
                          </tr>
                        ))}
                        {sessions.length === 0 && (
                          <tr>
                            <td colSpan={6} className="px-4 py-4 text-center text-sm text-gray-500 dark:text-gray-400">
                              No sessions found
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              {/* Diagnostics Tab */}
              {activeTab === 'diagnostics' && (
                <div className="space-y-6">
                  {/* Support Bundle */}
                  <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Support Bundle</h2>
                    <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                      Download a diagnostic bundle for support purposes. This includes system info, configuration, and logs.
                    </p>
                    <button
                      onClick={downloadSupportBundle}
                      className="px-4 py-2 bg-brand-primary text-white rounded hover:bg-brand-primary/90"
                    >
                      Download Support Bundle
                    </button>
                  </div>

                  {/* System Info */}
                  {diagnostics && (
                    <>
                      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">System Information</h2>
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">Hostname</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-white">{diagnostics.system.hostname}</p>
                          </div>
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">Go Version</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-white">{diagnostics.system.go_version}</p>
                          </div>
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">OS/Arch</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-white">
                              {diagnostics.system.goos}/{diagnostics.system.goarch}
                            </p>
                          </div>
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">CPUs</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-white">{diagnostics.system.num_cpu}</p>
                          </div>
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">Goroutines</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-white">{diagnostics.system.num_goroutine}</p>
                          </div>
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">Start Time</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-white">
                              {new Date(diagnostics.system.start_time).toLocaleString()}
                            </p>
                          </div>
                        </div>
                      </div>

                      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Memory Usage</h2>
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">Allocated</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-white">
                              {formatBytes(diagnostics.memory.alloc_bytes)}
                            </p>
                          </div>
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">Heap Allocated</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-white">
                              {formatBytes(diagnostics.memory.heap_alloc_bytes)}
                            </p>
                          </div>
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">System Memory</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-white">
                              {formatBytes(diagnostics.memory.sys_bytes)}
                            </p>
                          </div>
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">GC Cycles</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-white">{diagnostics.memory.num_gc}</p>
                          </div>
                        </div>
                      </div>

                      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Database Statistics</h2>
                        <div className="grid grid-cols-3 gap-4">
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">Applications</p>
                            <p className="text-2xl font-bold text-gray-900 dark:text-white">{diagnostics.database.app_count}</p>
                          </div>
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">Sessions</p>
                            <p className="text-2xl font-bold text-gray-900 dark:text-white">{diagnostics.database.session_count}</p>
                          </div>
                          <div>
                            <p className="text-xs text-gray-500 dark:text-gray-400">Audit Logs</p>
                            <p className="text-2xl font-bold text-gray-900 dark:text-white">{diagnostics.database.audit_count}</p>
                          </div>
                        </div>
                      </div>
                    </>
                  )}
                </div>
              )}
            </>
          )}
        </main>
      </div>
    </div>
  );
}
