export interface Application {
  id: string;
  name: string;
  description: string;
  url: string;
  icon: string;
  category: string;
  roles?: string[];
  enabled?: boolean;
  sortOrder?: number;
  isFavorite?: boolean;
}

export interface AppConfig {
  applications: Application[];
}

export interface User {
  id: string;
  email: string;
  name: string;
  roles: string[];
  isAdmin: boolean;
}
