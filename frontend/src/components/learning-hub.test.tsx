import { it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { LearningHub } from "./learning-hub";
import { catalogueFixture } from "../../tests/fixtures";
it("showsRealCatalogueWithoutPersonalProgress", () => {
  render(<LearningHub result={{ ok: true, data: catalogueFixture() }} />);
  expect(
    screen.getByRole("link", { name: "Explore knowledge" }),
  ).toHaveAttribute("href", "/knowledge");
  expect(
    screen.getByRole("heading", { name: "16 learning domains" }),
  ).toBeVisible();
  expect(screen.getAllByText("In development")).toHaveLength(16);
  expect(screen.getByText("Elementary Mathematics")).toBeVisible();
  expect(screen.queryByText(/12\s*\/\s*30/)).not.toBeInTheDocument();
  expect(
    screen.queryByText(/hours learned|unlocked|mastered/i),
  ).not.toBeInTheDocument();
});
it("distinguishesEmptyFromUnavailable", () => {
  const { rerender } = render(
    <LearningHub
      result={{
        ok: true,
        data: { items: [], total: 0, limit: 100, offset: 0 },
      }}
    />,
  );
  expect(screen.getByText("The catalogue is being prepared.")).toBeVisible();
  rerender(<LearningHub result={{ ok: false, kind: "unavailable" }} />);
  expect(screen.getByText("Content is temporarily unavailable.")).toBeVisible();
  expect(
    screen.queryByText("The catalogue is being prepared."),
  ).not.toBeInTheDocument();
});
it("doesNotSumOverlappingDomainKnowledge", () => {
  const data = catalogueFixture();
  data.items[0].publishedKnowledgeCount = 1;
  data.items[0].contentStatus = "published";
  data.items[1].publishedKnowledgeCount = 1;
  data.items[1].contentStatus = "published";
  render(<LearningHub result={{ ok: true, data }} />);
  expect(screen.getAllByText("1 published knowledge point")).toHaveLength(2);
  expect(
    screen.queryByText("2 published knowledge points"),
  ).not.toBeInTheDocument();
});
