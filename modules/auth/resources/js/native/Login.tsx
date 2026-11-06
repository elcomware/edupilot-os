import { useRef, useState } from "react";
import { View } from "react-native";
import type { SessionModel, SessionState } from "@edupilot/core/native/session";
import type { Translator } from "@edupilot/core/native/strings";
import { Action, Field, Heading, Notice, s } from "@edupilot/core/native/ui";

export function Login({
  model,
  state,
  t,
}: {
  model: SessionModel;
  state: SessionState;
  t: Translator;
}) {
  const credentials = useRef({ school: "", email: "", password: "" });
  const [visible, show] = useState(false),
    [invalid, setInvalid] = useState(false);
  const pending = state.kind === "signingIn";
  const submit = () => {
    const { school, email, password } = credentials.current;
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
          onChangeText={(value) => { credentials.current.school = value; }}
          autoCorrect={false}
          autoCapitalize="none"
          editable={!pending}
        />
        <Field
          testID="email"
          label={t("email")}
          onChangeText={(value) => { credentials.current.email = value; }}
          keyboardType="email-address"
          autoComplete="username"
          autoCorrect={false}
          autoCapitalize="none"
          editable={!pending}
        />
        <Field
          testID="password"
          label={t("password")}
          onChangeText={(value) => { credentials.current.password = value; }}
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
        {invalid && <Notice testID="login-error-incomplete" error>{t("incomplete")}</Notice>}
        {state.kind === "guest" && state.reason && (
          <Notice testID={`login-error-${state.reason}`} error>{t(state.reason)}</Notice>
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
