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

it("TestReviewShowsAllFrozenEvidence", () => {
 const s=submissionView(),k=s.frozen.package.knowledge[0];
 k.sources=[{kind:"external",author:"Original reviewer source",title:"Technical source title",url:"https://example.org/original-math",accessedAt:"2026-10-01",license:"CC0-1.0",attribution:"Original source attribution"}];
 k.relations=[{kind:"prerequisite",target:{id:"prior-knowledge",version:2}}];k.conditions=["A nonzero denominator is required."];k.proof="An exact technical proof.";
 s.frozen.package.paths=[{id:"review-route",version:3,title:"Frozen review route",titleZh:"固定复核路线",domainIds:k.domainIds,nodes:[{id:k.id,version:k.version}]}];
 s.frozen.package.units[0].counterexamples=["A labeled technical counterexample."];
 render(<ReviewPanel submission={s} user={reviewer}/>);
 expect(screen.queryByText("Technical source title",{selector:"a"})).toHaveAttribute("href","https://example.org/original-math");
 expect(screen.getByText("Original source attribution")).toBeVisible();expect(screen.getByText(/prerequisite.*prior-knowledge.*v2/)).toBeVisible();
 expect(screen.getByRole("heading",{name:/Frozen review route/})).toBeVisible();expect(screen.getByText(/review-route.*v3/)).toBeVisible();
 for(const name of ["Conditions","Learning objectives","Proof","Examples","Counterexamples"]){expect(screen.getByRole("heading",{name})).toBeVisible()}
});

it("administrator reviewer can check and approve their own fixed content", () => {
 const s = submissionView(); s.frozen.authorIds.push(fixtureID);
 render(<ReviewPanel submission={s} user={{...reviewer, roles:["learner","admin","reviewer"]}}/>);
 expect(screen.queryByRole("button",{name:"Approve submission"})).toBeInTheDocument();
 expect(screen.getByRole("heading",{name:"Administrator self-review"})).toBeVisible();
 expect(screen.getByRole("button",{name:"Approve submission"})).toBeDisabled();
 for (const label of ["Mathematics","Explanations","Relationships","Sources","Illustrations"]) fireEvent.click(screen.getByLabelText(label));
 fireEvent.change(screen.getByLabelText("Review responsibility statement"),{target:{value:"I authored this content and take responsibility for this administrator review."}});
 fireEvent.change(screen.getByLabelText("Review note"),{target:{value:"I checked all five requirements against the fixed version."}});
 expect(screen.getByRole("button",{name:"Approve submission"})).toBeEnabled();
 expect(screen.queryByLabelText("Independence statement")).not.toBeInTheDocument();
});
it("administrator without reviewer cannot approve their own content", () => {
 const s=submissionView(); s.frozen.authorIds.push(fixtureID);
 render(<ReviewPanel submission={s} user={{...reviewer,roles:["learner","admin"]}}/>);
 expect(screen.queryByRole("button",{name:"Approve submission"})).not.toBeInTheDocument();
});
