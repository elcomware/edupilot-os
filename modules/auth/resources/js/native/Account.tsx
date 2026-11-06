import { useCallback } from "react";
import { Text, View } from "react-native";
import type { ApiClient, Locale } from "@edupilot/api-client";
import { Action, Heading, Notice, Spinner, s } from "@native/ui";
import type { Translator } from "@native/strings";
import { useResource } from "@native/resource";
import { accountLabel } from "../account";
export function Account({
  api,
  locale,
  t,
}: {
  api: ApiClient;
  locale: Locale;
  t: Translator;
}) {
  const load = useCallback(
    (signal: AbortSignal) => api.currentUser(signal),
    [api],
  );
  const { state, refresh } = useResource(load);
  if (state.kind === "loading") return <Spinner t={t} />;
  if (state.kind === "failed")
    return (
      <>
        <Notice error>{t(state.reason)}</Notice>
        <Action label={t("retry")} onPress={refresh} />
      </>
    );
  const user = state.data;
  return (
    <>
      <Heading
        kicker={user.membership.school}
        title={accountLabel(locale, "title")}
      />
      <View style={s.panel}>
        <Text style={s.name}>{user.email}</Text>
        <Text>
          {accountLabel(locale, "role")} ·{" "}
          {accountLabel(locale, user.membership.role)}
        </Text>
        <Text style={s.name}>{accountLabel(locale, "permissions")}</Text>
        {user.permissions.map((p) => (
          <Text key={p}>{accountLabel(locale, p)}</Text>
        ))}
      </View>
    </>
  );
}
