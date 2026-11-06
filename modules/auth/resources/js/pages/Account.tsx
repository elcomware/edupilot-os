import { useQuery } from "@tanstack/react-query";
import type { Principal } from "@edupilot/api-client";
import { api } from "@resources/app/session";
import { useLocale } from "@resources/app/preferences";
import { t } from "@resources/app/copy";
import { PageHeading, StatePanel } from "@resources/components/primitives";
import { Button, Card, Skeleton } from "@resources/components/Surface";
import { accountLabel } from "../account";

export function Account({
  principal,
}: {
  principal: Principal;
  moduleKey: string;
}) {
  const locale = useLocale();
  const profile = useQuery({
    queryKey: ["account", principal.school, principal.id],
    queryFn: ({ signal }) => api.currentUser(signal),
    staleTime: 0,
  });
  if (profile.isPending) return <Skeleton label={t(locale, "loading")} />;
  if (profile.isError)
    return (
      <StatePanel
        type="error"
        title={t(locale, "loadFailed")}
        action={
          <Button onClick={() => void profile.refetch()}>
            {t(locale, "retry")}
          </Button>
        }
      />
    );
  const user = profile.data;
  return (
    <>
      <PageHeading
        eyebrow={user.membership.school}
        title={accountLabel(locale, "title")}
      />
      <Card>
        <dl className="account-facts">
          <dt>{t(locale, "email")}</dt>
          <dd>{user.email}</dd>
          <dt>{accountLabel(locale, "role")}</dt>
          <dd>{accountLabel(locale, user.membership.role)}</dd>
          <dt>{accountLabel(locale, "permissions")}</dt>
          <dd>
            <ul>
              {user.permissions.map((p) => (
                <li key={p}>{accountLabel(locale, p)}</li>
              ))}
            </ul>
          </dd>
        </dl>
      </Card>
    </>
  );
}
