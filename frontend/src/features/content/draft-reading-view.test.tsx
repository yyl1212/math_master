import { fireEvent, render, screen, within } from "@testing-library/react";
import { expect, it } from "vitest";
import { DraftReadingView } from "./draft-reading-view";
import { draftView } from "@/lib/content/test-fixtures";

function readingDraft() {
    const view = draftView();
    view.revision = 7;
    view.package.knowledge[0].conditions = ["The denominator must be nonzero."];
    view.package.knowledge[0].proof = "Equal parts give the same rational value.";
    const equivalent = structuredClone(view.package.knowledge[0]);
    equivalent.version = 2;
    equivalent.title = "Equivalent fractions";
    equivalent.titleZh = "等值分数";
    equivalent.statement = "Multiplying numerator and denominator by the same nonzero integer preserves the value.";
    const whole = structuredClone(equivalent);
    whole.id = "whole-numbers";
    whole.version = 1;
    whole.title = "Whole numbers";
    whole.titleZh = "整数";
    whole.statement = "This point introduces whole numbers.";
    const unit = structuredClone(view.package.units[0]);
    unit.version = 2;
    unit.knowledge.version = 2;
    unit.assetIds = [];
    unit.angles = [{ kind: "formal", body: "An equivalent representation uses the second unit version." }];
    view.package.knowledge.push(equivalent, whole);
    view.package.units.push(unit);
    view.package.paths = [{ id: "fractions-path", version: 1, domainIds: ["elementary-mathematics"], title: "Fractions route", titleZh: "分数路线", nodes: [{ id: "fractions", version: 1 }, { id: "fractions", version: 2 }] }];
    return view;
}

it("reads the saved revision with matching unit versions, sources, proof and private illustration", () => {
    const view = readingDraft();
    const before = JSON.stringify(view);
    render(<DraftReadingView initial={view}/>);
    expect(screen.getByText(/Saved revision 7.*editing/)).toBeVisible();
    expect(screen.getByRole("heading", { name: /fractions-unit · Version 1/ })).toBeVisible();
    expect(screen.queryByText("An equivalent representation uses the second unit version.")).not.toBeInTheDocument();
    expect(screen.getByText("The denominator must be nonzero.")).toBeVisible();
    expect(screen.getByText("Equal parts give the same rational value.")).toBeVisible();
    expect(screen.getByText("Original fractions test lesson")).toBeVisible();
    expect(screen.getByRole("heading", { name: /Fractions route/ })).toBeVisible();
    expect(screen.getByRole("img", { name: "Equal halves" })).toHaveAttribute("src", "/api/v1/content/drafts/11111111-1111-4111-8111-111111111111/assets/c0e9cba36c9276c123681183bd3b62cbd073f49adfa7a50a34f65b435f946b42");
    expect(screen.getByLabelText("Preview reference")).toHaveValue("Draft: 11111111-1111-4111-8111-111111111111\nSaved revision: 7\nPackage: workflow-ready v1\nKnowledge: fractions v1");
    fireEvent.click(screen.getByRole("button", { name: /Equivalent fractions/ }));
    expect(screen.getByRole("heading", { name: /fractions-unit · Version 2/ })).toBeVisible();
    expect(screen.queryByRole("heading", { name: /fractions-unit · Version 1/ })).not.toBeInTheDocument();
    expect(screen.getByText("An equivalent representation uses the second unit version.")).toBeVisible();
    expect(screen.queryByRole("img", { name: "Equal halves" })).not.toBeInTheDocument();
    expect(screen.getByLabelText("Preview reference")).toHaveValue("Draft: 11111111-1111-4111-8111-111111111111\nSaved revision: 7\nPackage: workflow-ready v1\nKnowledge: fractions v2");
    expect(JSON.stringify(view)).toBe(before);
});

it("searches English and Chinese titles and IDs without leaving an unrelated point on screen", () => {
    render(<DraftReadingView initial={readingDraft()}/>);
    const search = screen.getByRole("searchbox", { name: "Search knowledge points" });
    const navigation = screen.getByRole("navigation", { name: "Knowledge points" });
    fireEvent.change(search, { target: { value: " EQUIVALENT " } });
    expect(within(navigation).getAllByRole("button")).toHaveLength(1);
    expect(screen.getByText("An equivalent representation uses the second unit version.")).toBeVisible();
    fireEvent.change(search, { target: { value: "整数" } });
    expect(screen.getByText("This point introduces whole numbers.")).toBeVisible();
    fireEvent.change(search, { target: { value: "fractions" } });
    expect(within(navigation).getAllByRole("button")).toHaveLength(2);
    fireEvent.change(search, { target: { value: "no-such-knowledge" } });
    expect(screen.getByText(/No matching knowledge points/)).toBeVisible();
    expect(screen.queryByRole("heading", { name: "Statement" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next knowledge point" })).toBeDisabled();
    fireEvent.change(search, { target: { value: "" } });
    expect(within(navigation).getAllByRole("button")).toHaveLength(3);
    expect(screen.getByRole("heading", { name: /fractions-unit · Version 1/ })).toBeVisible();
});

it("switches within the current search results and provides general site feedback", () => {
    render(<DraftReadingView initial={readingDraft()}/>);
    expect(screen.getByRole("button", { name: "Previous knowledge point" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Next knowledge point" }));
    expect(screen.getByRole("button", { name: /Equivalent fractions/ })).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByRole("button", { name: "Next knowledge point" }));
    expect(screen.getByText("This point introduces whole numbers.")).toBeVisible();
    expect(screen.getByRole("button", { name: "Next knowledge point" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Previous knowledge point" }));
    expect(screen.getByText("An equivalent representation uses the second unit version.")).toBeVisible();
    expect(screen.getByRole("link", { name: "Give site feedback" })).toHaveAttribute("href", "/feedback/new?kind=site&area=other");
    expect(screen.getByLabelText("Preview reference")).toHaveAttribute("readonly");
    expect(screen.queryByRole("button", { name: /Submit|Complete|Mark as learned/ })).not.toBeInTheDocument();
});

it("shows an empty saved draft without inventing a selected knowledge point", () => {
    const view = draftView();
    view.package.knowledge = [];
    view.package.units = [];
    render(<DraftReadingView initial={view}/>);
    expect(screen.getByText(/No knowledge points in this saved draft/)).toBeVisible();
    expect(screen.getByRole("button", { name: "Next knowledge point" })).toBeDisabled();
    expect(screen.queryByRole("heading", { name: "Statement" })).not.toBeInTheDocument();
    expect(screen.getByLabelText("Preview reference")).not.toHaveValue(expect.stringContaining("undefined"));
});
