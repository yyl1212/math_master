import { it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SafeMarkdown } from "./safe-markdown";
import { KnowledgeView } from "./knowledge-view";
import type { AssetView } from "@/lib/api/types";
const asset: AssetView = {
  id: "fraction-bar",
  sha256: "a".repeat(64),
  author: "Math Master",
  license: "CC0-1.0",
  attribution: "Original fraction illustration",
  knowledge: { id: "fractions", version: 1 },
};
it("blocksHtmlUnsafeLinksAndUnboundImages", () => {
  const source =
    '<script>alert(1)</script>\n\n<iframe src="https://evil.test"></iframe>\n\n[bad](javascript:alert) [http](http://evil.test) [safe](https://example.org)\n\n![remote](https://evil.test/a.svg) ![missing](asset:unknown) ![Fractions](asset:fraction-bar)';
  const { container } = render(
    <SafeMarkdown source={source} assets={[asset]} />,
  );
  expect(container.querySelector("script,iframe")).toBeNull();
  expect(
    container.querySelector('a[href^="javascript:"],a[href^="http:"]'),
  ).toBeNull();
  expect(screen.getByRole("link", { name: "safe" })).toHaveAttribute(
    "href",
    "https://example.org",
  );
  expect(container.querySelectorAll("img")).toHaveLength(1);
  expect(screen.getByRole("img", { name: "Fractions" })).toHaveAttribute(
    "src",
    "/api/v1/assets/" + "a".repeat(64),
  );
  fireEvent.error(screen.getByRole("img", { name: "Fractions" }));
  expect(
    screen.getByText("Illustration is temporarily unavailable."),
  ).toBeVisible();
});
it("limitsMathAndKeepsConditionsSourcesAndAttribution", () => {
  const view = {
    knowledge: {
      id: "fractions",
      version: 2,
      domainIds: ["elementary-mathematics"],
      topicIds: [],
      type: "theorem" as const,
      title: "Equivalent fractions",
      titleZh: "等值分数",
      statement: "$\\frac{1}{2}=\\frac{2}{4}$",
      scope: "Rational numbers",
      system: "Rational arithmetic",
      proof: "Both represent the same rational number.",
      conditions: ["The denominator is nonzero."],
      objectives: ["Recognise equivalent fractions."],
      sources: [
        {
          kind: "original" as const,
          author: "Math Master",
          title: "Original explanation",
          url: "",
          accessedAt: "",
          license: "CC0-1.0",
          attribution: "Original work by Math Master",
        },
      ],
      relations: [],
    },
    units: [
      {
        id: "fractions-unit",
        version: 1,
        knowledge: { id: "fractions", version: 2 },
        angles: [{ kind: "visual", body: "![Fractions](asset:fraction-bar)" }],
        examples: ["One half."],
        counterexamples: [],
        assetIds: ["fraction-bar"],
      },
    ],
    assets: [{ ...asset, knowledge: { id: "fractions", version: 2 } }],
  };
  const { container, rerender } = render(
    <KnowledgeView result={{ ok: true, data: view }} />,
  );
  expect(container.querySelector(".katex")).not.toBeNull();
  expect(screen.getByText("The denominator is nonzero.")).toBeVisible();
  expect(screen.getByText("Version 2")).toBeVisible();
  expect(screen.getAllByText("CC0-1.0").length).toBeGreaterThan(0);
  expect(screen.getByText("Original work by Math Master")).toBeVisible();
  rerender(
    <SafeMarkdown
      source={
        "$" +
        String.raw`\href{https://evil.test}{click}` +
        "$\n\n$" +
        String.raw`\def\x{\x}\x` +
        "$\n\n$" +
        "x".repeat(4097) +
        "$"
      }
      assets={[]}
    />,
  );
  expect(container.querySelector("a,img")).toBeNull();
  expect(
    screen.getAllByText(/Formula could not be displayed\./).length,
  ).toBeGreaterThan(0);
});
it("doesNotShareCustomMacrosOrRenderUnboundedSize", () => {
  const { container } = render(
    <SafeMarkdown
      source={String.raw`$\gdef\secret{hi}$ $\secret$ $\rule{500em}{500em}$`}
      assets={[]}
    />,
  );
  expect(container.querySelector("a,img,script")).toBeNull();
  expect(container.querySelector(".katex")).not.toBeNull();
  expect(container.querySelector('[style*="500em"]')).toBeNull();
});
