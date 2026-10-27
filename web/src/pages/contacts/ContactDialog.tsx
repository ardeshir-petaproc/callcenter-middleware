import { useEffect, useState } from 'react';
import type { Contact } from '../../api/types';
import { post, put } from '../../api/client';
import { Dialog } from '../../components/ui/Dialog';
import { Field, Toggle } from '../../components/ui/primitives';
import { TagInput } from '../../components/ui/TagInput';
import { toast } from '../../components/ui/Toast';

const blank: Omit<Contact, 'id'> = { name: '', phone: '', email: '', company: '', notes: '', tags: [], vip: false };

interface Props { open: boolean; initial: Contact | null; presetPhone?: string; onClose: () => void; onSaved: (c: Contact) => void }

export function ContactDialog({ open, initial, presetPhone, onClose, onSaved }: Props) {
  const [c, setC] = useState<Omit<Contact, 'id'>>(blank);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    if (open) setC(initial ? { ...initial, tags: initial.tags ?? [] } : { ...blank, phone: presetPhone ?? '' });
  }, [open, initial, presetPhone]);
  const set = (p: Partial<Contact>) => setC((x) => ({ ...x, ...p }));
  const save = async () => {
    setBusy(true);
    try {
      const saved = initial ? await put<Contact>(`/contacts/${initial.id}`, { ...c, id: initial.id }) : await post<Contact>('/contacts', c);
      toast.success(initial ? 'Contact updated' : 'Contact created');
      onSaved(saved);
      onClose();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  return (
    <Dialog open={open} onClose={onClose} title={initial ? 'Edit contact' : 'New contact'}
      footer={<><button className="btn" onClick={onClose}>Cancel</button><button className="btn primary" disabled={busy || !c.phone.trim()} onClick={save}>Save</button></>}>
      <div className="form-grid">
        <Field label="Name"><input className="input" value={c.name} onChange={(e) => set({ name: e.target.value })} data-autofocus /></Field>
        <Field label="Phone"><input className="input" inputMode="tel" value={c.phone} onChange={(e) => set({ phone: e.target.value })} /></Field>
        <Field label="Email"><input className="input" type="email" value={c.email} onChange={(e) => set({ email: e.target.value })} /></Field>
        <Field label="Company"><input className="input" value={c.company} onChange={(e) => set({ company: e.target.value })} /></Field>
        <div className="field span-all"><span className="field-label">Tags</span><TagInput value={c.tags} onChange={(v) => set({ tags: v })} /></div>
        <Field label="Notes" className="span-all"><textarea className="textarea" value={c.notes} onChange={(e) => set({ notes: e.target.value })} /></Field>
        <div className="span-all"><Toggle checked={c.vip} onChange={(v) => set({ vip: v })} label="VIP — prioritise and highlight this caller" /></div>
      </div>
    </Dialog>
  );
}
