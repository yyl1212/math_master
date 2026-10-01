import { it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { KnowledgeMap } from "./knowledge-map";
import { DomainView } from "./domain-view";
import { catalogueFixture } from "../../../tests/fixtures";
it("filtersConstructionStateWithoutInventingUnlocks", () => {
  const data = catalogueFixture();
  data.items[1].contentStatus = "published";
  data.items[1].publishedKnowledgeCount = 10;
  const { rerender } = render(
    <KnowledgeMap result={{ ok: true, data }} q="" status="published" />,
  );
  expect(
    screen.getByRole("link", { name: "Elementary Mathematics" }),
  ).toHaveAttribute("href", "/domains/elementary-mathematics");
  expect(
    screen.queryByRole("link", { name: "Linear Algebra" }),
  ).not.toBeInTheDocument();
  expect(
    screen.queryByText(/locked|unlocked|mastered/i),
  ).not.toBeInTheDocument();
  rerender(<KnowledgeMap result={{ ok: true, data }} q="" status="planned" />);
  expect(screen.getByRole("link", { name: "Linear Algebra" })).toBeVisible();
  expect(
    screen.queryByRole("link", { name: "Elementary Mathematics" }),
  ).not.toBeInTheDocument();
  expect(
    screen.getByRole("searchbox", { name: "Search learning domains" }),
  ).toBeVisible();
});
it("showsTopicsAndRelatedDomainsSeparatelyFromPaths", () => {
  const data = { ...catalogueFixture().items[1], paths: [] };
  const { rerender } = render(<DomainView result={{ ok: true, data }} />);
  expect(
    screen.getByRole("heading", { name: "Topics to explore" }),
  ).toBeVisible();
  expect(screen.getByText("Arithmetic")).toBeVisible();
  expect(
    screen.getByRole("heading", { name: "Related domains" }),
  ).toBeVisible();
  expect(screen.getByText("Learning paths are in development.")).toBeVisible();
  rerender(
    <DomainView
      result={{
        ok: true,
        data: {
          ...data,
          contentStatus: "published",
          paths: [
            {
              id: "fractions-path",
              version: 1,
              title: "Fractions",
              titleZh: "分数",
            },
          ],
        },
      }}
    />,
  );
  expect(screen.getByRole("link", { name: /Fractions/ })).toHaveAttribute(
    "href",
    "/paths/fractions-path",
  );
});
