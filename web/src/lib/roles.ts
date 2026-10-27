import type { Role } from '../api/types';

const rank: Record<Role, number> = { agent: 1, supervisor: 2, admin: 3 };

export function hasRole(role: Role | undefined, min: Role): boolean {
  if (!role) return false;
  return (rank[role] ?? 0) >= rank[min];
}

export const isSupPlus = (r?: Role) => hasRole(r, 'supervisor');
export const isAdmin = (r?: Role) => hasRole(r, 'admin');
