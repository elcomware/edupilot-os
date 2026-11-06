import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
} from "react";
import {
  BackHandler,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StatusBar,
  View,
} from "react-native";
import { SafeAreaProvider, SafeAreaView } from "react-native-safe-area-context";
import { getLocales } from "expo-localization";
import {
  createApiClient,
  type ApiClient,
  type Principal,
  type ModuleDescriptor,
} from "@edupilot/api-client";
import { apiRoot } from "./src/endpoint";
import { SessionModel } from "@edupilot/core/native/session";
import { tokenStore } from "./src/storage";
import { copy, type Locale, type Translator } from "@edupilot/core/native/strings";
import { Action, Brand, Notice, Spinner, s } from "@edupilot/core/native/ui";
import { directoryIcons, LoginPage, workspacePages } from "./src/pages";
import { Catalog } from "@edupilot/core/native/screens/Catalog";
import { blank, type Draft } from "@edupilot/core/native/screens/Register";

const root = apiRoot(process.env.EXPO_PUBLIC_API_ORIGIN, __DEV__);
const timeoutFetch: typeof fetch = (input, init) => {
  const controller = new AbortController();
  const abort = () => controller.abort();
  init?.signal?.addEventListener("abort", abort);
  if (init?.signal?.aborted) controller.abort();
  const timer = setTimeout(abort, 18000);
  return fetch(input, { ...init, signal: controller.signal }).finally(() => {
    clearTimeout(timer);
    init?.signal?.removeEventListener("abort", abort);
  });
};
function Member({
  api,
  principal,
  t,
  locale,
  logout,
}: {
  api: ApiClient;
  principal: Principal;
  t: Translator;
  locale: Locale;
  logout: () => void;
}) {
  const [module, setModule] = useState<ModuleDescriptor | null>(null);
  // Retain unsaved drafts across in-app navigation only; signing out
  // unmounts this entire component and discards every school-specific value.
  const [drafts, saveDrafts] = useState<Record<string, Draft>>({});
  useEffect(() => {
    const sub = BackHandler.addEventListener("hardwareBackPress", () => {
      if (!module) return false;
      setModule(null);
      return true;
    });
    return () => sub.remove();
  }, [module]);
  const Register=module && workspacePages.get(module.key);
  return (
    <>
      <View style={s.row}>
        {module && <Action label={t("back")} onPress={() => setModule(null)} />}
        <Action label={t("logout")} onPress={logout} />
      </View>
      {module && !Register ? <Notice error>{t("forbidden")}</Notice> : module && Register ? (
        <Register
          key={module.key}
          api={api}
          module={module}
          principal={principal}
          t={t}
          locale={locale}
          draft={drafts[module.key] ?? blank}
          setDraft={(d) => saveDrafts((prev) => ({ ...prev, [module.key]: d }))}
        />
      ) : (
        <Catalog
          icons={directoryIcons}
          api={api}
          principal={principal}
          open={setModule}
          t={t}
          locale={locale}
        />
      )}
    </>
  );
}
function SessionContent({
  base,
  t,
  locale,
}: {
  base: string;
  t: Translator;
  locale: Locale;
}) {
  const { model, api } = useMemo(() => {
    const model = new SessionModel(
      tokenStore(base),
      (getToken, onUnauthorized) =>
        createApiClient({
          baseUrl: base,
          getToken,
          onUnauthorized,
          fetcher: timeoutFetch,
        }),
    );
    const api = createApiClient({
      baseUrl: base,
      getToken: model.getToken,
      onUnauthorized: model.onExpired,
      fetcher: timeoutFetch,
    });
    return { model, api };
  }, [base]);
  const state = useSyncExternalStore(model.subscribe, model.snapshot);
  useEffect(() => {
    void model.restore();
  }, [model]);
  switch (state.kind) {
    case "guest":
    case "signingIn":
      return <LoginPage model={model} state={state} t={t} />;
    case "checking":
    case "signingOut":
      return <Spinner t={t} />;
    case "member":
      return (
        <Member
          key={state.principal.school + "/" + state.principal.id}
          api={api}
          principal={state.principal}
          locale={locale}
          t={t}
          logout={() => void model.logout()}
        />
      );
    case "retry":
    case "erase":
      return (
        <>
          <Notice error>
            {t(state.kind === "retry" ? "restore" : "erase")}
          </Notice>
          {state.kind === "retry" && (
            <Action label={t("retry")} onPress={() => void model.restore()} />
          )}
          <Action label={t("clearLocal")} onPress={() => void model.logout()} />
        </>
      );
  }
}
export default function App() {
  const [locale, setLocale] = useState<Locale>(() =>
    getLocales()[0]?.languageCode === "fr" ? "fr" : "en",
  );
  const t: Translator = useCallback((key) => copy[locale][key], [locale]);
  return (
    <SafeAreaProvider>
      <StatusBar barStyle="dark-content" />
      <SafeAreaView style={s.full}>
        <View style={s.header}>
          <View style={s.headerInner}>
            <Brand label={t("brand")} />
            <Action
              label={t("language")}
              variant="dark"
              onPress={() => setLocale(locale === "en" ? "fr" : "en")}
            />
          </View>
        </View>
        <KeyboardAvoidingView
          style={s.full}
          behavior={Platform.OS === "ios" ? "padding" : undefined}
        >
          <ScrollView
            contentContainerStyle={s.scroll}
            keyboardShouldPersistTaps="handled"
          >
            <View style={s.body}>
              {root ? (
                <SessionContent base={root} locale={locale} t={t} />
              ) : (
                <Notice error>{t("setup")}</Notice>
              )}
            </View>
          </ScrollView>
        </KeyboardAvoidingView>
      </SafeAreaView>
    </SafeAreaProvider>
  );
}
