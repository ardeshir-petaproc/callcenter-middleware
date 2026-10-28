import type { Role, User } from '../api/types';
import { hasRole } from './roles';

/** Who may see a page or nav item: any of `perm`, or at least role `min`, or platform admins only. */
export interface Access { min?: Role; perm?: string | string[]; platform?: boolean }

export function hasAccess(a: Access, user: User | null, can: (p: string) => boolean): boolean {
  if (!user) return false;
  if (a.platform) return !!user.platform_admin;
  const perms = a.perm === undefined ? [] : Array.isArray(a.perm) ? a.perm : [a.perm];
  if (perms.some(can)) return true;
  return a.min ? hasRole(user.role, a.min) : perms.length === 0;
}
