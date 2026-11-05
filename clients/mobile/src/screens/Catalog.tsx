import { useCallback } from "react";
import { Text, View } from "react-native";
import type {
  ApiClient,
  ModuleDescriptor,
  Principal,
} from "@edupilot/api-client";
import type { Locale, Translator } from "../strings";
import { useResource } from "../resource";
import { Action, Heading, Notice, Spinner, s } from "../ui";

export function Catalog({
  api,
  principal,
  open,
  t,
  locale,
}: {
  api: ApiClient;
  principal: Principal;
  open: (m: ModuleDescriptor) => void;
  t: Translator;
  locale: Locale;
}) {
  const load = useCallback((signal: AbortSignal) => api.modules(signal), [api]);
  const { state, refresh } = useResource(load);
  return (
    <>
      <Heading
        kicker={principal.school}
        title={t("modules")}
        detail={t("modulesIntro")}
      />
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
            <Text style={s.muted}>{t("noModules")}</Text>
          )}
          {state.data.items.map((m) => (
            <Action
              key={m.key}
              label={m.title[locale]}
              onPress={() => open(m)}
            />
          ))}
        </View>
      )}
    </>
  );
}
