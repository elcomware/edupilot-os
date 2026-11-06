import { useState } from "react";
import { View } from "react-native";
import type { SessionModel, SessionState } from "../../../../../clients/mobile/src/session";
import type { Translator } from "../../../../../clients/mobile/src/strings";
import { Action, Field, Heading, Notice, s } from "../../../../../clients/mobile/src/ui";

export function Login({
  model,
  state,
  t,
}: {
  model: SessionModel;
  state: SessionState;
  t: Translator;
}) {
  const [school, setSchool] = useState(""),
    [email, setEmail] = useState(""),
    [password, setPassword] = useState(""),
    [visible, show] = useState(false),
    [invalid, setInvalid] = useState(false);
  const pending = state.kind === "signingIn";
  const submit = () => {
    if (!school.trim() || !email.trim() || !password) {
      setInvalid(true);
      return;
    }
    setInvalid(false);
    show(false);
    void model.login({ school: school.trim(), email: email.trim(), password });
  };
  return (
    <>
      <Heading
        kicker={t("register")}
        title={t("signIn")}
        detail={t("signInIntro")}
      />
      <View style={s.panel}>
        <Field
          testID="school-id"
          label={t("school")}
          value={school}
          onChangeText={setSchool}
          autoCorrect={false}
          autoCapitalize="none"
          editable={!pending}
        />
        <Field
          testID="email"
          label={t("email")}
          value={email}
          onChangeText={setEmail}
          keyboardType="email-address"
          autoComplete="username"
          autoCorrect={false}
          autoCapitalize="none"
          editable={!pending}
        />
        <Field
          testID="password"
          label={t("password")}
          value={password}
          onChangeText={setPassword}
          secureTextEntry={!visible}
          autoComplete="current-password"
          autoCapitalize="none"
          editable={!pending}
          returnKeyType="go"
          onSubmitEditing={submit}
        />
        <View style={s.row}>
          <Action
            label={t(visible ? "hide" : "show")}
            onPress={() => show(!visible)}
            disabled={pending}
          />
        </View>
        {invalid && <Notice error>{t("incomplete")}</Notice>}
        {state.kind === "guest" && state.reason && (
          <Notice error>{t(state.reason)}</Notice>
        )}
        <Action
          testID="sign-in"
          label={t("signIn")}
          variant="solid"
          onPress={submit}
          busy={pending}
        />
      </View>
    </>
  );
}
