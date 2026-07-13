import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Link,
  useBlocker,
  useParams,
  useSearchParams,
} from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import { getConfig } from "@/api/configs";
import { OverviewTab } from "@/pages/detail/OverviewTab";
import { SourcesTab } from "@/pages/detail/SourcesTab";
import { RuleProvidersTab } from "@/pages/detail/RuleProvidersTab";
import { RuleGroupsTab } from "@/pages/detail/RuleGroupsTab";
import { SubscriptionTab } from "@/pages/detail/SubscriptionTab";

const DETAIL_TABS = [
  "overview",
  "sources",
  "providers",
  "groups",
  "subscription",
] as const;

type DetailTab = (typeof DETAIL_TABS)[number];

export function ConfigDetailPage() {
  const { t } = useTranslation();
  const params = useParams<{ id: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const [dirtyTabs, setDirtyTabs] = useState({ overview: false, sources: false });
  const id = params.id ?? "";
  const requestedTab = searchParams.get("tab");
  const activeTab: DetailTab = DETAIL_TABS.includes(requestedTab as DetailTab)
    ? (requestedTab as DetailTab)
    : "overview";
  const hasUnsavedChanges = dirtyTabs.overview || dirtyTabs.sources;

  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      hasUnsavedChanges &&
      (currentLocation.pathname !== nextLocation.pathname ||
        currentLocation.search !== nextLocation.search),
  );

  useEffect(() => {
    if (blocker.state !== "blocked") return;
    if (window.confirm(t("detail.unsavedConfirm"))) {
      blocker.proceed();
    } else {
      blocker.reset();
    }
  }, [blocker, t]);

  useEffect(() => {
    if (!hasUnsavedChanges) return;
    const onBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, [hasUnsavedChanges]);

  const setOverviewDirty = useCallback((dirty: boolean) => {
    setDirtyTabs((previous) =>
      previous.overview === dirty ? previous : { ...previous, overview: dirty },
    );
  }, []);

  const setSourcesDirty = useCallback((dirty: boolean) => {
    setDirtyTabs((previous) =>
      previous.sources === dirty ? previous : { ...previous, sources: dirty },
    );
  }, []);

  const onTabChange = (tab: string) => {
    setSearchParams((previous) => {
      const next = new URLSearchParams(previous);
      next.set("tab", tab);
      return next;
    });
  };

  const ruleQuery = useQuery({
    queryKey: ["configs", id],
    queryFn: ({ signal }) => getConfig(id, signal),
    enabled: !!id,
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" asChild>
          <Link to="/configs" aria-label={t("common.back")}>
            <ArrowLeft className="h-4 w-4" aria-hidden />
          </Link>
        </Button>
        <h1
          className="min-w-0 break-all text-xl font-semibold"
          translate="no"
        >
          {id}
        </h1>
      </div>

      {ruleQuery.isLoading ? (
        <Card>
          <CardContent
            role="status"
            aria-live="polite"
            className="py-8 text-sm text-muted-foreground"
          >
            {t("common.loading")}
          </CardContent>
        </Card>
      ) : ruleQuery.isError ? (
        <Card>
          <CardContent role="alert" className="py-8 text-sm text-destructive">
            {(ruleQuery.error as Error).message}
          </CardContent>
        </Card>
      ) : ruleQuery.data ? (
        <Tabs value={activeTab} onValueChange={onTabChange}>
          <TabsList className="max-w-full justify-start overflow-x-auto">
            <TabsTrigger value="overview">{t("detail.tabOverview")}</TabsTrigger>
            <TabsTrigger value="sources">{t("detail.tabSources")}</TabsTrigger>
            <TabsTrigger value="providers">
              {t("detail.tabRuleProviders")}
            </TabsTrigger>
            <TabsTrigger value="groups">
              {t("detail.tabRuleGroups")}
            </TabsTrigger>
            <TabsTrigger value="subscription">
              {t("detail.tabSubscription")}
            </TabsTrigger>
          </TabsList>
          <TabsContent value="overview">
            <Card>
              <CardHeader>
                <CardTitle>{t("detail.tabOverview")}</CardTitle>
                <CardDescription>{t("app.subtitle")}</CardDescription>
              </CardHeader>
              <CardContent>
                <OverviewTab
                  id={id}
                  rule={ruleQuery.data}
                  onDirtyChange={setOverviewDirty}
                />
              </CardContent>
            </Card>
          </TabsContent>
          <TabsContent value="sources">
            <SourcesTab
              id={id}
              rule={ruleQuery.data}
              onDirtyChange={setSourcesDirty}
            />
          </TabsContent>
          <TabsContent value="providers">
            <Card>
              <CardHeader>
                <CardTitle>{t("providers.title")}</CardTitle>
                <CardDescription>{t("providers.description")}</CardDescription>
              </CardHeader>
              <CardContent>
                <RuleProvidersTab id={id} />
              </CardContent>
            </Card>
          </TabsContent>
          <TabsContent value="groups">
            <Card>
              <CardHeader>
                <CardTitle>{t("groups.title")}</CardTitle>
                <CardDescription>{t("groups.description")}</CardDescription>
              </CardHeader>
              <CardContent>
                <RuleGroupsTab id={id} />
              </CardContent>
            </Card>
          </TabsContent>
          <TabsContent value="subscription">
            <Card>
              <CardHeader>
                <CardTitle>{t("subscription.title")}</CardTitle>
                <CardDescription>
                  {t("subscription.description")}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <SubscriptionTab id={id} />
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      ) : null}
    </div>
  );
}
