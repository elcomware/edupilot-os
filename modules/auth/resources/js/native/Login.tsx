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
    [missing, setMissing] = useState("");
  const [proof, setProof] = useState("idle");
  const pending = state.kind === "signingIn";
  function record(field: keyof typeof credentials.current, value: string) {
    credentials.current[field] = value;
    setMissing("");
  }
  const submit = () => {
    const { school, email, password } = credentials.current;
    if (!school.trim() || !email.trim() || !password) {
      setProof("rejected-empty");
      setMissing(!school.trim() ? "school" : !email.trim() ? "email" : "password");
      return;
    }
    setMissing("");
    show(false);
    setProof("submitted-" + model.snapshot().kind);
    void model.login({ school: school.trim(), email: email.trim(), password }).then(() => {
      const result = model.snapshot();
      setProof("resolved-" + result.kind + (result.kind === "guest" && result.reason ? "-" + result.reason : ""));
    });
  };
  return (
    <>
      <Heading
        kicker={t("register")}
        title={t("signIn")}
        detail={t("signInIntro")}
      />
      <View testID={`login-proof-${proof}`} style={s.panel}>
        <Field
          testID="school-id"
          label={t("school")}
          onChangeText={(value) => record("school", value)}
          onEndEditing={(event) => record("school", event.nativeEvent.text)}
          autoCorrect={false}
          autoCapitalize="none"
          editable={!pending}
        />
        <Field
          testID="email"
          label={t("email")}
          onChangeText={(value) => record("email", value)}
          onEndEditing={(event) => record("email", event.nativeEvent.text)}
          keyboardType="email-address"
          autoComplete="username"
          autoCorrect={false}
          autoCapitalize="none"
          editable={!pending}
        />
        <Field
          testID="password"
          label={t("password")}
          onChangeText={(value) => record("password", value)}
          onEndEditing={(event) => record("password", event.nativeEvent.text)}
          secureTextEntry={!visible}
          autoComplete="current-password"
          autoCapitalize="none"
          editable={!pending}
          returnKeyType="go"
          onSubmitEditing={(event) => {
            record("password", event.nativeEvent.text);
            submit();
          }}
        />
        <View style={s.row}>
          <Action
            label={t(visible ? "hide" : "show")}
            onPress={() => show(!visible)}
            disabled={pending}
          />
        </View>
        {missing !== "" && <Notice testID={`login-error-incomplete-${missing}`} error>{t("incomplete")}</Notice>}
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
