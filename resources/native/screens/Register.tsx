import { useCallback, useRef, useState } from "react";
import { Text, View } from "react-native";
import type {
  ApiClient,
  ModuleDescriptor,
  Page,
  Principal,
  WorkspaceEntry,
} from "@edupilot/api-client";
import type { Locale, Translator, TextKey } from "../strings";
import { feedback } from "../session";
import { useResource } from "../resource";
import { Action, Field, Heading, Notice, Pager, Spinner, s } from "../ui";

export interface Draft {
  title: string;
}
export const blank: Draft = { title: "" };
const pageSize = 20;
type Row = WorkspaceEntry;
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
  const canRead = principal.permissions.includes("workspace.read");
  const canWrite =
    principal.permissions.includes("workspace.write") &&
    module.capabilities.includes("create");
  const [offset, setOffset] = useState(0);
  const load = useCallback(
    async (signal: AbortSignal): Promise<Page<Row>> => {
      if (!canRead) return { items: [], total: 0 };
      const p = { limit: pageSize, offset };
      return api.workspace(module.key, p, signal);
    },
    [api, module.key, offset, canRead],
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
    const title = draft.title.trim();
    if (!title || Array.from(title).length > 160) {
      setMessage({ key: "invalidTitle", error: true });
      return;
    }
    gate.current = true;
    setSaving(true);
    setMessage(undefined);
    try {
      await api.createEntry(module.key, { title });
      setDraft(blank);
      setMessage({ key: "added", error: false });
      setOffset(0);
      refresh();
    } catch (e) {
      const key = feedback(e);
      setMessage({ key, error: true, uncertain: key === "network" });
    } finally {
      gate.current = false;
      setSaving(false);
    }
  };
  return (
    <>
      <Heading kicker={t("workspace")} title={module.title[locale]} />
      {canWrite && (
        <View style={s.panel}>
          <Field
            testID="entry-title"
            label={t("entryTitle")}
            value={draft.title}
            onChangeText={(title) => setDraft({ ...draft, title })}
            editable={!saving}
          />
          <Action
            label={t("addEntry")}
            onPress={() => void submit()}
            variant="solid"
            busy={saving}
          />
          {message && (
            <Notice error={message.error}>
              {t(message.key)}
              {message.uncertain ? " " + t("pendingCreate") : ""}
            </Notice>
          )}
        </View>
      )}
      {!canRead ? (
        <Notice error>{t("forbidden")}</Notice>
      ) : (
        <>
          <View style={s.row}>
            <Action
              label={t("refresh")}
              onPress={refresh}
              disabled={state.kind === "loading"}
            />
          </View>
          {state.kind === "loading" && <Spinner t={t} />}
          {state.kind === "failed" && <Notice error>{t(state.reason)}</Notice>}
          {state.kind === "ready" && (
            <View style={s.panel}>
              {!state.data.items.length && (
                <Text style={s.muted}>{t("emptyEntries")}</Text>
              )}
              {state.data.items.map((row) => (
                <View style={s.rowItem} key={row.id}>
                  <Text style={s.name}>{row.title}</Text>
                  <Text style={s.muted}>
                    {row.status === "open" || row.status === "closed"
                      ? t(row.status)
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
