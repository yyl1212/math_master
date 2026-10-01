"use client";
import { useEffect, useRef, useState } from "react";
import type { VersionRef } from "@/lib/api/types";
import { refKey } from "./path-graph";
import styles from "@/styles/reading.module.css";
export function PathConnections({
  edges,
}: {
  edges: { from: VersionRef; to: VersionRef }[];
}) {
  const ref = useRef<SVGSVGElement>(null);
  const [drawing, setDrawing] = useState<{
    width: number;
    height: number;
    paths: string[];
  }>({ width: 1, height: 1, paths: [] });
  useEffect(() => {
    const parent = ref.current?.parentElement;
    if (!parent) return;
    const nodes = new Map(
      Array.from(parent.querySelectorAll<HTMLElement>("[data-node-key]")).map(
        (el) => [el.dataset.nodeKey, el],
      ),
    );
    const draw = () => {
      const box = parent.getBoundingClientRect();
      const paths = edges.flatMap((edge) => {
        const from = nodes.get(refKey(edge.from)),
          to = nodes.get(refKey(edge.to));
        if (!from || !to) return [];
        const a = from.getBoundingClientRect(),
          b = to.getBoundingClientRect(),
          x1 = a.left + a.width / 2 - box.left,
          y1 = a.bottom - box.top,
          x2 = b.left + b.width / 2 - box.left,
          y2 = b.top - box.top,
          mid = (y1 + y2) / 2;
        return [`M ${x1} ${y1} C ${x1} ${mid}, ${x2} ${mid}, ${x2} ${y2}`];
      });
      setDrawing({ width: box.width, height: box.height, paths });
    };
    const observer = new ResizeObserver(draw);
    observer.observe(parent);
    nodes.forEach((node) => observer.observe(node));
    draw();
    return () => observer.disconnect();
  }, [edges]);
  return (
    <svg
      ref={ref}
      className={styles.connections}
      viewBox={`0 0 ${drawing.width} ${drawing.height}`}
      aria-hidden="true"
    >
      {drawing.paths.map((d, i) => (
        <path key={i} d={d} fill="none" stroke="#9eaf91" strokeWidth="1.5" />
      ))}
    </svg>
  );
}
