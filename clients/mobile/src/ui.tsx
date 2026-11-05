import {
  ActivityIndicator,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
  type TextInputProps,
} from 'react-native';
import type { ReactNode } from 'react';
import type { Translator } from './strings';

export const color = {
  paper: '#f5f7f3',
  surface: '#ffffff',
  ink: '#172e29',
  muted: '#50645e',
  nav: '#183c36',
  accent: '#226049',
  band: '#e2efe5',
  rule: '#c7d3c9',
  error: '#842e27',
};
const display = Platform.select({
  ios: 'Georgia',
  android: 'serif',
  default: 'serif',
});
export const s = StyleSheet.create({
  full: { flex: 1, backgroundColor: color.paper },
  scroll: { flexGrow: 1 },
  body: {
    width: '100%',
    maxWidth: 720,
    alignSelf: 'center',
    padding: 20,
    gap: 16,
  },
  header: {
    backgroundColor: color.nav,
    paddingHorizontal: 20,
    paddingVertical: 14,
  },
  headerInner: {
    alignSelf: 'center',
    width: '100%',
    maxWidth: 680,
    flexDirection: 'row',
    flexWrap: 'wrap',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 8,
  },
  logo: { fontFamily: display, color: 'white', fontSize: 29 },
  onDark: { color: '#e2efe5', fontSize: 15 },
  eyebrow: {
    color: color.accent,
    fontSize: 13,
    fontWeight: '700',
    letterSpacing: 1.4,
  },
  title: { fontFamily: display, fontSize: 33, color: color.ink },
  text: { fontSize: 16, lineHeight: 23, color: color.ink },
  muted: { fontSize: 15, lineHeight: 22, color: color.muted },
  row: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 10,
    alignItems: 'center',
  },
  stack: { gap: 10 },
  panel: {
    backgroundColor: color.surface,
    borderWidth: 1,
    borderColor: color.rule,
    padding: 16,
    borderRadius: 8,
    gap: 14,
  },
  rowItem: {
    borderBottomWidth: 1,
    borderBottomColor: color.rule,
    paddingVertical: 12,
    gap: 4,
  },
  name: { fontSize: 17, fontWeight: '600', color: color.ink },
  label: { color: color.ink, fontWeight: '600', fontSize: 15 },
  field: {
    borderWidth: 1,
    borderColor: '#82948a',
    backgroundColor: color.surface,
    borderRadius: 5,
    minHeight: 46,
    paddingHorizontal: 12,
    paddingVertical: 10,
    color: color.ink,
    fontSize: 16,
  },
  btn: {
    minHeight: 44,
    borderRadius: 5,
    paddingHorizontal: 14,
    paddingVertical: 9,
    justifyContent: 'center',
    alignItems: 'center',
  },
  solid: { backgroundColor: color.accent },
  secondary: { backgroundColor: color.band },
  darkBtn: { backgroundColor: '#295448' },
  btnLabel: { fontSize: 15, fontWeight: '600', textAlign: 'center' },
  error: { color: color.error, fontSize: 15, lineHeight: 21 },
  notice: { color: color.accent, fontSize: 15, lineHeight: 21 },
});
export function Action({
  label,
  onPress,
  variant = 'secondary',
  disabled = false,
  busy = false,
}: {
  label: string;
  onPress: () => void;
  variant?: 'solid' | 'secondary' | 'dark';
  disabled?: boolean;
  busy?: boolean;
}) {
  const fg = variant === 'secondary' ? color.ink : '#ffffff';
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: disabled || busy, busy }}
      disabled={disabled || busy}
      onPress={onPress}
      style={({ pressed }) => [
        s.btn,
        variant === 'solid'
          ? s.solid
          : variant === 'dark'
            ? s.darkBtn
            : s.secondary,
        { opacity: disabled || pressed ? 0.65 : 1 },
      ]}
    >
      {busy ? (
        <ActivityIndicator color={fg} />
      ) : (
        <Text style={[s.btnLabel, { color: fg }]}>{label}</Text>
      )}
    </Pressable>
  );
}
export function Field({ label, ...props }: TextInputProps & { label: string }) {
  return (
    <View style={s.stack}>
      <Text style={s.label}>{label}</Text>
      <TextInput
        style={s.field}
        accessibilityLabel={label}
        placeholderTextColor={color.muted}
        {...props}
      />
    </View>
  );
}
export function Notice({
  children,
  error = false,
}: {
  children: ReactNode;
  error?: boolean;
}) {
  return (
    <Text
      accessibilityRole={error ? 'alert' : undefined}
      accessibilityLiveRegion="polite"
      style={error ? s.error : s.notice}
    >
      {children}
    </Text>
  );
}
export function Heading({
  kicker,
  title,
  detail,
}: {
  kicker: string;
  title: string;
  detail?: string;
}) {
  return (
    <View style={s.stack}>
      <Text style={s.eyebrow}>{kicker}</Text>
      <Text accessibilityRole="header" style={s.title}>
        {title}
      </Text>
      {detail && <Text style={s.muted}>{detail}</Text>}
    </View>
  );
}
export function Spinner({ t }: { t: Translator }) {
  return (
    <View style={s.row}>
      <ActivityIndicator color={color.accent} />
      <Text style={s.muted}>{t('progress')}</Text>
    </View>
  );
}
export function Pager({
  offset,
  total,
  size,
  setOffset,
  t,
  locale,
}: {
  offset: number;
  total: number;
  size: number;
  setOffset: (n: number) => void;
  t: Translator;
  locale: string;
}) {
  const fmt = new Intl.NumberFormat(locale);
  return (
    <View style={s.row}>
      <Action
        label={t('previous')}
        onPress={() => setOffset(Math.max(0, offset - size))}
        disabled={offset === 0}
      />
      <Text style={s.muted}>
        {t('page')} {fmt.format(1 + Math.floor(offset / size))} {t('of')}{' '}
        {fmt.format(Math.max(1, Math.ceil(total / size)))}
      </Text>
      <Action
        label={t('next')}
        onPress={() => setOffset(offset + size)}
        disabled={offset + size >= total}
      />
    </View>
  );
}
