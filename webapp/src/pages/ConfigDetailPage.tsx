import { useTranslation } from "react-i18next";
import { Link, useParams } from "react-router-dom";
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

export function ConfigDetailPage() {
  const { t } = useTranslation();
  const params = useParams<{ id: string }>();
  const id = params.id ?? "";

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
        <h1 className="text-xl font-semibold">{id}</h1>
      </div>

      {ruleQuery.isLoading ? (
        <Card>
          <CardContent className="py-8 text-sm text-muted-foreground">
            {t("common.loading")}
          </CardContent>
        </Card>
      ) : ruleQuery.isError ? (
        <Card>
          <CardContent className="py-8 text-sm text-destructive">
            {(ruleQuery.error as Error).message}
          </CardContent>
        </Card>
      ) : ruleQuery.data ? (
        <Tabs defaultValue="overview">
          <TabsList>
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
                <OverviewTab id={id} rule={ruleQuery.data} />
              </CardContent>
            </Card>
          </TabsContent>
          <TabsContent value="sources">
            <Card>
              <CardHeader>
                <CardTitle>{t("sources.title")}</CardTitle>
                <CardDescription>{t("sources.description")}</CardDescription>
              </CardHeader>
              <CardContent>
                <SourcesTab id={id} rule={ruleQuery.data} />
              </CardContent>
            </Card>
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
