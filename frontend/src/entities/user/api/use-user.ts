/** @format */

import { useQuery, useMutation } from "@apollo/client";
import {
  GET_ME_QUERY,
  GET_USER_QUERY,
  GET_USER_BY_TELEGRAM_ID_QUERY,
  UPDATE_USER_SETTINGS_MUTATION,
  SET_USER_ACTIVE_MUTATION,
} from "@/entities/user";
import type { User, UpdateUserSettingsInput } from "@/entities/user";

/**
 * Hook to get current authenticated user
 */
export function useMe() {
  return useQuery<{ me: User | null }>(GET_ME_QUERY);
}

/**
 * Hook to get user by ID
 */
export function useUser(id: string) {
  return useQuery<{ user: User | null }>(GET_USER_QUERY, {
    variables: { id },
    skip: !id,
  });
}

/**
 * Hook to get user by telegram ID
 */
export function useUserByTelegramID(telegramID: string) {
  return useQuery<{ userByTelegramID: User | null }>(
    GET_USER_BY_TELEGRAM_ID_QUERY,
    {
      variables: { telegramID },
      skip: !telegramID,
    }
  );
}

/**
 * Hook to update user settings
 */
export function useUpdateUserSettings() {
  return useMutation<
    { updateUserSettings: User },
    { userID: string; input: UpdateUserSettingsInput }
  >(UPDATE_USER_SETTINGS_MUTATION);
}

/**
 * Hook to set user active/inactive
 */
export function useSetUserActive() {
  return useMutation<
    { setUserActive: User },
    { userID: string; isActive: boolean }
  >(SET_USER_ACTIVE_MUTATION);
}
