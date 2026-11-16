import { useState } from 'react';
import { KeyRound, Pencil, Plus, ShieldCheck, UserCheck, UserX } from 'lucide-react';
import { api, errorMessage, runAction, type app } from '../lib/api';
import { useAsync } from '../lib/hooks';
import { useToast } from './Toast';
import {
  Badge,
  Button,
  Card,
  CardHeader,
  ConfirmDialog,
  Field,
  InlineError,
  Input,
  Modal,
  Select,
  Spinner,
} from './ui';

type FormMode =
  | { kind: 'create' }
  | { kind: 'edit'; user: app.UserDTO }
  | { kind: 'reset'; user: app.UserDTO }
  | { kind: 'toggle'; user: app.UserDTO }
  | null;

export function AccountManagement({
  currentUserId,
  onAuthUpdated,
}: {
  currentUserId: number;
  onAuthUpdated: (state: app.AuthStateDTO) => void;
}) {
  const { notify } = useToast();
  const users = useAsync(() => api.Users(), []);

  const [mode, setMode] = useState<FormMode>(null);
  const [displayName, setDisplayName] = useState('');
  const [username, setUsername] = useState('');
  const [role, setRole] = useState('staff');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [formError, setFormError] = useState('');
  const [busy, setBusy] = useState(false);

  const openCreate = () => {
    setDisplayName('');
    setUsername('');
    setRole('staff');
    setPassword('');
    setConfirmPassword('');
    setFormError('');
    setMode({ kind: 'create' });
  };

  const openEdit = (user: app.UserDTO) => {
    setDisplayName(user.displayName);
    setRole(user.role);
    setFormError('');
    setMode({ kind: 'edit', user });
  };

  const openReset = (user: app.UserDTO) => {
    setPassword('');
    setConfirmPassword('');
    setFormError('');
    setMode({ kind: 'reset', user });
  };

  const close = () => {
    if (busy) {
      return;
    }
    setMode(null);
    setFormError('');
  };

  const refreshSession = async () => {
    const state = await api.AuthStatus();
    onAuthUpdated(state);
  };

  const submitCreate = async () => {
    setFormError('');
    if (username.trim().length < 3) {
      setFormError('اسم المستخدم يجب أن يكون 3 أحرف على الأقل.');
      return;
    }
    if ([...password].length < 8) {
      setFormError('يجب أن تتكون كلمة المرور من 8 أحرف على الأقل.');
      return;
    }
    if (password !== confirmPassword) {
      setFormError('كلمتا المرور غير متطابقتين.');
      return;
    }
    setBusy(true);
    try {
      const result = await runAction(
        api.CreateUser({ username: username.trim(), displayName: displayName.trim(), password, role }),
      );
      notify(result.message);
      setMode(null);
      users.reload();
    } catch (error) {
      setFormError(errorMessage(error));
    } finally {
      setBusy(false);
    }
  };

  const submitEdit = async () => {
    if (mode?.kind !== 'edit') {
      return;
    }
    setFormError('');
    setBusy(true);
    try {
      const result = await runAction(
        api.UpdateUser(mode.user.id, {
          displayName: displayName.trim(),
          role,
          active: mode.user.active,
        }),
      );
      notify(result.message);
      setMode(null);
      users.reload();
      await refreshSession();
    } catch (error) {
      setFormError(errorMessage(error));
    } finally {
      setBusy(false);
    }
  };

  const submitReset = async () => {
    if (mode?.kind !== 'reset') {
      return;
    }
    setFormError('');
    if ([...password].length < 8) {
      setFormError('يجب أن تتكون كلمة المرور من 8 أحرف على الأقل.');
      return;
    }
    if (password !== confirmPassword) {
      setFormError('كلمتا المرور غير متطابقتين.');
      return;
    }
    setBusy(true);
    try {
      const result = await runAction(api.ResetUserPassword(mode.user.id, password));
      notify(result.message);
      setMode(null);
      await refreshSession();
    } catch (error) {
      setFormError(errorMessage(error));
    } finally {
      setBusy(false);
    }
  };

  const submitToggle = async () => {
    if (mode?.kind !== 'toggle') {
      return;
    }
    setBusy(true);
    try {
      const result = await runAction(api.SetUserActive(mode.user.id, !mode.user.active));
      notify(result.message);
      setMode(null);
      users.reload();
      await refreshSession();
    } catch (error) {
      notify(errorMessage(error), 'error');
      setMode(null);
    } finally {
      setBusy(false);
    }
  };

  const items = users.data ?? [];

  return (
    <Card>
      <CardHeader
        title="إدارة المستخدمين"
        description="حسابات المشغّلين المحليين على محطة العمل هذه."
        action={
          <Button variant="primary" size="sm" icon={<Plus className="h-4 w-4" />} onClick={openCreate}>
            مستخدم جديد
          </Button>
        }
      />
      <div className="p-4">
        <InlineError message={users.error} />
        {users.loading && !users.data ? (
          <Spinner label="جارٍ تحميل المستخدمين…" />
        ) : (
          <ul className="divide-y divide-slate-700/50">
            {items.map((user) => {
              const self = user.id === currentUserId;
              return (
                <li key={user.id} className="flex flex-wrap items-center gap-x-3 gap-y-2 py-3 first:pt-0 last:pb-0">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="truncate text-[13px] font-medium text-slate-100">{user.displayName}</p>
                      <Badge tone={user.role === 'admin' ? 'emerald' : 'sky'}>
                        {user.role === 'admin' ? 'مسؤول' : 'موظف'}
                      </Badge>
                      {self ? <Badge tone="amber">أنت</Badge> : null}
                      {!user.active ? <Badge tone="rose">معطّل</Badge> : null}
                    </div>
                    <p className="mt-0.5 font-mono text-[11px] text-slate-500">{user.username}</p>
                  </div>
                  <div className="flex flex-wrap items-center gap-1.5">
                    <Button
                      variant="ghost"
                      size="sm"
                      icon={<Pencil className="h-3.5 w-3.5" />}
                      onClick={() => openEdit(user)}
                    >
                      تعديل
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      icon={<KeyRound className="h-3.5 w-3.5" />}
                      onClick={() => openReset(user)}
                    >
                      إعادة تعيين
                    </Button>
                    {self ? null : (
                      <Button
                        variant="ghost"
                        size="sm"
                        icon={
                          user.active ? (
                            <UserX className="h-3.5 w-3.5" />
                          ) : (
                            <UserCheck className="h-3.5 w-3.5" />
                          )
                        }
                        onClick={() => setMode({ kind: 'toggle', user })}
                      >
                        {user.active ? 'تعطيل' : 'تفعيل'}
                      </Button>
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
        )}
        <p className="mt-3 flex items-start gap-2 border-t border-slate-700/50 pt-3 text-[11px] leading-5 text-slate-500">
          <ShieldCheck className="mt-0.5 h-3.5 w-3.5 shrink-0 text-emerald-500" aria-hidden />
          <span>لا يمكن تعطيل آخر مسؤول نشط أو تعطيل حسابك الحالي.</span>
        </p>
      </div>

      <Modal
        open={mode?.kind === 'create'}
        onClose={close}
        title="مستخدم جديد"
        description="أنشئ حساباً محلياً بكلمة مرور مؤقتة."
        footer={
          <>
            <Button variant="ghost" onClick={close} disabled={busy}>
              إلغاء
            </Button>
            <Button variant="primary" loading={busy} onClick={submitCreate}>
              إضافة المستخدم
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          {formError ? <InlineError message={formError} /> : null}
          <Field label="اسم المستخدم" htmlFor="new-user-username" hint="3 أحرف على الأقل، بلا مسافات.">
            <Input
              id="new-user-username"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              autoComplete="off"
              spellCheck={false}
              disabled={busy}
            />
          </Field>
          <Field label="الاسم المعروض" htmlFor="new-user-display">
            <Input
              id="new-user-display"
              value={displayName}
              onChange={(event) => setDisplayName(event.target.value)}
              disabled={busy}
            />
          </Field>
          <Field label="الدور" htmlFor="new-user-role">
            <Select id="new-user-role" value={role} onChange={(event) => setRole(event.target.value)} disabled={busy}>
              <option value="staff">موظف</option>
              <option value="admin">مسؤول</option>
            </Select>
          </Field>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="كلمة المرور" htmlFor="new-user-password" hint="8 أحرف على الأقل.">
              <Input
                id="new-user-password"
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                disabled={busy}
              />
            </Field>
            <Field label="تأكيد كلمة المرور" htmlFor="new-user-confirm">
              <Input
                id="new-user-confirm"
                type="password"
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                disabled={busy}
              />
            </Field>
          </div>
        </div>
      </Modal>

      <Modal
        open={mode?.kind === 'edit'}
        onClose={close}
        title="تعديل الحساب"
        description={mode?.kind === 'edit' ? `الحساب: ${mode.user.username}` : undefined}
        footer={
          <>
            <Button variant="ghost" onClick={close} disabled={busy}>
              إلغاء
            </Button>
            <Button variant="primary" loading={busy} onClick={submitEdit}>
              حفظ التعديلات
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          {formError ? <InlineError message={formError} /> : null}
          <Field label="الاسم المعروض" htmlFor="edit-user-display">
            <Input
              id="edit-user-display"
              value={displayName}
              onChange={(event) => setDisplayName(event.target.value)}
              disabled={busy}
            />
          </Field>
          <Field label="الدور" htmlFor="edit-user-role">
            <Select id="edit-user-role" value={role} onChange={(event) => setRole(event.target.value)} disabled={busy}>
              <option value="staff">موظف</option>
              <option value="admin">مسؤول</option>
            </Select>
          </Field>
        </div>
      </Modal>

      <Modal
        open={mode?.kind === 'reset'}
        onClose={close}
        title="إعادة تعيين كلمة المرور"
        description={mode?.kind === 'reset' ? `الحساب: ${mode.user.username}` : undefined}
        footer={
          <>
            <Button variant="ghost" onClick={close} disabled={busy}>
              إلغاء
            </Button>
            <Button variant="primary" loading={busy} onClick={submitReset}>
              تعيين كلمة المرور
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          {formError ? <InlineError message={formError} /> : null}
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="كلمة المرور الجديدة" htmlFor="reset-user-password" hint="8 أحرف على الأقل.">
              <Input
                id="reset-user-password"
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                disabled={busy}
              />
            </Field>
            <Field label="تأكيد كلمة المرور" htmlFor="reset-user-confirm">
              <Input
                id="reset-user-confirm"
                type="password"
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                disabled={busy}
              />
            </Field>
          </div>
          <p className="text-[11px] leading-5 text-slate-500">
            لا تُعرض كلمة المرور الحالية ولا يمكن استعادتها، فقط استبدالها.
          </p>
        </div>
      </Modal>

      <ConfirmDialog
        open={mode?.kind === 'toggle'}
        title={mode?.kind === 'toggle' && mode.user.active ? 'تعطيل الحساب؟' : 'تفعيل الحساب؟'}
        message={
          mode?.kind === 'toggle' ? (
            <p>
              {mode.user.active
                ? 'سيفقد هذا المستخدم القدرة على تسجيل الدخول فوراً، وتبقى سجلاته في التدقيق.'
                : 'سيستعيد هذا المستخدم القدرة على تسجيل الدخول بكلمة مروره الحالية.'}
            </p>
          ) : null
        }
        confirmLabel={mode?.kind === 'toggle' && mode.user.active ? 'تعطيل' : 'تفعيل'}
        tone={mode?.kind === 'toggle' && mode.user.active ? 'danger' : 'primary'}
        busy={busy}
        onConfirm={submitToggle}
        onClose={close}
      />
    </Card>
  );
}
