import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { it, expect, vi } from "vitest";
import { CoveragePanel } from "./coverage-panel";
import { coverageReport } from "./test-fixtures";
const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("@/lib/question/client", () => ({ requestQuestion: mocks.request }));
it("CoveragePanelUsesEffectiveLimitAndFreshSnapshot", async () => {
    const report = coverageReport();
    render(<CoveragePanel initial={report}/>);
    expect(screen.getByText(/Published trusted instances: 5/)).toBeVisible();
    expect(screen.getByText("Ready for five questions")).toBeVisible();
    mocks.request.mockResolvedValue({ ok: true, data: { ...report, questionHead: null, nodes: { ...report.nodes, offset: 1 } } });
    fireEvent.click(screen.getByRole("button", { name: "Next coverage nodes" }));
    await waitFor(() => expect(mocks.request.mock.calls[0][0]).toEqual({ kind: "readCoverage", query: { limit: 1, offset: 1 } }));
    await screen.findByText(/Snapshots changed/);
    expect(screen.queryByText("Ready for five questions")).not.toBeInTheDocument();
});
