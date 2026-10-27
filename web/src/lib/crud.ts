import { useCallback } from 'react';
import { del, get, post, put } from '../api/client';
import { useApi } from './hooks';
import { refreshLookup } from '../state/lookups';

type LookupKey = Parameters<typeof refreshLookup>[0];

/** List + create/update/delete for a REST collection; keeps the shared lookup cache in sync. */
export function useCrud<T extends { id: number }>(path: string, lookup?: LookupKey) {
  const state = useApi((s) => get<T[]>(path, undefined, s), [path]);
  const { setData } = state;

  const save = useCallback(async (item: Partial<T> & { id?: number }, extra?: Record<string, unknown>): Promise<T> => {
    const body = { ...item, ...extra };
    const saved = item.id ? await put<T>(`${path}/${item.id}`, body) : await post<T>(path, body);
    const result = (saved && typeof saved === 'object' && 'id' in saved ? saved : { ...item, id: item.id }) as T;
    setData((list) => {
      const arr = list ?? [];
      return arr.some((x) => x.id === result.id) ? arr.map((x) => (x.id === result.id ? result : x)) : [...arr, result];
    });
    if (lookup) void refreshLookup(lookup);
    return result;
  }, [path, lookup, setData]);

  const remove = useCallback(async (id: number) => {
    await del(`${path}/${id}`);
    setData((list) => list?.filter((x) => x.id !== id));
    if (lookup) void refreshLookup(lookup);
  }, [path, lookup, setData]);

  return { ...state, items: state.data ?? [], save, remove };
}
