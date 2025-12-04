/** @format */

import {
  useGetScreensQuery,
  useCreateScreenMutation,
  useUpdateScreenMutation,
  useDeleteScreenMutation,
  useArchiveScreenMutation,
  useUnarchiveScreenMutation,
  usePublishScreenMutation,
} from "@/shared/api/generated/graphql";

/**
 * Hook to get single screen by UID
 * Uses screens list query and filters by UID on client side
 * since backend doesn't have a single screen query
 */
export const useScreen = (screenId?: string) => {
  const { data, loading, error, refetch } = useGetScreensQuery({
    variables: { first: 100 }, // Load enough screens to find the one we need
    skip: !screenId, // Skip query if no screenId provided
  });

  // Find the specific screen by UID (screenId parameter contains UID from URL)
  const screen = screenId
    ? data?.screens?.edges
        ?.map((edge) => edge?.node)
        .filter((node): node is NonNullable<typeof node> => node != null)
        .find((node) => node.uid === screenId) ?? null
    : null;

  return {
    screen,
    loading,
    error,
    refetch,
  };
};

/**
 * Hook to get screens list
 */
export const useScreens = (
  options?: {
    first?: number;
    after?: string;
  }
) => {
  const { data, loading, error, refetch } = useGetScreensQuery({
    variables: options,
  });

  return {
    screens: data?.screens?.edges?.map((edge) => edge?.node).filter((node): node is NonNullable<typeof node> => node != null) ?? [],
    pageInfo: data?.screens?.pageInfo,
    loading,
    error,
    refetch,
  };
};


/**
 * Hook to create screen
 */
export const useCreateScreen = () => {
  const [createScreen, { loading, error }] = useCreateScreenMutation({
    refetchQueries: ["GetScreens"],
  });

  return {
    createScreen,
    loading,
    error,
  };
};

/**
 * Hook to update screen
 */
export const useUpdateScreen = () => {
  const [updateScreen, { loading, error }] = useUpdateScreenMutation({
    refetchQueries: ["GetScreens"],
  });

  return {
    updateScreen,
    loading,
    error,
  };
};

/**
 * Hook to delete screen
 */
export const useDeleteScreen = () => {
  const [deleteScreen, { loading, error }] = useDeleteScreenMutation({
    refetchQueries: ["GetScreens"],
  });

  return {
    deleteScreen,
    loading,
    error,
  };
};

/**
 * Hook to archive screen
 */
export const useArchiveScreen = () => {
  const [archiveScreen, { loading, error }] = useArchiveScreenMutation({
    refetchQueries: ["GetScreens"],
  });

  return {
    archiveScreen,
    loading,
    error,
  };
};

/**
 * Hook to unarchive screen
 */
export const useUnarchiveScreen = () => {
  const [unarchiveScreen, { loading, error }] = useUnarchiveScreenMutation({
    refetchQueries: ["GetScreens"],
  });

  return {
    unarchiveScreen,
    loading,
    error,
  };
};

/**
 * Hook to publish screen
 */
export const usePublishScreen = () => {
  const [publishScreen, { loading, error }] = usePublishScreenMutation({
    refetchQueries: ["GetScreens"],
  });

  return {
    publishScreen,
    loading,
    error,
  };
};


