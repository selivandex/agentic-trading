/** @format */

import { useQuery } from "@apollo/client";
import { Select } from "@/shared/base/select/select";
import { gql } from "@apollo/client";
import type { User } from "@/entities/user";
import type { CrudFormFieldRenderProps, CrudEntity } from "@/shared/lib/crud";

interface GetUsersResponse {
  users: User[];
}

// Query for getting all users
const GET_USERS_QUERY = gql`
  query GetAllUsers {
    users {
      id
      firstName
      lastName
      email
      telegramUsername
    }
  }
`;

/**
 * Reusable user select field component for CRUD forms
 *
 * @example
 * ```tsx
 * // In CRUD config:
 * formFields: [
 *   {
 *     name: "userID",
 *     label: "User",
 *     type: "custom",
 *     helperText: "Select the user",
 *     validation: z.string().uuid("Please select a user"),
 *     render: (props) => <UserSelectField {...props} />,
 *     colSpan: 12,
 *   },
 * ]
 * ```
 */
export function UserSelectField<TEntity extends CrudEntity = CrudEntity>({
  field,
  value,
  onChange,
  error,
  disabled,
}: CrudFormFieldRenderProps<TEntity>) {
  const { data, loading } = useQuery<GetUsersResponse>(GET_USERS_QUERY);

  const users = data?.users ?? [];

  // Format user display name
  const getUserLabel = (user: User): string => {
    if (user.firstName && user.lastName) {
      return `${user.firstName} ${user.lastName}`;
    }
    if (user.firstName) {
      return user.firstName;
    }
    if (user.telegramUsername) {
      return `@${user.telegramUsername}`;
    }
    if (user.email) {
      return user.email;
    }
    return user.id;
  };

  return (
    <Select
      label={field.label}
      placeholder={loading ? "Loading users..." : "Select a user"}
      hint={field.helperText}
      isInvalid={!!error}
      isDisabled={disabled || loading}
      selectedKey={value as string}
      onSelectionChange={onChange}
    >
      {users.map((user) => (
        <Select.Item key={user.id} id={user.id}>
          {getUserLabel(user)}
        </Select.Item>
      ))}
    </Select>
  );
}

