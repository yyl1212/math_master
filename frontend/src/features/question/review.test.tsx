import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, it, expect, vi } from "vitest";
import { ReviewPanel } from "./review-panel";
import { componentSubmission, componentInstances, fixtureID } from "./test-fixtures";
const mocks = vi.hoisted(() => ({ request: vi.fn(), push: vi.fn(), refresh: vi.fn() }));
vi.mock("@/lib/question/client", () => ({ requestQuestion: mocks.request }));
vi.mock("next/navigation", () => ({ useRouter: () => mocks }));
beforeEach(() => Object.values(mocks).forEach(m => m.mockReset()));
it("ReviewPanel", async () => {
    const sub = componentSubmission(), instances = componentInstances();
    mocks.request.mockResolvedValue({ ok: true, data: { items: [instances[1]], total: 2, limit: 1, offset: 1 } });
    render(<ReviewPanel submission={sub} initialInstances={{ items: [instances[0]], total: 2, limit: 1, offset: 0 }} user={{ id: fixtureID, username: "reviewer", roles: ["learner", "reviewer"], mustChangePassword: false }}/>);
    expect(screen.getAllByRole("checkbox")).toHaveLength(6);
    expect(screen.getByText(/Verify historical authorship/)).toBeVisible();
    expect(screen.getByText("original/fractions.json")).toBeVisible();
    expect(screen.getByText("Add rational numbers. 比较分数")).toBeVisible();
    expect(screen.getByText("fixed-one")).toBeVisible();
    expect(screen.getByRole("button", { name: "Approve submission" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Next instances" }));
    expect(await screen.findByText("fixed-two")).toBeVisible();
    expect(mocks.request.mock.calls[0][0]).toEqual({ kind: "listInstances", id: sub.id, query: { limit: 1, offset: 1 } });
    expect(mocks.request.mock.calls.some(a => a[0].kind === "readDraft")).toBe(false);
    screen.getAllByRole("checkbox").forEach(c => fireEvent.click(c));
    fireEvent.change(screen.getByLabelText("Independence statement"), { target: { value: "Independent review of the original authorship." } });
    fireEvent.change(screen.getByLabelText("Review note"), { target: { value: "Every answer, objective and source checked." } });
    expect(screen.getByRole("button", { name: "Approve submission" })).toBeDisabled();
    fireEvent.change(screen.getByLabelText("Generation review statement"), { target: { value: "No templates are present; generation does not apply." } });
    await waitFor(() => expect(screen.getByRole("button", { name: "Approve submission" })).toBeEnabled());
});

it("TemplateApprovalAllowsEmptyGenerationStatement", async () => {
 const sub = componentSubmission(); sub.frozen.questionPackage.templates = (await import("./test-fixtures")).componentDraft().questionPackage.templates;
 render(<ReviewPanel submission={sub} initialInstances={{ items: [], total: 0, limit: 1, offset: 0 }} user={{ id: fixtureID, username: "reviewer", roles: ["learner", "reviewer"], mustChangePassword: false }}/>);
 screen.getAllByRole("checkbox").forEach(c => fireEvent.click(c));
 fireEvent.change(screen.getByLabelText("Independence statement"), { target: { value: "Independently checked all authors and generated questions." } });
 fireEvent.change(screen.getByLabelText("Review note"), { target: { value: "All six checks cover every finite generated instance." } });
 expect(screen.getByRole("button", { name: "Approve submission" })).toBeEnabled();
});
