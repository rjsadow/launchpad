import type { Application, User } from '../types';

const API_BASE = '/api';

class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const message = await response.text().catch(() => response.statusText);
    throw new ApiError(response.status, message);
  }
  return response.json();
}

export async function fetchCurrentUser(): Promise<User | null> {
  try {
    const response = await fetch(`${API_BASE}/me`);
    if (response.status === 401) {
      return null;
    }
    return handleResponse<User>(response);
  } catch {
    return null;
  }
}

export async function fetchApps(): Promise<Application[]> {
  const response = await fetch(`${API_BASE}/apps`);
  return handleResponse<Application[]>(response);
}

export async function fetchFavorites(): Promise<Application[]> {
  const response = await fetch(`${API_BASE}/favorites`);
  return handleResponse<Application[]>(response);
}

export async function addFavorite(appId: string): Promise<void> {
  const response = await fetch(`${API_BASE}/favorites/${appId}`, {
    method: 'POST',
  });
  await handleResponse(response);
}

export async function removeFavorite(appId: string): Promise<void> {
  const response = await fetch(`${API_BASE}/favorites/${appId}`, {
    method: 'DELETE',
  });
  await handleResponse(response);
}

export async function logout(): Promise<void> {
  await fetch('/auth/logout', {
    method: 'POST',
    headers: { Accept: 'application/json' },
  });
}

// Admin API
export async function fetchAllApps(): Promise<Application[]> {
  const response = await fetch(`${API_BASE}/admin/apps`);
  return handleResponse<Application[]>(response);
}

export async function createApp(app: Partial<Application>): Promise<Application> {
  const response = await fetch(`${API_BASE}/apps`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(app),
  });
  return handleResponse<Application>(response);
}

export async function updateApp(app: Application): Promise<Application> {
  const response = await fetch(`${API_BASE}/apps/${app.id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(app),
  });
  return handleResponse<Application>(response);
}

export async function deleteApp(appId: string): Promise<void> {
  const response = await fetch(`${API_BASE}/apps/${appId}`, {
    method: 'DELETE',
  });
  await handleResponse(response);
}
