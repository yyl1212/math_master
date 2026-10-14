import { it, expect,vi } from "vitest";
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
it('study公式在父组件重复渲染后仍可键盘滚动，切回default清理展示属性',()=>{
 vi.spyOn(Element.prototype,'clientWidth','get').mockReturnValue(100);
 vi.spyOn(Element.prototype,'scrollWidth','get').mockReturnValue(900);
 const source='$$\nx+1\n$$';const {rerender,container}=render(<SafeMarkdown presentation="study" source={source} assets={[]}/>);
 expect(container.querySelector('.katex-display')).toHaveAttribute('tabindex','0');
 rerender(<SafeMarkdown presentation="study" source={source} assets={[]}/>);
 expect(container.querySelector('.katex-display')).toHaveAttribute('tabindex','0');
 rerender(<SafeMarkdown source={source} assets={[]}/>);expect(container.querySelector('[role="region"]')).toBeNull();
});
it('study容器左右箭头局部滚动，修饰键保持原浏览器操作',()=>{
 vi.spyOn(Element.prototype,'clientWidth','get').mockReturnValue(100);vi.spyOn(Element.prototype,'scrollWidth','get').mockReturnValue(900);
 const {container}=render(<SafeMarkdown presentation="study" source={'$$\nx+1\n$$'} assets={[]}/>);const formula=container.querySelector<HTMLElement>('.katex-display')!;formula.focus();
 fireEvent.keyDown(formula,{key:'ArrowRight'});expect(formula.scrollLeft).toBeGreaterThan(0);const value=formula.scrollLeft;fireEvent.keyDown(formula,{key:'ArrowRight',ctrlKey:true});expect(formula.scrollLeft).toBe(value);fireEvent.keyDown(formula,{key:'ArrowLeft'});expect(formula.scrollLeft).toBe(0);
});
it.each(["default","study"] as const)("blocksHtmlUnsafeLinksAndUnboundImages %s", (presentation) => {
  const source =
    '<script>alert(1)</script>\n\n<iframe src="https://evil.test"></iframe>\n\n[bad](javascript:alert) [http](http://evil.test) [safe](https://example.org)\n\n![remote](https://evil.test/a.svg) ![missing](asset:unknown) ![Fractions](asset:fraction-bar)';
  const { container } = render(
    <SafeMarkdown presentation={presentation} source={source} assets={[asset]} />,
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
it.each(["default","study"] as const)("limitsMathAndKeepsConditionsSourcesAndAttribution %s", (presentation) => {
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
    <SafeMarkdown presentation={presentation}
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

it("TestPrivateBoundAssetPreviewAndRecovery",()=>{
 const id="11111111-1111-4111-8111-111111111111";
 const {rerender,container}=render(<SafeMarkdown source="![Private](asset:fraction-bar)" assets={[asset]} assetScope={{kind:"draft",id}}/>);
 const img=screen.getByRole("img",{name:"Private"});expect(img).toHaveAttribute("src",`/api/v1/content/drafts/${id}/assets/${asset.sha256}`);fireEvent.error(img);
 rerender(<SafeMarkdown source="![Private](asset:fraction-bar)" assets={[{...asset,sha256:"b".repeat(64)}]} assetScope={{kind:"submission",id}}/>);
 expect(screen.getByRole("img",{name:"Private"})).toHaveAttribute("src",`/api/v1/content/submissions/${id}/assets/${"b".repeat(64)}`);
 rerender(<SafeMarkdown source="![Private](asset:fraction-bar)" assets={[asset]} assetScope={{kind:"draft",id:"https://unsafe.example"}}/>);expect(screen.queryByRole("img")).not.toBeInTheDocument();
});
