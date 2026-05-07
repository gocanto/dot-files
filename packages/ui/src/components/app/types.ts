import type { Component } from "vue";
import type { Phase } from "@api";

export type SectionId = "template" | "current" | "update" | "status" | "settings" | "logs";

export interface NavItem {
  id: SectionId;
  label: string;
  icon: Component;
  count: number | null;
}

export interface StepMeta {
  id: "template" | "current" | "update";
  title: string;
  summary: string;
  emptyMessage: string;
}

export interface SettingsGroup {
  id: string;
  label: string;
  icon: Component;
  count: number;
}

export interface SelectOption {
  value: string;
  label: string;
  missing: boolean;
}

export interface SavedField {
  key: string;
  label: string;
  saved: string;
  pending: string;
}

export type DisplayPhase = Phase & {
  status: string;
};

export interface RunOutputSection {
  id: string;
  label: string;
  context: string;
  status: string;
  code: string;
}
