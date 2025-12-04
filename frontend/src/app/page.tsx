/**
 * Root page - redirects to last visited context or first available org/project
 *
 * @format
 */

import { redirect } from "next/navigation";
import { auth } from "@/shared/lib/auth";
import { getLastContext } from "@/shared/lib/last-context";

// Extended types to include projects (until GraphQL codegen runs)
type ProjectNode = {
  id: string;
  name: string;
  slug: string;
};

type OrganizationWithProjects = {
  id: string;
  name: string;
  slug: string;
  status: string;
  projects?: {
    edges?: Array<{
      node?: ProjectNode;
    } | null> | null;
  } | null;
};

type MembershipWithProjects = {
  id: string;
  active: boolean;
  role: {
    id: string;
    key: string;
    name: string;
  };
  organization: OrganizationWithProjects;
};

export default async function Home() {
  // Check if user is authenticated
  const session = await auth();

  // If not authenticated, redirect to login
  if (!session?.user) {
    redirect("/login");
  }

  // Get last visited context from cookies
  const { organizationId, projectId } = await getLastContext();

  // Get user data
  const user = session.user;

  // If we have saved context, validate it before redirecting
  if (organizationId && projectId) {
    // Check if user still has access to this organization
    const hasAccess = user.memberships?.edges?.some(
      (edge) =>
        edge?.node?.active && edge.node.organization.slug === organizationId
    );

    if (hasAccess) {
      // User has access - redirect to last context
      // Note: We don't validate project here because authorization
      // will be checked on the target page
      redirect(`/${organizationId}/${projectId}/home`);
    }
    // If no access, fall through to find first available org/project
  }

  // No saved context or invalid context - find first available organization and project

  // Check if user has memberships with organizations
  if (!user.memberships?.edges || user.memberships.edges.length === 0) {
    // No organizations - redirect to onboarding
    redirect("/onboarding");
  }

  // Find first active membership
  const firstMembershipEdge = user.memberships.edges.find(
    (edge) => edge?.node?.active
  );

  if (!firstMembershipEdge?.node) {
    // No active memberships - redirect to onboarding
    redirect("/onboarding");
  }

  // Type assertion for organization with projects
  const membership =
    firstMembershipEdge.node as unknown as MembershipWithProjects;
  const organization = membership.organization;

  // Check if organization has projects
  if (
    !organization.projects?.edges ||
    organization.projects.edges.length === 0
  ) {
    // No projects - redirect to onboarding
    redirect("/onboarding");
  }

  const firstProject = organization.projects.edges[0]?.node;

  if (!firstProject) {
    // No valid project - redirect to onboarding
    redirect("/onboarding");
  }

  // Redirect to first available org/project
  redirect(`/${organization.slug}/${firstProject.slug}/home`);
}
