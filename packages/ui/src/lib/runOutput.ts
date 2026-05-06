import type { RunOutputSection } from "@app/types";
import type { Phase, RunEvent } from "@api";

const outputEventTypes = new Set(["confirmation_output", "phase_output"]);

export function formatRunOutputSections(
  events: RunEvent[],
  phases: Phase[] = [],
): RunOutputSection[] {
  const sections = new Map<string, RunOutputSection>();
  const phaseOrder = new Map(phases.map((phase, index) => [phase.id, index + 1]));
  const phaseNames = new Map(phases.map((phase) => [phase.id, phase.name]));

  for (const event of events) {
    updateSectionStatus(sections, event, phaseOrder, phaseNames);

    if (!outputEventTypes.has(event.type) || !event.message) {
      continue;
    }

    const section = sectionForEvent(sections, event, phaseOrder, phaseNames);
    section.code += event.message;
  }

  return [...sections.values()].filter((section) => section.code.length > 0);
}

function updateSectionStatus(
  sections: Map<string, RunOutputSection>,
  event: RunEvent,
  phaseOrder: Map<string, number>,
  phaseNames: Map<string, string>,
) {
  if (event.type.startsWith("phase_") && event.phaseId) {
    const section = ensurePhaseSection(sections, event, phaseOrder, phaseNames);

    if (event.status) {
      section.status = event.status;
    }

    return;
  }

  if (event.type.startsWith("run_")) {
    const section = ensureSection(sections, "workflow", "Workflow", "Workflow");

    if (event.status) {
      section.status = event.status;
    }
  }
}

function sectionForEvent(
  sections: Map<string, RunOutputSection>,
  event: RunEvent,
  phaseOrder: Map<string, number>,
  phaseNames: Map<string, string>,
) {
  if (event.type === "phase_output" && event.phaseId) {
    return ensurePhaseSection(sections, event, phaseOrder, phaseNames);
  }

  return ensureSection(sections, "confirmation", "Confirmation", "Confirmation");
}

function ensurePhaseSection(
  sections: Map<string, RunOutputSection>,
  event: RunEvent,
  phaseOrder: Map<string, number>,
  phaseNames: Map<string, string>,
) {
  const id = event.phaseId ?? "unknown-phase";
  const step = phaseOrder.get(id);
  const name = event.phaseName || phaseNames.get(id) || "Workflow step";
  const label = step ? `Step ${step}: ${name}` : name;

  return ensureSection(sections, `phase:${id}`, label, "Step");
}

function ensureSection(
  sections: Map<string, RunOutputSection>,
  id: string,
  label: string,
  context: string,
) {
  const existing = sections.get(id);

  if (existing) {
    return existing;
  }

  const section: RunOutputSection = {
    id,
    label,
    context,
    status: "running",
    code: "",
  };

  sections.set(id, section);

  return section;
}
