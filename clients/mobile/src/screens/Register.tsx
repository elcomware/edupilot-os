import { useCallback, useRef, useState } from 'react';
import { Text, View } from 'react-native';
import type {
  ApiClient,
  ModuleDescriptor,
  Page,
  Principal,
  Student,
  WorkspaceEntry,
} from '@edupilot/api-client';
import type { Locale, Translator, TextKey } from '../strings';
import { feedback } from '../session';
import { useResource } from '../resource';
import { Action, Field, Heading, Notice, Pager, Spinner, s } from '../ui';

export interface Draft {
  given: string;
  family: string;
  title: string;
}
export const blank: Draft = { given: '', family: '', title: '' };
const pageSize = 20;
type Row = Student | WorkspaceEntry;
export function Register({
  api,
  module,
  principal,
  locale,
  t,
  draft,
  setDraft,
}: {
  api: ApiClient;
  module: ModuleDescriptor;
  principal: Principal;
  locale: Locale;
  t: Translator;
  draft: Draft;
  setDraft: (d: Draft) => void;
}) {
  const student = module.key === 'students';
  const canRead = principal.permissions.includes(
    student ? 'students.read' : 'workspace.read',
  );
  const canWrite =
    principal.permissions.includes(
      student ? 'students.write' : 'workspace.write',
    ) && module.capabilities.includes('create');
  const [offset, setOffset] = useState(0);
  const load = useCallback(
    async (signal: AbortSignal): Promise<Page<Row>> => {
      if (!canRead) return { items: [], total: 0 };
      const p = { limit: pageSize, offset };
      return student
        ? api.students(p, signal)
        : api.workspace(module.key, p, signal);
    },
    [api, student, module.key, offset, canRead],
  );
  const { state, refresh } = useResource(load);
  const gate = useRef(false);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{
    key: TextKey;
    error: boolean;
    uncertain?: boolean;
  }>();
  const submit = async () => {
    if (gate.current || !canWrite) return;
    const given = draft.given.trim(),
      family = draft.family.trim(),
      title = draft.title.trim();
    if (
      student &&
      ![given, family].every(
        (v) =>
          v &&
          Array.from(v).length <= 100 &&
          !/[\u0000-\u001f\u007f-\u009f]/u.test(v),
      )
    ) {
      setMessage({ key: 'invalidNames', error: true });
      return;
    }
    if (!student && (!title || Array.from(title).length > 160)) {
      setMessage({ key: 'invalidTitle', error: true });
      return;
    }
    gate.current = true;
    setSaving(true);
    setMessage(undefined);
    try {
      if (student)
        await api.createStudent({ givenName: given, familyName: family });
      else await api.createEntry(module.key, { title });
      setDraft(blank);
      setMessage({ key: student ? 'registered' : 'added', error: false });
      setOffset(0);
      refresh();
    } catch (e) {
      const key = feedback(e);
      setMessage({ key, error: true, uncertain: key === 'network' });
    } finally {
      gate.current = false;
      setSaving(false);
    }
  };
  return (
    <>
      <Heading
        kicker={t(student ? 'roster' : 'workspace')}
        title={module.title[locale]}
      />
      {canWrite && (
        <View style={s.panel}>
          {student ? (
            <>
              <Field
                label={t('given')}
                value={draft.given}
                onChangeText={(given) => setDraft({ ...draft, given })}
                autoCapitalize="words"
                editable={!saving}
              />
              <Field
                label={t('family')}
                value={draft.family}
                onChangeText={(family) => setDraft({ ...draft, family })}
                autoCapitalize="words"
                editable={!saving}
              />
            </>
          ) : (
            <Field
              label={t('entryTitle')}
              value={draft.title}
              onChangeText={(title) => setDraft({ ...draft, title })}
              editable={!saving}
            />
          )}
          <Action
            label={t(student ? 'addStudent' : 'addEntry')}
            onPress={() => void submit()}
            variant="solid"
            busy={saving}
          />
          {message && (
            <Notice error={message.error}>
              {t(message.key)}
              {message.uncertain ? ' ' + t('pendingCreate') : ''}
            </Notice>
          )}
        </View>
      )}
      {!canRead ? (
        <Notice error>{t('forbidden')}</Notice>
      ) : (
        <>
          <View style={s.row}>
            <Action
              label={t('refresh')}
              onPress={refresh}
              disabled={state.kind === 'loading'}
            />
          </View>
          {state.kind === 'loading' && <Spinner t={t} />}
          {state.kind === 'failed' && <Notice error>{t(state.reason)}</Notice>}
          {state.kind === 'ready' && (
            <View style={s.panel}>
              {!state.data.items.length && (
                <Text style={s.muted}>
                  {t(student ? 'emptyStudents' : 'emptyEntries')}
                </Text>
              )}
              {state.data.items.map((row) => (
                <View style={s.rowItem} key={row.id}>
                  <Text style={s.name}>
                    {'givenName' in row
                      ? row.givenName + ' ' + row.familyName
                      : row.title}
                  </Text>
                  <Text style={s.muted}>
                    {row.status === 'active'
                      ? t('active')
                      : row.status === 'archived'
                        ? t('archived')
                        : row.status === 'open'
                          ? t('open')
                          : row.status === 'closed'
                            ? t('closed')
                            : row.status}
                  </Text>
                </View>
              ))}
              {(offset > 0 || state.data.total > pageSize) && (
                <Pager
                  size={pageSize}
                  offset={offset}
                  setOffset={setOffset}
                  total={state.data.total}
                  t={t}
                  locale={locale}
                />
              )}
            </View>
          )}
        </>
      )}
    </>
  );
}
