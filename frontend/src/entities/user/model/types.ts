/** @format */

/**
 * User Entity Types
 *
 * Core user entity with settings and profile information
 */

import type { CrudEntity } from "@/shared/lib/crud";

export interface User extends CrudEntity {
  id: string;
  telegramID?: string | null;
  telegramUsername?: string | null;
  email?: string | null;
  firstName: string;
  lastName: string;
  languageCode: string;
  isActive: boolean;
  isPremium: boolean;
  limitProfileID?: string | null;
  settings: UserSettings;
  createdAt: string;
  updatedAt: string;
}

export interface UserSettings {
  defaultAIProvider: string;
  defaultAIModel: string;
  riskLevel: RiskLevel;
  maxPositions: number;
  maxPortfolioRisk: number;
  maxDailyDrawdown: number;
  maxConsecutiveLoss: number;
  notificationsOn: boolean;
  dailyReportTime: string;
  timezone: string;
  circuitBreakerOn: boolean;
  maxPositionSizeUSD: number;
  maxTotalExposureUSD: number;
  minPositionSizeUSD: number;
  maxLeverageMultiple: number;
  allowedExchanges: string[];
}

export enum RiskLevel {
  CONSERVATIVE = "CONSERVATIVE",
  MODERATE = "MODERATE",
  AGGRESSIVE = "AGGRESSIVE",
}

export interface UpdateUserSettingsInput {
  defaultAIProvider?: string;
  defaultAIModel?: string;
  riskLevel?: RiskLevel;
  maxPositions?: number;
  maxPortfolioRisk?: number;
  maxDailyDrawdown?: number;
  maxConsecutiveLoss?: number;
  notificationsOn?: boolean;
  dailyReportTime?: string;
  timezone?: string;
  circuitBreakerOn?: boolean;
  maxPositionSizeUSD?: number;
  maxTotalExposureUSD?: number;
  minPositionSizeUSD?: number;
  maxLeverageMultiple?: number;
  allowedExchanges?: string[];
}
