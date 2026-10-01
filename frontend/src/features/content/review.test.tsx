import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, it, expect, vi } from "vitest";
import { ReviewPanel } from "./review-panel";
import { submissionView, fixtureID, otherID } from "@/lib/content/test-fixtures";
import { contentFailure } from "@/lib/content/schemas";
const mocks = vi.hoisted(() => ({ request: vi.fn(), refresh: vi.fn(), push: vi.fn() }));
vi.mock("@/lib/content/client", () => ({ contentRequest: mocks.request }));
vi.mock("next/navigation", () => ({ useRouter: () => mocks }));
const reviewer = { id: fixtureID, username: "independent_reviewer", roles: ["learner", "reviewer"] as ("learner" | "reviewer")[], mustChangePassword: false };
beforeEach(() => Object.values(mocks).forEach(m => m.mockReset()));
it("TestIndependentReviewAuthor", () => { const s = submissionView(); s.frozen.authorIds.push(fixtureID); render(<ReviewPanel submission={s} user={reviewer}/>); expect(screen.queryByRole("button", { name: "Approve submission" })).not.toBeInTheDocument(); expect(screen.getByText(/Authors cannot review/)).toBeVisible(); expect(screen.queryByLabelText("Knowledge 1 statement")).not.toBeInTheDocument(); });
it("TestIndependentReviewUI", async () => { render(<ReviewPanel submission={submissionView()} user={reviewer}/>); expect(screen.getByRole("button", { name: "Approve submission" })).toBeDisabled(); for (const label of ["Mathematics", "Explanations", "Relationships", "Sources", "Illustrations"])
    fireEvent.click(screen.getByLabelText(label)); fireEvent.change(screen.getByLabelText("Independence statement"), { target: { value: "I independently reviewed this content." } }); fireEvent.change(screen.getByLabelText("Review note"), { target: { value: "All checks have been independently completed." } }); mocks.request.mockResolvedValue(contentFailure()); fireEvent.click(screen.getByRole("button", { name: "Approve submission" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Service temporarily unavailable.")); expect(screen.queryByText("Review saved.")).not.toBeInTheDocument(); expect(mocks.request.mock.calls[0][1].checks).toEqual({ mathematics: true, explanations: true, relationships: true, sources: true, illustrations: true }); });
it("TestIndependentReviewReturnNote", () => { render(<ReviewPanel submission={submissionView()} user={reviewer}/>); expect(screen.getByRole("button", { name: "Return for changes" })).toBeDisabled(); fireEvent.change(screen.getByLabelText("Review note"), { target: { value: "Please correct the second explanation." } }); expect(screen.getByRole("button", { name: "Return for changes" })).toBeEnabled(); });
